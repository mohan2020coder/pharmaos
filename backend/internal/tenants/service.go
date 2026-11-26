package tenants

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type CreateRequest struct {
	Code        string         `json:"code" validate:"required,min=2,max=40"`
	Name        string         `json:"name" validate:"required,min=2,max=180"`
	TenantType  string         `json:"tenant_type" validate:"required,oneof=PHARMA_COMPANY DISTRIBUTOR"`
	CountryCode string         `json:"country_code" validate:"omitempty,len=2"`
	Timezone    string         `json:"timezone" validate:"omitempty,max=80"`
	Currency    string         `json:"currency_code" validate:"omitempty,len=3"`
	Settings    map[string]any `json:"settings"`
}

type UpdateRequest struct {
	Name     string         `json:"name" validate:"required,min=2,max=180"`
	Status   string         `json:"status" validate:"required,oneof=ACTIVE SUSPENDED INACTIVE"`
	Timezone string         `json:"timezone" validate:"required,max=80"`
	Currency string         `json:"currency_code" validate:"required,len=3"`
	Settings map[string]any `json:"settings"`
}

type AuditRecorder interface {
	Record(ctx context.Context, entry audit.Entry) error
}

type Service struct {
	repo  *Repository
	audit AuditRecorder
}

func NewService(repo *Repository, recorder AuditRecorder) *Service {
	return &Service{repo: repo, audit: recorder}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*Tenant, error) {
	if !principal.IsPrivileged {
		return nil, apperrors.ErrForbidden
	}
	tenant := &Tenant{
		ID:          uuid.New(),
		Code:        strings.ToUpper(strings.TrimSpace(request.Code)),
		Name:        strings.TrimSpace(request.Name),
		TenantType:  request.TenantType,
		Status:      "ACTIVE",
		CountryCode: emptyToNil(request.CountryCode),
		Timezone:    defaultString(request.Timezone, "UTC"),
		Currency:    defaultString(request.Currency, "USD"),
		Settings:    jsontypes.Map(request.Settings),
	}
	if tenant.Settings == nil {
		tenant.Settings = jsontypes.Map{}
	}
	if err := s.repo.Create(ctx, tenant); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, "TENANT_CREATED", tenant.ID.String(), nil, tenant)
	return tenant, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page) ([]Tenant, int64, error) {
	var ids []uuid.UUID
	if !principal.IsPrivileged {
		ids = []uuid.UUID{principal.TenantID}
	}
	return s.repo.List(ctx, page, ids, principal.IsPrivileged)
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Tenant, error) {
	if !principal.IsPrivileged && id != principal.TenantID {
		return nil, apperrors.ErrForbidden
	}
	tenant, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}
	return tenant, err
}

func (s *Service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, request UpdateRequest) (*Tenant, error) {
	if !principal.IsPrivileged && id != principal.TenantID {
		return nil, apperrors.ErrForbidden
	}
	tenant, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.ErrNotFound
	}
	previous := *tenant
	tenant.Name = request.Name
	tenant.Status = request.Status
	tenant.Timezone = request.Timezone
	tenant.Currency = request.Currency
	if request.Settings != nil {
		tenant.Settings = jsontypes.Map(request.Settings)
	}
	if err := s.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}
	s.record(ctx, "TENANT_UPDATED", id.String(), previous, tenant)
	return tenant, nil
}

func (s *Service) SetSetting(ctx context.Context, principal auth.Principal, id uuid.UUID, key string, value map[string]any) error {
	if !principal.IsPrivileged && id != principal.TenantID {
		return apperrors.ErrForbidden
	}
	return s.repo.UpsertSetting(ctx, id, key, value)
}

func (s *Service) record(ctx context.Context, action, entityID string, previous, next interface{}) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		Action: action, Module: "tenants", EntityType: "tenant", EntityID: entityID,
		Previous: previous, New: next,
	})
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
