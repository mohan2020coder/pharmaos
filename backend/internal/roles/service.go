package roles

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type CreateRequest struct {
	Code         string   `json:"code" validate:"required,min=2,max=40"`
	Name         string   `json:"name" validate:"required,min=2,max=100"`
	Description  string   `json:"description"`
	DataScope    string   `json:"data_scope" validate:"required,oneof=SELF TEAM TERRITORY REGION DIVISION TENANT"`
	IsPrivileged bool     `json:"is_privileged"`
	Permissions  []string `json:"permissions"`
}

type UpdateRequest struct {
	Name         string `json:"name" validate:"required,min=2,max=100"`
	Description  string `json:"description"`
	DataScope    string `json:"data_scope" validate:"required,oneof=SELF TEAM TERRITORY REGION DIVISION TENANT"`
	IsPrivileged bool   `json:"is_privileged"`
}

type Service struct {
	repo  *Repository
	audit *audit.Service
}

func NewService(repo *Repository, recorder *audit.Service) *Service {
	return &Service{repo: repo, audit: recorder}
}

func (s *Service) List(ctx context.Context, page query.Page) ([]Role, int64, error) {
	return s.repo.List(ctx, page)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*RoleWithPermissions, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	permissions, err := s.repo.Permissions(ctx, id)
	if err != nil {
		return nil, err
	}
	return &RoleWithPermissions{Role: *role, Permissions: permissions}, nil
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*RoleWithPermissions, error) {
	if request.IsPrivileged && !principal.IsPrivileged {
		return nil, apperrors.ErrForbidden
	}
	tenantID := principal.TenantID
	if principal.IsPrivileged {
		tenantID = uuid.Nil
	}
	role := &Role{
		ID: uuid.New(), TenantID: nullableTenant(tenantID), Code: strings.ToUpper(strings.TrimSpace(request.Code)),
		Name: request.Name, Description: request.Description, DataScope: request.DataScope,
		IsPrivileged: request.IsPrivileged,
	}
	if err := s.repo.Create(ctx, role); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	if err := s.setPermissions(ctx, role.ID, request.Permissions); err != nil {
		return nil, err
	}
	s.record(ctx, principal, "ROLE_CREATED", role.ID.String(), nil, role)
	return s.Get(ctx, role.ID)
}

func (s *Service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, request UpdateRequest) (*RoleWithPermissions, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	previous := *role
	role.Name = request.Name
	role.Description = request.Description
	role.DataScope = request.DataScope
	if principal.IsPrivileged {
		role.IsPrivileged = request.IsPrivileged
	}
	if err := s.repo.Update(ctx, role); err != nil {
		return nil, err
	}
	s.record(ctx, principal, "ROLE_UPDATED", id.String(), previous, role)
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperrors.ErrNotFound
	}
	if role.IsSystem {
		return apperrors.New("SYSTEM_ROLE_IMMUTABLE", "System roles cannot be deleted", 409)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.record(ctx, principal, "ROLE_DELETED", id.String(), role, nil)
	return nil
}

func (s *Service) SetPermissions(ctx context.Context, principal auth.Principal, id uuid.UUID, codes []string) (*RoleWithPermissions, error) {
	role, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	if role.IsPrivileged && !principal.IsPrivileged {
		return nil, apperrors.ErrForbidden
	}
	if err := s.setPermissions(ctx, id, codes); err != nil {
		return nil, err
	}
	s.record(ctx, principal, "ROLE_PERMISSIONS_UPDATED", id.String(), nil, codes)
	return s.Get(ctx, id)
}

func (s *Service) setPermissions(ctx context.Context, roleID uuid.UUID, codes []string) error {
	ids, err := s.repo.PermissionIDs(ctx, codes)
	if err != nil {
		return err
	}
	if len(ids) != len(codes) {
		return apperrors.New("UNKNOWN_PERMISSION", "One or more permissions do not exist", 422)
	}
	return s.repo.ReplacePermissions(ctx, roleID, ids)
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action, entityID string, previous, next interface{}) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID, Action: action,
		Module: "roles", EntityType: "role", EntityID: entityID, Previous: previous, New: next,
	})
}

func nullableTenant(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
