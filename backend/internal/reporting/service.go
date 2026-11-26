package reporting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Service struct {
	db    *gorm.DB
	audit *audit.Service
}

func NewService(db *gorm.DB, auditService *audit.Service) *Service {
	return &Service{db: db, audit: auditService}
}

func (s *Service) FieldActivity(ctx context.Context, principal auth.Principal, from, to string) (*FieldActivityReport, error) {
	start, end, fromLabel, toLabel, err := periodBounds(from, to)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)

	visitBase := func() *gorm.DB {
		return scopeQuery(db.Table("visits").Where("tenant_id = ? AND visited_at BETWEEN ? AND ?", principal.TenantID, start, end), principal, "visits")
	}
	var total, submitted, draft, uniqueCustomers int64
	if err := visitBase().Count(&total).Error; err != nil {
		return nil, err
	}
	if err := visitBase().Where("status = 'SUBMITTED'").Count(&submitted).Error; err != nil {
		return nil, err
	}
	if err := visitBase().Where("status = 'DRAFT'").Count(&draft).Error; err != nil {
		return nil, err
	}
	if err := visitBase().Distinct("entity_id").Count(&uniqueCustomers).Error; err != nil {
		return nil, err
	}

	dcrBase := func() *gorm.DB {
		return scopeQuery(db.Table("daily_call_reports").Where("tenant_id = ? AND report_date BETWEEN ? AND ?", principal.TenantID, start, end), principal, "daily_call_reports")
	}
	var dcrTotal, dcrSubmitted, dcrApproved int64
	if err := dcrBase().Count(&dcrTotal).Error; err != nil {
		return nil, err
	}
	if err := dcrBase().Where("status = 'SUBMITTED'").Count(&dcrSubmitted).Error; err != nil {
		return nil, err
	}
	if err := dcrBase().Where("status = 'APPROVED'").Count(&dcrApproved).Error; err != nil {
		return nil, err
	}

	return &FieldActivityReport{
		Period: Period{From: fromLabel, To: toLabel},
		Scope:  dataScope(principal),
		Visits: VisitMetrics{Total: total, Submitted: submitted, Draft: draft, UniqueCustomers: uniqueCustomers},
		DailyCallReports: DailyCallReportMetrics{
			Total: dcrTotal, Submitted: dcrSubmitted, Approved: dcrApproved,
		},
	}, nil
}

func (s *Service) DistributionSummary(ctx context.Context, principal auth.Principal, from, to string) (*DistributionSummaryReport, error) {
	start, end, fromLabel, toLabel, err := periodBounds(from, to)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	orderScope := `(tenant_id = ? OR (distributor_tenant_id = ? AND pharmaos_active_distributor_link(tenant_id, distributor_tenant_id))) AND order_date BETWEEN ? AND ?`
	orderArgs := []any{principal.TenantID, principal.TenantID, start, end}

	type result struct {
		Total     int64
		Submitted int64
		Approved  int64
		Cancelled int64
		Value     float64
	}
	var orders result
	if err := db.Raw(`SELECT
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE status = 'SUBMITTED') AS submitted,
		COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved,
		COUNT(*) FILTER (WHERE status = 'CANCELLED') AS cancelled,
		COALESCE(SUM(total_amount), 0) AS value
		FROM dist_orders WHERE `+orderScope, orderArgs...).Scan(&orders).Error; err != nil {
		return nil, err
	}

	var inventory struct {
		SKUCount int64
		OnHand   int64
		Reserved int64
	}
	if err := db.Raw(`SELECT
		COUNT(*) AS sku_count,
		COALESCE(SUM(quantity_on_hand), 0) AS on_hand,
		COALESCE(SUM(quantity_reserved), 0) AS reserved
		FROM dist_inventory WHERE tenant_id = ? OR pharmaos_can_view_distributor_data(tenant_id)`,
		principal.TenantID).Scan(&inventory).Error; err != nil {
		return nil, err
	}

	var sales struct {
		Records int64
		Units   int64
		Value   float64
	}
	if err := db.Raw(`SELECT
		COUNT(*) AS records,
		COALESCE(SUM(quantity), 0) AS units,
		COALESCE(SUM(total_amount), 0) AS value
		FROM dist_secondary_sales
		WHERE (tenant_id = ? OR (company_tenant_id = ? AND pharmaos_active_distributor_link(company_tenant_id, tenant_id)))
		  AND sale_date BETWEEN ? AND ?`,
		principal.TenantID, principal.TenantID, start, end).Scan(&sales).Error; err != nil {
		return nil, err
	}

	var activeLinks int64
	if err := db.Raw(`SELECT COUNT(*) FROM dist_company_distributor_links
		WHERE (tenant_id = ? OR distributor_tenant_id = ?)
		  AND status = 'ACTIVE'
		  AND effective_from <= CURRENT_DATE
		  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)`,
		principal.TenantID, principal.TenantID).Scan(&activeLinks).Error; err != nil {
		return nil, err
	}

	return &DistributionSummaryReport{
		Period: Period{From: fromLabel, To: toLabel},
		Orders: OrderMetrics{
			Total: orders.Total, Submitted: orders.Submitted, Approved: orders.Approved,
			Cancelled: orders.Cancelled, Value: orders.Value,
		},
		Inventory: InventoryMetrics{SKUCount: inventory.SKUCount, OnHand: inventory.OnHand, Reserved: inventory.Reserved},
		SecondarySales: SecondarySalesMetrics{
			Records: sales.Records, Units: sales.Units, Value: sales.Value,
		},
		ActiveLinks: activeLinks,
	}, nil
}

func (s *Service) AuditTrail(ctx context.Context, principal auth.Principal, page query.Page, from, to, module, action string) ([]AuditEntry, int64, error) {
	start, end, _, _, err := periodBounds(from, to)
	if err != nil {
		return nil, 0, err
	}
	base := s.db.WithContext(ctx).Model(&AuditEntry{}).
		Where("tenant_id = ?", principal.TenantID).
		Where("created_at BETWEEN ? AND ?", start, end)
	if strings.TrimSpace(module) != "" {
		base = base.Where("module = ?", strings.TrimSpace(module))
	}
	if strings.TrimSpace(action) != "" {
		base = base.Where("action = ?", strings.TrimSpace(action))
	}
	if page.Search != "" {
		base = base.Where("entity_id ILIKE ?", "%"+page.Search+"%")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sortField := "created_at"
	if page.Sort == "module" || page.Sort == "action" || page.Sort == "entity_type" {
		sortField = page.Sort
	}
	sortOrder := "DESC"
	if strings.EqualFold(page.Order, "asc") {
		sortOrder = "ASC"
	}
	entries := make([]AuditEntry, 0, page.Size)
	if err := base.Order(fmt.Sprintf("%s %s", sortField, sortOrder)).
		Offset(page.Offset()).Limit(page.Size).Find(&entries).Error; err != nil {
		return nil, 0, err
	}
	return entries, total, nil
}

func (s *Service) ListDefinitions(ctx context.Context, principal auth.Principal, page query.Page) ([]ReportDefinition, int64, error) {
	base := s.db.WithContext(ctx).Model(&ReportDefinition{}).Where("tenant_id = ?", principal.TenantID)
	if page.Search != "" {
		base = base.Where("name ILIKE ? OR code ILIKE ?", "%"+page.Search+"%", "%"+page.Search+"%")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ReportDefinition, 0, page.Size)
	if err := base.Order("name ASC").Offset(page.Offset()).Limit(page.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) CreateDefinition(ctx context.Context, principal auth.Principal, request ReportDefinitionRequest) (*ReportDefinition, error) {
	if !principal.HasPermission("report.manage") {
		return nil, apperrors.ErrForbidden
	}
	if request.Config == nil {
		request.Config = JSONMap{}
	}
	record := ReportDefinition{
		TenantID: principal.TenantID, Code: strings.TrimSpace(request.Code), Name: strings.TrimSpace(request.Name),
		Description: strings.TrimSpace(request.Description), ReportType: request.ReportType,
		Config: request.Config, CreatedBy: principal.UserID,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "CREATE", "report_definition", record.ID.String())
	return &record, nil
}

func (s *Service) CreateSnapshot(ctx context.Context, principal auth.Principal, request SnapshotRequest) (*ReportSnapshot, error) {
	if !principal.HasPermission("report.manage") {
		return nil, apperrors.ErrForbidden
	}
	start, end, _, _, err := periodBounds(request.From, request.To)
	if err != nil {
		return nil, err
	}
	record := ReportSnapshot{
		TenantID: principal.TenantID, DefinitionID: request.DefinitionID,
		PeriodFrom: &start, PeriodTo: &end, Payload: request.Payload, CreatedBy: principal.UserID,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	s.record(ctx, principal, "CREATE", "report_snapshot", record.ID.String())
	return &record, nil
}

func (s *Service) ListSnapshots(ctx context.Context, principal auth.Principal, page query.Page) ([]ReportSnapshot, int64, error) {
	base := s.db.WithContext(ctx).Model(&ReportSnapshot{}).Where("tenant_id = ?", principal.TenantID)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]ReportSnapshot, 0, page.Size)
	if err := base.Order("created_at DESC").Offset(page.Offset()).Limit(page.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetSnapshot(ctx context.Context, principal auth.Principal, id uuid.UUID) (*ReportSnapshot, error) {
	var record ReportSnapshot
	if err := s.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", principal.TenantID, id).First(&record).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action, entityType, entityID string) {
	if s.audit == nil {
		return
	}
	tenantID := principal.TenantID
	userID := principal.UserID
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &tenantID, UserID: &userID, Action: action,
		Module: "reporting", EntityType: entityType, EntityID: entityID,
	})
}

func periodBounds(from, to string) (time.Time, time.Time, string, string, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.AddDate(0, 0, -29)
	end := today
	var err error
	if strings.TrimSpace(from) != "" {
		start, err = time.Parse("2006-01-02", strings.TrimSpace(from))
		if err != nil {
			return time.Time{}, time.Time{}, "", "", apperrors.New("INVALID_DATE_FROM", "from must be YYYY-MM-DD", 400)
		}
	}
	if strings.TrimSpace(to) != "" {
		end, err = time.Parse("2006-01-02", strings.TrimSpace(to))
		if err != nil {
			return time.Time{}, time.Time{}, "", "", apperrors.New("INVALID_DATE_TO", "to must be YYYY-MM-DD", 400)
		}
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, "", "", apperrors.New("INVALID_PERIOD", "to must not precede from", 400)
	}
	end = end.Add(24*time.Hour - time.Nanosecond)
	return start, end, start.Format("2006-01-02"), end.Format("2006-01-02"), nil
}

func dataScope(principal auth.Principal) string {
	if principal.DataScope == "" {
		return "TENANT"
	}
	return principal.DataScope
}

func scopeQuery(db *gorm.DB, principal auth.Principal, alias string) *gorm.DB {
	if principal.IsPrivileged || principal.DataScope == "" || principal.DataScope == "TENANT" {
		return db
	}
	switch principal.DataScope {
	case "SELF":
		subquery := db.Session(&gorm.Session{NewDB: true}).Table("employees").Select("id").
			Where("tenant_id = ? AND user_id = ?", principal.TenantID, principal.UserID)
		return db.Where(alias+".employee_id IN (?)", subquery)
	case "TEAM":
		return db.Where(alias+".employee_id IN (?)", db.Raw(`WITH RECURSIVE team AS (
			SELECT id FROM employees WHERE tenant_id = ? AND user_id = ?
			UNION
			SELECT e.id FROM employees e JOIN team t ON e.manager_employee_id = t.id WHERE e.tenant_id = ?
		) SELECT id FROM team`, principal.TenantID, principal.UserID, principal.TenantID))
	case "DIVISION":
		if len(principal.DivisionIDs) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(`EXISTS (SELECT 1 FROM employees e WHERE e.id = `+alias+`.employee_id
			AND e.tenant_id = ? AND e.division_id IN ?)`, principal.TenantID, principal.DivisionIDs)
	case "TERRITORY":
		if len(principal.TerritoryIDs) == 0 {
			return db.Where("1 = 0")
		}
		return db.Where(`EXISTS (SELECT 1 FROM employees e WHERE e.id = `+alias+`.employee_id
			AND e.tenant_id = ? AND e.territory_id IN ?)`, principal.TenantID, principal.TerritoryIDs)
	default:
		return db
	}
}
