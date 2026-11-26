package crm

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Request struct {
	EntityType   string     `json:"entity_type" validate:"required,oneof=DOCTOR CHEMIST STOCKIST"`
	EntityID     uuid.UUID  `json:"entity_id" validate:"required,uuid"`
	EmployeeID   uuid.UUID  `json:"employee_id" validate:"required,uuid"`
	TerritoryID  *uuid.UUID `json:"territory_id"`
	PeriodStart  string     `json:"period_start" validate:"required"`
	PeriodEnd    string     `json:"period_end" validate:"required"`
	TargetVisits int        `json:"target_visits"`
	TargetSales  float64    `json:"target_sales"`
	Status       string     `json:"status" validate:"required,oneof=ACTIVE CLOSED CANCELLED"`
}

type Service struct {
	repo *crud.Repository[Target]
	db   *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{repo: crud.New[Target](db), db: db}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request Request) (*Target, error) {
	start, end, err := dates(request.PeriodStart, request.PeriodEnd)
	if err != nil {
		return nil, err
	}
	table := entityTable(request.EntityType)
	var count int64
	if err := s.db.WithContext(ctx).Table(table).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", principal.TenantID, request.EntityID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, apperrors.New("INVALID_TARGET_ENTITY", "Target entity does not exist", 422)
	}
	count = 0
	if err := s.db.WithContext(ctx).Table("employees").
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", principal.TenantID, request.EmployeeID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, apperrors.New("INVALID_EMPLOYEE", "Employee does not exist", 422)
	}
	status := request.Status
	if status == "" {
		status = "ACTIVE"
	}
	target := &Target{
		ID: uuid.New(), TenantID: principal.TenantID, EntityType: request.EntityType,
		EntityID: request.EntityID, EmployeeID: request.EmployeeID, TerritoryID: request.TerritoryID,
		PeriodStart: start, PeriodEnd: end, TargetVisits: request.TargetVisits,
		TargetSales: request.TargetSales, Status: status,
	}
	if err := s.repo.Create(ctx, target); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return target, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page, employeeID, territoryID uuid.UUID, status, entityType string) ([]Target, int64, error) {
	var filters []func(*gorm.DB) *gorm.DB
	if employeeID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("employee_id = ?", employeeID) })
	}
	if territoryID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("territory_id = ?", territoryID) })
	}
	if status != "" {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("status = ?", status) })
	}
	if entityType != "" {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("entity_type = ?", entityType) })
	}
	return s.repo.List(ctx, principal.TenantID, page, filters...)
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Target, error) {
	return s.repo.Get(ctx, principal.TenantID, id)
}

func (s *Service) Update(ctx context.Context, principal auth.Principal, id uuid.UUID, request Request) error {
	start, end, err := dates(request.PeriodStart, request.PeriodEnd)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, principal.TenantID, id, map[string]interface{}{
		"entity_type": request.EntityType, "entity_id": request.EntityID,
		"employee_id": request.EmployeeID, "territory_id": request.TerritoryID,
		"period_start": start, "period_end": end,
		"target_visits": request.TargetVisits, "target_sales": request.TargetSales,
		"status": request.Status,
	})
}

func (s *Service) Delete(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	return s.repo.Delete(ctx, principal.TenantID, id)
}

func dates(startValue, endValue string) (time.Time, time.Time, error) {
	start, err := time.Parse("2006-01-02", startValue)
	if err != nil {
		return time.Time{}, time.Time{}, apperrors.New("INVALID_DATE", "period_start must be YYYY-MM-DD", 422)
	}
	end, err := time.Parse("2006-01-02", endValue)
	if err != nil {
		return time.Time{}, time.Time{}, apperrors.New("INVALID_DATE", "period_end must be YYYY-MM-DD", 422)
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, apperrors.New("INVALID_DATE_RANGE", "period_end must not precede period_start", 422)
	}
	return start, end, nil
}

func entityTable(entityType string) string {
	switch entityType {
	case "DOCTOR":
		return "crm_doctors"
	case "CHEMIST":
		return "crm_chemists"
	case "STOCKIST":
		return "crm_stockists"
	default:
		return ""
	}
}
