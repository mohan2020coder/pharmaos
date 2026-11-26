package territories

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

type RegionRequest struct {
	DivisionID uuid.UUID `json:"division_id" validate:"required,uuid"`
	Code       string    `json:"code" validate:"required,min=2,max=40"`
	Name       string    `json:"name" validate:"required,min=2,max=150"`
	Status     string    `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
}

type TerritoryRequest struct {
	DivisionID uuid.UUID  `json:"division_id" validate:"required,uuid"`
	RegionID   uuid.UUID  `json:"region_id" validate:"required,uuid"`
	BranchID   *uuid.UUID `json:"branch_id"`
	Code       string     `json:"code" validate:"required,min=2,max=40"`
	Name       string     `json:"name" validate:"required,min=2,max=150"`
	City       string     `json:"city"`
	Status     string     `json:"status" validate:"required,oneof=ACTIVE INACTIVE"`
}

type Service struct {
	regions     *crud.Repository[Region]
	territories *crud.Repository[Territory]
	db          *gorm.DB
}

func NewService(db *gorm.DB, regions *crud.Repository[Region], territories *crud.Repository[Territory]) *Service {
	return &Service{db: db, regions: regions, territories: territories}
}

func (s *Service) validateRegion(ctx context.Context, tenantID, divisionID uuid.UUID) error {
	var count int64
	err := s.db.WithContext(ctx).Model(&Region{}).
		Where("tenant_id = ? AND id = ?", tenantID, divisionID).Count(&count).Error
	if err != nil || count == 0 {
		return apperrors.New("INVALID_DIVISION", "Division does not exist", 422)
	}
	return nil
}

func (s *Service) CreateRegion(ctx context.Context, principal auth.Principal, request RegionRequest) (*Region, error) {
	if err := s.validateRegion(ctx, principal.TenantID, request.DivisionID); err != nil {
		return nil, err
	}
	region := &Region{
		ID: uuid.New(), TenantID: principal.TenantID, DivisionID: request.DivisionID,
		Code: strings.ToUpper(strings.TrimSpace(request.Code)), Name: strings.TrimSpace(request.Name), Status: request.Status,
	}
	if err := s.regions.Create(ctx, region); err != nil {
		return nil, conflict(err)
	}
	return region, nil
}

func (s *Service) ListRegions(ctx context.Context, principal auth.Principal, page query.Page, divisionID uuid.UUID) ([]Region, int64, error) {
	var filters []func(*gorm.DB) *gorm.DB
	if divisionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("division_id = ?", divisionID) })
	}
	return s.regions.List(ctx, principal.TenantID, page, filters...)
}

func (s *Service) CreateTerritory(ctx context.Context, principal auth.Principal, request TerritoryRequest) (*Territory, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&Region{}).
		Where("tenant_id = ? AND id = ? AND division_id = ?", principal.TenantID, request.RegionID, request.DivisionID).
		Count(&count).Error; err != nil || count == 0 {
		return nil, apperrors.New("INVALID_REGION", "Region does not belong to the division", 422)
	}
	territory := &Territory{
		ID: uuid.New(), TenantID: principal.TenantID, DivisionID: request.DivisionID, RegionID: request.RegionID,
		BranchID: request.BranchID, Code: strings.ToUpper(strings.TrimSpace(request.Code)),
		Name: strings.TrimSpace(request.Name), City: request.City, Status: request.Status,
	}
	if err := s.territories.Create(ctx, territory); err != nil {
		return nil, conflict(err)
	}
	return territory, nil
}

func (s *Service) ListTerritories(ctx context.Context, principal auth.Principal, page query.Page, divisionID, regionID uuid.UUID) ([]Territory, int64, error) {
	var filters []func(*gorm.DB) *gorm.DB
	if regionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("region_id = ?", regionID) })
	}
	if divisionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("division_id = ?", divisionID) })
	}
	return s.territories.List(ctx, principal.TenantID, page, filters...)
}

func conflict(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return apperrors.ErrConflict
	}
	return err
}
