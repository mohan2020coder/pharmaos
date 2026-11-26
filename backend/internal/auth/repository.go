package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

func (r *Repository) FindUserByEmail(ctx context.Context, db *gorm.DB, tenantID uuid.UUID, email string) (*AuthUser, error) {
	var user AuthUser
	err := db.WithContext(ctx).Where("tenant_id = ? AND email = ? AND deleted_at IS NULL", tenantID, email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserByID(ctx context.Context, db *gorm.DB, id uuid.UUID) (*AuthUser, error) {
	var user AuthUser
	if err := db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindUserForPasswordReset(ctx context.Context, db *gorm.DB, email string) (*AuthUser, error) {
	var user AuthUser
	if err := db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL AND status = 'ACTIVE'", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) RecordLoginResult(ctx context.Context, db *gorm.DB, userID uuid.UUID, success bool, lockUntil *time.Time) error {
	if success {
		return db.WithContext(ctx).Exec(`UPDATE users SET failed_login_attempts = 0, locked_until = NULL, last_login_at = now() WHERE id = ?`, userID).Error
	}
	return db.WithContext(ctx).Exec(`UPDATE users SET failed_login_attempts = failed_login_attempts + 1, locked_until = ? WHERE id = ?`, lockUntil, userID).Error
}

func (r *Repository) UpdatePassword(ctx context.Context, db *gorm.DB, userID uuid.UUID, hash string) error {
	return db.WithContext(ctx).Exec(`UPDATE users SET password_hash = ?, must_change_password = false WHERE id = ?`, hash, userID).Error
}

func (r *Repository) AccessContext(ctx context.Context, db *gorm.DB, userID uuid.UUID) (Principal, error) {
	var grants []UserRoleGrant
	err := db.WithContext(ctx).Raw(`SELECT ur.user_id, ur.role_id, ur.scope_type, ur.scope_id, ur.valid_from, ur.valid_to,
		r.code, r.data_scope, r.is_privileged, r.tenant_id
		FROM user_roles ur JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = ? AND ur.valid_from <= CURRENT_DATE AND (ur.valid_to IS NULL OR ur.valid_to >= CURRENT_DATE)
		  AND r.deleted_at IS NULL`, userID).Scan(&grants).Error
	if err != nil {
		return Principal{}, err
	}
	if len(grants) == 0 {
		return Principal{}, gorm.ErrRecordNotFound
	}
	principal := Principal{
		UserID: userID, Roles: make([]string, 0, len(grants)), Permissions: map[string]bool{}, DataScope: "SELF",
	}
	for _, grant := range grants {
		principal.Roles = append(principal.Roles, grant.Code)
		if grant.ScopeID != nil && *grant.ScopeID != uuid.Nil {
			switch grant.ScopeType {
			case "DIVISION":
				principal.DivisionIDs = append(principal.DivisionIDs, *grant.ScopeID)
			case "TERRITORY", "REGION":
				principal.TerritoryIDs = append(principal.TerritoryIDs, *grant.ScopeID)
			}
		}
		if grant.IsPrivileged {
			principal.IsPrivileged = true
		}
		scopeRank := map[string]int{"SELF": 1, "TEAM": 2, "TERRITORY": 3, "REGION": 4, "DIVISION": 5, "TENANT": 6}
		if scopeRank[grant.DataScope] > scopeRank[principal.DataScope] {
			principal.DataScope = grant.DataScope
		}
	}
	var codes []string
	if err := db.WithContext(ctx).Raw(`SELECT DISTINCT p.code FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ? AND ur.valid_from <= CURRENT_DATE AND (ur.valid_to IS NULL OR ur.valid_to >= CURRENT_DATE)`, userID).Scan(&codes).Error; err != nil {
		return Principal{}, err
	}
	for _, code := range codes {
		principal.Permissions[code] = true
	}
	return principal, nil
}

func (r *Repository) CreateSession(ctx context.Context, db *gorm.DB, session *Session) error {
	return db.WithContext(ctx).Create(session).Error
}

func (r *Repository) FindSession(ctx context.Context, db *gorm.DB, id uuid.UUID) (*Session, error) {
	var session Session
	if err := db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *Repository) FindSessionByHash(ctx context.Context, db *gorm.DB, hash string) (*Session, error) {
	var session Session
	if err := db.WithContext(ctx).Where("refresh_token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *Repository) RevokeSession(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return db.WithContext(ctx).Exec(`UPDATE auth_sessions SET revoked_at = now() WHERE id = ? AND revoked_at IS NULL`, id).Error
}

func (r *Repository) RevokeFamily(ctx context.Context, db *gorm.DB, familyID uuid.UUID) error {
	return db.WithContext(ctx).Exec(`UPDATE auth_sessions SET revoked_at = now() WHERE family_id = ? AND revoked_at IS NULL`, familyID).Error
}

func (r *Repository) ListSessions(ctx context.Context, db *gorm.DB, userID uuid.UUID) ([]Session, error) {
	var sessions []Session
	err := db.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL AND expires_at > now()", userID).
		Order("created_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *Repository) CreatePasswordReset(ctx context.Context, db *gorm.DB, reset *PasswordReset) error {
	return db.WithContext(ctx).Create(reset).Error
}

func (r *Repository) FindPasswordReset(ctx context.Context, db *gorm.DB, hash string) (*PasswordReset, error) {
	var reset PasswordReset
	if err := db.WithContext(ctx).Where("token_hash = ? AND used_at IS NULL AND expires_at > now()", hash).First(&reset).Error; err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *Repository) MarkResetUsed(ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	return db.WithContext(ctx).Exec(`UPDATE password_reset_tokens SET used_at = now() WHERE id = ?`, id).Error
}
