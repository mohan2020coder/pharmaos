package users

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type CreateRequest struct {
	TenantID   uuid.UUID `json:"tenant_id"`
	Email      string    `json:"email" validate:"required,email"`
	Password   string    `json:"password" validate:"required,min=10"`
	FirstName  string    `json:"first_name" validate:"required,min=2,max=80"`
	LastName   string    `json:"last_name" validate:"required,min=2,max=80"`
	Phone      string    `json:"phone"`
	MustChange bool      `json:"must_change_password"`
}

type UpdateRequest struct {
	FirstName     string `json:"first_name" validate:"required,min=2,max=80"`
	LastName      string `json:"last_name" validate:"required,min=2,max=80"`
	Phone         string `json:"phone"`
	Status        string `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
	EmailVerified bool   `json:"email_verified"`
}

type Service struct {
	repo     *Repository
	audit    *audit.Service
	hashCost int
}

func NewService(repo *Repository, recorder *audit.Service, hashCost int) *Service {
	return &Service{repo: repo, audit: recorder, hashCost: hashCost}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*User, error) {
	tenantID := principal.TenantID
	if request.TenantID != uuid.Nil {
		if !principal.IsPrivileged {
			return nil, apperrors.ErrForbidden
		}
		tenantID = request.TenantID
	}
	if tenantID == uuid.Nil {
		return nil, apperrors.ErrBadRequest
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(request.Password), s.hashCost)
	if err != nil {
		return nil, err
	}
	phone := nullable(request.Phone)
	user := &User{
		ID: uuid.New(), TenantID: tenantID, Email: strings.ToLower(strings.TrimSpace(request.Email)),
		PasswordHash: string(hash), FirstName: strings.TrimSpace(request.FirstName), LastName: strings.TrimSpace(request.LastName),
		Phone: phone, Status: "ACTIVE", MustChangePassword: request.MustChange,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "USER_CREATED", user.ID.String(), nil, user)
	return user, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page) ([]User, int64, error) {
	return s.repo.List(ctx, page, principal.TenantID, principal.IsPrivileged)
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*User, error) {
	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	if !principal.IsPrivileged && user.TenantID != principal.TenantID {
		return nil, apperrors.ErrTenantMismatch
	}
	return user, nil
}

func (s *Service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, request UpdateRequest) (*User, error) {
	user, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	previous := *user
	user.FirstName = request.FirstName
	user.LastName = request.LastName
	user.Phone = nullable(request.Phone)
	user.Status = request.Status
	user.EmailVerified = request.EmailVerified
	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}
	s.record(ctx, principal, "USER_UPDATED", id.String(), previous, user)
	return user, nil
}

func (s *Service) Deactivate(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	user, err := s.Get(ctx, principal, id)
	if err != nil {
		return err
	}
	if user.ID == principal.UserID {
		return apperrors.New("CANNOT_DEACTIVATE_SELF", "You cannot deactivate your own account", 400)
	}
	if err := s.repo.Deactivate(ctx, id); err != nil {
		return err
	}
	s.record(ctx, principal, "USER_DEACTIVATED", id.String(), user, nil)
	return nil
}

func (s *Service) AssignRole(ctx context.Context, principal auth.Principal, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID, validFrom time.Time, validTo *time.Time) error {
	user, err := s.Get(ctx, principal, userID)
	if err != nil {
		return err
	}
	valid, err := s.repo.RoleBelongsToTenant(ctx, roleID, user.TenantID)
	if err != nil {
		return err
	}
	if !valid {
		return apperrors.ErrBadRequest
	}
	if validFrom.IsZero() {
		validFrom = time.Now().UTC()
	}
	grant := RoleAssignment{UserID: userID, RoleID: roleID, ScopeType: scopeType, ScopeID: scopeID, ValidFrom: validFrom, ValidTo: validTo}
	if err := s.repo.AssignRole(ctx, grant); err != nil {
		return err
	}
	s.record(ctx, principal, "USER_ROLE_ASSIGNED", userID.String(), nil, grant)
	return nil
}

func (s *Service) RemoveRole(ctx context.Context, principal auth.Principal, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID) error {
	if _, err := s.Get(ctx, principal, userID); err != nil {
		return err
	}
	if err := s.repo.RemoveRole(ctx, userID, roleID, scopeType, scopeID); err != nil {
		return err
	}
	s.record(ctx, principal, "USER_ROLE_REMOVED", userID.String(), nil, map[string]interface{}{"role_id": roleID})
	return nil
}

func (s *Service) RolesForUser(ctx context.Context, principal auth.Principal, userID uuid.UUID) ([]RoleAssignment, error) {
	if _, err := s.Get(ctx, principal, userID); err != nil {
		return nil, err
	}
	return s.repo.RolesForUser(ctx, userID)
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action, entityID string, previous, next interface{}) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID, Action: action,
		Module: "users", EntityType: "user", EntityID: entityID, Previous: previous, New: next,
	})
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
