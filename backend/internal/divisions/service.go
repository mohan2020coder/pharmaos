package divisions

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Request struct {
	Code          string        `json:"code" validate:"required,min=2,max=40"`
	Name          string        `json:"name" validate:"required,min=2,max=150"`
	Description   string        `json:"description"`
	Status        string        `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
	ManagerUserID *uuid.UUID    `json:"manager_user_id"`
	Settings      jsontypes.Map `json:"settings"`
}

type Service struct {
	repo *crud.Repository[Division]
}

func NewService(repo *crud.Repository[Division]) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, principal auth.Principal, request Request) (*Division, error) {
	division := &Division{
		ID: uuid.New(), TenantID: principal.TenantID, Code: strings.ToUpper(strings.TrimSpace(request.Code)),
		Name: strings.TrimSpace(request.Name), Description: request.Description, Status: request.Status,
		ManagerUserID: request.ManagerUserID, Settings: request.Settings,
	}
	if division.Settings == nil {
		division.Settings = jsontypes.Map{}
	}
	if err := s.repo.Create(ctx, division); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return division, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page) ([]Division, int64, error) {
	var filters []func(*gorm.DB) *gorm.DB
	if page.Search != "" {
		term := "%" + page.Search + "%"
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("name ILIKE ? OR code ILIKE ?", term, term) })
	}
	return s.repo.List(ctx, principal.TenantID, page, filters...)
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Division, error) {
	return s.repo.Get(ctx, principal.TenantID, id)
}

func (s *Service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, request Request) error {
	return s.repo.Update(ctx, principal.TenantID, id, map[string]interface{}{
		"name": request.Name, "description": request.Description, "status": request.Status,
		"manager_user_id": request.ManagerUserID, "settings": request.Settings,
	})
}

func (s *Service) Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	return s.repo.Delete(ctx, principal.TenantID, id)
}
