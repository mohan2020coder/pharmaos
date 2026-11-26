package tours

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
	"github.com/pharmaos/pharmaos/backend/internal/platform/scope"
)

type CreateRequest struct {
	ID          *uuid.UUID `json:"id"`
	EmployeeID  uuid.UUID  `json:"employee_id" validate:"required,uuid"`
	Title       string     `json:"title" validate:"required,min=2,max=150"`
	PeriodStart string     `json:"period_start" validate:"required"`
	PeriodEnd   string     `json:"period_end" validate:"required"`
}

type LineRequest struct {
	ID          *uuid.UUID `json:"id"`
	PlannedDate string     `json:"planned_date" validate:"required"`
	TerritoryID *uuid.UUID `json:"territory_id"`
	EntityType  string     `json:"entity_type" validate:"required,oneof=DOCTOR CHEMIST STOCKIST"`
	EntityID    uuid.UUID  `json:"entity_id" validate:"required,uuid"`
	Purpose     string     `json:"purpose"`
	Sequence    int        `json:"sequence" validate:"required,min=1"`
}

type DecisionRequest struct {
	Comments string `json:"comments"`
}

type Service struct {
	plans *crud.Repository[TourPlan]
	lines *crud.Repository[TourPlanLine]
	db    *gorm.DB
	audit *audit.Service
}

func NewService(db *gorm.DB, auditService *audit.Service) *Service {
	return &Service{
		plans: crud.New[TourPlan](db), lines: crud.New[TourPlanLine](db),
		db: db, audit: auditService,
	}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*TourPlan, error) {
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, request.EmployeeID); err != nil {
		return nil, err
	}
	start, end, err := dates(request.PeriodStart, request.PeriodEnd)
	if err != nil {
		return nil, err
	}
	recordID := uuid.New()
	if request.ID != nil {
		recordID = *request.ID
	}
	plan := &TourPlan{
		ID: recordID, TenantID: principal.TenantID, EmployeeID: request.EmployeeID,
		Title: strings.TrimSpace(request.Title), PeriodStart: start, PeriodEnd: end, Status: "DRAFT",
	}
	if err := s.plans.Create(ctx, plan); err != nil {
		return nil, err
	}
	s.record(ctx, principal, "TOUR_PLAN_CREATE", plan)
	return plan, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page, employeeID uuid.UUID, status string) ([]TourPlan, int64, error) {
	db := scope.Apply(s.db.WithContext(ctx), principal, "employee_id", "", "")
	db = db.Model(&TourPlan{}).Where("tenant_id = ?", principal.TenantID)
	if employeeID != uuid.Nil {
		db = db.Where("employee_id = ?", employeeID)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []TourPlan
	err := db.Order(page.Sort + " " + page.Order).Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*TourPlan, error) {
	plan, err := s.plans.Get(ctx, principal.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, plan.EmployeeID); err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *Service) AddLine(ctx context.Context, principal auth.Principal, planID uuid.UUID, request LineRequest) (*TourPlanLine, error) {
	plan, err := s.Get(ctx, principal, planID)
	if err != nil {
		return nil, err
	}
	if plan.Status != "DRAFT" {
		return nil, apperrors.New("PLAN_NOT_EDITABLE", "Only draft tour plans can be changed", 422)
	}
	plannedDate, err := time.Parse("2006-01-02", request.PlannedDate)
	if err != nil {
		return nil, apperrors.New("INVALID_DATE", "planned_date must be YYYY-MM-DD", 422)
	}
	if plannedDate.Before(plan.PeriodStart) || plannedDate.After(plan.PeriodEnd) {
		return nil, apperrors.New("DATE_OUT_OF_RANGE", "planned_date must fall within the plan period", 422)
	}
	table := entityTable(request.EntityType)
	var count int64
	if err := s.db.WithContext(ctx).Table(table).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", principal.TenantID, request.EntityID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, apperrors.New("INVALID_ENTITY", "Plan entity does not exist", 422)
	}
	lineID := uuid.New()
	if request.ID != nil {
		lineID = *request.ID
	}
	line := &TourPlanLine{
		ID: lineID, TenantID: principal.TenantID, TourPlanID: planID, PlannedDate: plannedDate,
		TerritoryID: request.TerritoryID, EntityType: request.EntityType, EntityID: request.EntityID,
		Purpose: request.Purpose, Sequence: request.Sequence, Status: "PLANNED",
	}
	if err := s.lines.Create(ctx, line); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return line, nil
}

func (s *Service) Lines(ctx context.Context, principal auth.Principal, planID uuid.UUID) ([]TourPlanLine, error) {
	if _, err := s.Get(ctx, principal, planID); err != nil {
		return nil, err
	}
	var lines []TourPlanLine
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND tour_plan_id = ?", principal.TenantID, planID).
		Order("planned_date ASC, sequence ASC").Find(&lines).Error
	return lines, err
}

func (s *Service) Submit(ctx context.Context, principal auth.Principal, id uuid.UUID) (*TourPlan, error) {
	plan, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if plan.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_STATE", "Only draft plans can be submitted", 422)
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&TourPlanLine{}).Where("tour_plan_id = ?", id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, apperrors.New("EMPTY_PLAN", "Plan must contain at least one line", 422)
	}
	now := time.Now().UTC()
	if err := s.plans.Update(ctx, principal.TenantID, id, map[string]interface{}{"status": "SUBMITTED", "submitted_at": now}); err != nil {
		return nil, err
	}
	plan.Status, plan.SubmittedAt = "SUBMITTED", &now
	s.record(ctx, principal, "TOUR_PLAN_SUBMIT", plan)
	return plan, nil
}

func (s *Service) Decide(ctx context.Context, principal auth.Principal, id uuid.UUID, approve bool, comments string) (*TourPlan, error) {
	plan, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if plan.Status != "SUBMITTED" {
		return nil, apperrors.New("INVALID_STATE", "Only submitted plans can be approved or rejected", 422)
	}
	status := "REJECTED"
	if approve {
		status = "APPROVED"
	}
	now := time.Now().UTC()
	values := map[string]interface{}{
		"status": status, "approved_by": principal.UserID, "approved_at": now, "comments": comments,
	}
	if err := s.plans.Update(ctx, principal.TenantID, id, values); err != nil {
		return nil, err
	}
	plan.Status, plan.ApprovedBy, plan.ApprovedAt, plan.Comments = status, &principal.UserID, &now, comments
	action := "TOUR_PLAN_REJECT"
	if approve {
		action = "TOUR_PLAN_APPROVE"
	}
	s.record(ctx, principal, action, plan)
	return plan, nil
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action string, plan *TourPlan) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID, Action: action,
		Module: "tours", EntityType: "tour_plan", EntityID: plan.ID.String(), New: plan,
	})
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
