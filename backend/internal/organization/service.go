package organization

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type BranchRequest struct {
	DivisionID  *uuid.UUID `json:"division_id"`
	Code        string     `json:"code" validate:"required,min=2,max=40"`
	Name        string     `json:"name" validate:"required,min=2,max=150"`
	Address     string     `json:"address"`
	City        string     `json:"city"`
	State       string     `json:"state"`
	PostalCode  string     `json:"postal_code"`
	CountryCode string     `json:"country_code" validate:"omitempty,len=2"`
	Status      string     `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
}

type Service struct {
	branches *crud.Repository[Branch]
	db       *gorm.DB
}

func NewService(repo *crud.Repository[Branch], db *gorm.DB) *Service {
	return &Service{branches: repo, db: db}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request BranchRequest) (*Branch, error) {
	var country *string
	if request.CountryCode != "" {
		country = &request.CountryCode
	}
	branch := &Branch{
		ID: uuid.New(), TenantID: principal.TenantID, DivisionID: request.DivisionID,
		Code: strings.ToUpper(strings.TrimSpace(request.Code)), Name: strings.TrimSpace(request.Name),
		Address: request.Address, City: request.City, State: request.State, PostalCode: request.PostalCode,
		CountryCode: country, Status: request.Status,
	}
	if err := s.branches.Create(ctx, branch); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return branch, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page) ([]Branch, int64, error) {
	return s.branches.List(ctx, principal.TenantID, page)
}

func (s *Service) Hierarchy(ctx context.Context, principal auth.Principal) (map[string]interface{}, error) {
	var divisions []struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}
	if err := s.db.WithContext(ctx).Raw(`SELECT id, name FROM business_divisions WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY name`, principal.TenantID).Scan(&divisions).Error; err != nil {
		return nil, err
	}
	var regions []struct {
		ID         uuid.UUID `json:"id"`
		DivisionID uuid.UUID `json:"division_id"`
		Name       string    `json:"name"`
	}
	if err := s.db.WithContext(ctx).Raw(`SELECT id, division_id, name FROM regions WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY name`, principal.TenantID).Scan(&regions).Error; err != nil {
		return nil, err
	}
	var territories []struct {
		ID       uuid.UUID `json:"id"`
		RegionID uuid.UUID `json:"region_id"`
		Name     string    `json:"name"`
	}
	if err := s.db.WithContext(ctx).Raw(`SELECT id, region_id, name FROM territories WHERE tenant_id = ? AND deleted_at IS NULL ORDER BY name`, principal.TenantID).Scan(&territories).Error; err != nil {
		return nil, err
	}
	return map[string]interface{}{"divisions": divisions, "regions": regions, "territories": territories}, nil
}
