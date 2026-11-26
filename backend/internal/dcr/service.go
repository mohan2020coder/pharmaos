package dcr

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
	ID         *uuid.UUID `json:"id"`
	EmployeeID uuid.UUID  `json:"employee_id" validate:"required,uuid"`
	ReportDate string     `json:"report_date" validate:"required"`
}

type ProductRequest struct {
	ID          *uuid.UUID `json:"id"`
	ProductID   uuid.UUID  `json:"product_id" validate:"required,uuid"`
	Quantity    int        `json:"quantity"`
	SampleCount int        `json:"sample_count"`
	Outcome     string     `json:"outcome"`
	Remarks     string     `json:"remarks"`
}

type DecisionRequest struct {
	Remarks string `json:"remarks"`
}

type Service struct {
	reports  *crud.Repository[DailyCallReport]
	products *crud.Repository[ProductDetail]
	db       *gorm.DB
	audit    *audit.Service
}

func NewService(db *gorm.DB, auditService *audit.Service) *Service {
	return &Service{
		reports: crud.New[DailyCallReport](db), products: crud.New[ProductDetail](db),
		db: db, audit: auditService,
	}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest) (*DailyCallReport, error) {
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, request.EmployeeID); err != nil {
		return nil, err
	}
	reportDate, err := time.Parse("2006-01-02", request.ReportDate)
	if err != nil {
		return nil, apperrors.New("INVALID_DATE", "report_date must be YYYY-MM-DD", 422)
	}
	var existing DailyCallReport
	err = s.db.WithContext(ctx).
		Where("tenant_id = ? AND employee_id = ? AND report_date = ? AND deleted_at IS NULL",
			principal.TenantID, request.EmployeeID, reportDate).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	recordID := uuid.New()
	if request.ID != nil {
		recordID = *request.ID
	}
	report := &DailyCallReport{
		ID: recordID, TenantID: principal.TenantID, EmployeeID: request.EmployeeID,
		ReportDate: reportDate, Status: "DRAFT",
	}
	if err := s.reports.Create(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page, employeeID uuid.UUID, status, from, to string) ([]DailyCallReport, int64, error) {
	db := scope.Apply(s.db.WithContext(ctx), principal, "employee_id", "", "")
	db = db.Model(&DailyCallReport{}).Where("tenant_id = ?", principal.TenantID)
	if employeeID != uuid.Nil {
		db = db.Where("employee_id = ?", employeeID)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if from != "" {
		db = db.Where("report_date >= ?", from)
	}
	if to != "" {
		db = db.Where("report_date <= ?", to)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []DailyCallReport
	err := db.Order(page.Sort + " " + page.Order).Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*DailyCallReport, error) {
	report, err := s.reports.Get(ctx, principal.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, report.EmployeeID); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *Service) AddProduct(ctx context.Context, principal auth.Principal, dcrID uuid.UUID, request ProductRequest) (*ProductDetail, error) {
	report, err := s.Get(ctx, principal, dcrID)
	if err != nil {
		return nil, err
	}
	if report.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_STATE", "Only draft reports can be changed", 422)
	}
	var count int64
	if err := s.db.WithContext(ctx).Table("cat_products").
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", principal.TenantID, request.ProductID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, apperrors.New("INVALID_PRODUCT", "Product does not exist", 422)
	}
	detailID := uuid.New()
	if request.ID != nil {
		detailID = *request.ID
	}
	detail := &ProductDetail{
		ID: detailID, TenantID: principal.TenantID, DCRID: dcrID, ProductID: request.ProductID,
		Quantity: request.Quantity, SampleCount: request.SampleCount, Outcome: request.Outcome, Remarks: request.Remarks,
	}
	if err := s.products.Create(ctx, detail); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	return detail, nil
}

func (s *Service) Products(ctx context.Context, principal auth.Principal, dcrID uuid.UUID) ([]ProductDetail, error) {
	if _, err := s.Get(ctx, principal, dcrID); err != nil {
		return nil, err
	}
	var details []ProductDetail
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND dcr_id = ?", principal.TenantID, dcrID).
		Order("created_at ASC").Find(&details).Error
	return details, err
}

func (s *Service) Submit(ctx context.Context, principal auth.Principal, id uuid.UUID) (*DailyCallReport, error) {
	report, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if report.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_STATE", "Only draft reports can be submitted", 422)
	}
	details, err := s.Products(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return nil, apperrors.New("EMPTY_DCR", "Report must contain at least one product detail", 422)
	}
	var visitCount int64
	if err := s.db.WithContext(ctx).Model(&struct{}{}).Table("visits").
		Where("tenant_id = ? AND employee_id = ? AND visited_at >= ? AND visited_at < ? AND status IN ?",
			principal.TenantID, report.EmployeeID, report.ReportDate,
			report.ReportDate.AddDate(0, 0, 1), []string{"SUBMITTED", "APPROVED"}).
		Count(&visitCount).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.reports.Update(ctx, principal.TenantID, id, map[string]interface{}{
		"status": "SUBMITTED", "submitted_at": now, "total_visits": visitCount,
	}); err != nil {
		return nil, err
	}
	report.Status, report.SubmittedAt, report.TotalVisits = "SUBMITTED", &now, int(visitCount)
	s.record(ctx, principal, "DCR_SUBMIT", report)
	return report, nil
}

func (s *Service) Decide(ctx context.Context, principal auth.Principal, id uuid.UUID, approve bool, remarks string) (*DailyCallReport, error) {
	report, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if report.Status != "SUBMITTED" {
		return nil, apperrors.New("INVALID_STATE", "Only submitted reports can be reviewed", 422)
	}
	status := "REJECTED"
	if approve {
		status = "APPROVED"
	}
	now := time.Now().UTC()
	if err := s.reports.Update(ctx, principal.TenantID, id, map[string]interface{}{
		"status": status, "approved_by": principal.UserID, "approved_at": now, "remarks": remarks,
	}); err != nil {
		return nil, err
	}
	report.Status, report.ApprovedBy, report.ApprovedAt, report.Remarks = status, &principal.UserID, &now, remarks
	action := "DCR_REJECT"
	if approve {
		action = "DCR_APPROVE"
	}
	s.record(ctx, principal, action, report)
	return report, nil
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action string, report *DailyCallReport) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID, Action: action,
		Module: "dcr", EntityType: "daily_call_report", EntityID: report.ID.String(), New: report,
	})
}
