package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
)

type LoginRequest struct {
	TenantCode string `json:"tenant_code" validate:"required,min=2,max=40"`
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=6"`
	DeviceID   string `json:"device_id"`
}

type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
}

type PasswordResetSender interface {
	SendPasswordReset(ctx context.Context, email, token string) error
}

type Service struct {
	db         *gorm.DB
	repo       *Repository
	tokens     *Manager
	audit      AuditRecorder
	hashCost   int
	maxAttempt int
	lockTime   time.Duration
	sender     PasswordResetSender
}

type AuditRecorder interface {
	Record(ctx context.Context, entry audit.Entry) error
}

func NewService(db *gorm.DB, repo *Repository, tokens *Manager, audit AuditRecorder, hashCost int, sender PasswordResetSender) *Service {
	return &Service{db: db, repo: repo, tokens: tokens, audit: audit, hashCost: hashCost,
		maxAttempt: 5, lockTime: 15 * time.Minute, sender: sender}
}

func (s *Service) Login(ctx context.Context, request LoginRequest, ip, userAgent string) (*TokenPair, *Principal, error) {
	tx, err := beginAppTx(ctx, s.db)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	var tenantID uuid.UUID
	var tenantStatus string
	if err := tx.Raw(`SELECT id, status FROM tenants WHERE code = ? AND deleted_at IS NULL`, strings.ToUpper(request.TenantCode)).
		Row().Scan(&tenantID, &tenantStatus); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, apperrors.ErrUnauthorized
		}
		return nil, nil, err
	}
	if tenantStatus != "ACTIVE" {
		return nil, nil, apperrors.ErrForbidden
	}
	if err := tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantID.String()).Error; err != nil {
		return nil, nil, err
	}
	user, err := s.repo.FindUserByEmail(ctx, tx, tenantID, strings.TrimSpace(request.Email))
	if err != nil {
		return nil, nil, apperrors.ErrUnauthorized
	}
	if user.Status != "ACTIVE" || user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, nil, apperrors.ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		lockUntil := (*time.Time)(nil)
		if user.FailedLoginAttempts+1 >= s.maxAttempt {
			value := time.Now().Add(s.lockTime)
			lockUntil = &value
		}
		_ = s.repo.RecordLoginResult(ctx, tx, user.ID, false, lockUntil)
		return nil, nil, apperrors.ErrUnauthorized
	}
	_ = s.repo.RecordLoginResult(ctx, tx, user.ID, true, nil)

	principal, err := s.repo.AccessContext(ctx, tx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	principal.TenantID = tenantID
	principal.Email = user.Email
	principal.SessionID = uuid.New()
	pair, err := s.issueTokens(ctx, tx, principal, uuid.New(), ip, userAgent, request.DeviceID)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, nil, err
	}
	s.record(ctx, user.ID, tenantID, "LOGIN", "auth", "user", user.ID.String(), ip, userAgent)
	return pair, &principal, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken, ip, userAgent string) (*TokenPair, *Principal, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil, nil, apperrors.ErrUnauthorized
	}
	tx, err := beginAppTx(ctx, s.db)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, nil, apperrors.ErrUnauthorized
	}
	if err := tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantID.String()).Error; err != nil {
		return nil, nil, err
	}
	hash := hashToken(refreshToken)
	session, err := s.repo.FindSessionByHash(ctx, tx, hash)
	if err != nil {
		return nil, nil, apperrors.ErrUnauthorized
	}
	if session.TenantID != tenantID {
		return nil, nil, apperrors.ErrUnauthorized
	}
	if session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) || session.ID.String() != claims.SessionID {
		_ = s.repo.RevokeFamily(ctx, tx, session.FamilyID)
		return nil, nil, apperrors.ErrUnauthorized
	}
	user, err := s.repo.FindUserByID(ctx, tx, session.UserID)
	if err != nil || user.Status != "ACTIVE" {
		return nil, nil, apperrors.ErrUnauthorized
	}
	principal, err := s.repo.AccessContext(ctx, tx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	principal.TenantID = tenantID
	principal.Email = user.Email
	principal.SessionID = uuid.New()
	pair, err := s.issueTokens(ctx, tx, principal, session.FamilyID, ip, userAgent, "")
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.RevokeSession(ctx, tx, session.ID); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, nil, err
	}
	return pair, &principal, nil
}

func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.repo.RevokeSession(ctx, s.db.WithContext(ctx), sessionID)
}

func (s *Service) Sessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.repo.ListSessions(ctx, s.db.WithContext(ctx), userID)
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	tx, err := beginAppTx(ctx, s.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error; err != nil {
		return err
	}
	user, err := s.repo.FindUserForPasswordReset(ctx, tx, email)
	if err != nil {
		return nil
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	if err := s.repo.CreatePasswordReset(ctx, tx, &PasswordReset{
		ID: uuid.New(), UserID: user.ID, TokenHash: hashToken(token), ExpiresAt: time.Now().Add(30 * time.Minute),
	}); err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	if s.sender != nil {
		_ = s.sender.SendPasswordReset(ctx, string(user.Email), token)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < 10 {
		return apperrors.ErrValidation
	}
	tx, err := beginAppTx(ctx, s.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error; err != nil {
		return err
	}
	reset, err := s.repo.FindPasswordReset(ctx, tx, hashToken(token))
	if err != nil {
		return apperrors.ErrBadRequest
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.hashCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePassword(ctx, tx, reset.UserID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.MarkResetUsed(ctx, tx, reset.ID); err != nil {
		return err
	}
	if err := tx.Exec(`UPDATE auth_sessions SET revoked_at = now() WHERE user_id = ? AND revoked_at IS NULL`, reset.UserID).Error; err != nil {
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, current, next string) error {
	user, err := s.repo.FindUserByID(ctx, s.db.WithContext(ctx), userID)
	if err != nil {
		return apperrors.ErrNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)); err != nil {
		return apperrors.New("INVALID_PASSWORD", "Current password is incorrect", 400)
	}
	if len(next) < 10 {
		return apperrors.New("WEAK_PASSWORD", "New password must be at least 10 characters", 422)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), s.hashCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, s.db.WithContext(ctx), userID, string(hash))
}

func (s *Service) issueTokens(ctx context.Context, tx *gorm.DB, principal Principal, familyID uuid.UUID, ip, userAgent, deviceID string) (*TokenPair, error) {
	access, _, err := s.tokens.GenerateAccess(principal, principal.SessionID)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.tokens.GenerateRefresh(principal.UserID, principal.TenantID, principal.SessionID, familyID)
	if err != nil {
		return nil, err
	}
	session := &Session{
		ID: principal.SessionID, UserID: principal.UserID, TenantID: principal.TenantID, FamilyID: familyID,
		RefreshTokenHash: hashToken(refresh), IPAddress: nullableString(ip), UserAgent: nullableString(userAgent),
		DeviceID: nullableString(deviceID), ExpiresAt: time.Now().Add(s.tokens.RefreshTTL()),
	}
	if err := s.repo.CreateSession(ctx, tx, session); err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken: access, RefreshToken: refresh, TokenType: "Bearer",
		ExpiresIn: int(s.tokens.accessTTL.Seconds()), RefreshExpiresIn: int(s.tokens.RefreshTTL().Seconds()),
	}, nil
}

func (s *Service) record(ctx context.Context, userID, tenantID uuid.UUID, action, module, entityType, entityID, ip, userAgent string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &tenantID, UserID: &userID, Action: action, Module: module, EntityType: entityType,
		EntityID: entityID, IPAddress: ip, UserAgent: userAgent,
	})
}

func beginAppTx(ctx context.Context, db *gorm.DB) (*gorm.DB, error) {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Exec(`SET LOCAL ROLE pharmaos_app`).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
