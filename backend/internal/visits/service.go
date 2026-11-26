package visits

import (
	"context"
	"encoding/json"
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
	ID                *uuid.UUID `json:"id"`
	EmployeeID        uuid.UUID  `json:"employee_id" validate:"required,uuid"`
	TourPlanLineID    *uuid.UUID `json:"tour_plan_line_id"`
	TerritoryID       *uuid.UUID `json:"territory_id"`
	EntityType        string     `json:"entity_type" validate:"required,oneof=DOCTOR CHEMIST STOCKIST"`
	EntityID          uuid.UUID  `json:"entity_id" validate:"required,uuid"`
	ClientReferenceID *uuid.UUID `json:"client_reference_id"`
	VisitedAt         string     `json:"visited_at"`
	CheckInLat        *float64   `json:"check_in_lat"`
	CheckInLng        *float64   `json:"check_in_lng"`
	CheckInAccuracy   *float64   `json:"check_in_accuracy"`
	Remarks           string     `json:"remarks"`
}

type AttachmentRequest struct {
	ID          *uuid.UUID `json:"id"`
	FileName    string     `json:"file_name" validate:"required"`
	ContentType string     `json:"content_type"`
	StorageKey  string     `json:"storage_key" validate:"required"`
	Checksum    string     `json:"checksum"`
}

type Service struct {
	visits *crud.Repository[Visit]
	db     *gorm.DB
	audit  *audit.Service
}

func NewService(db *gorm.DB, auditService *audit.Service) *Service {
	return &Service{visits: crud.New[Visit](db), db: db, audit: auditService}
}

func (s *Service) Create(ctx context.Context, principal auth.Principal, request CreateRequest, idempotencyKey uuid.UUID) (*Visit, error) {
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, request.EmployeeID); err != nil {
		return nil, err
	}
	if err := validateCoordinates(request.CheckInLat, request.CheckInLng); err != nil {
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
		return nil, apperrors.New("INVALID_ENTITY", "Visit entity does not exist", 422)
	}
	db := s.db.WithContext(ctx)
	result := db.Exec(`INSERT INTO idempotency_keys (tenant_id, key, operation, status)
		VALUES (?, ?, 'visit.create', 'PENDING')
		ON CONFLICT (tenant_id, key) DO NOTHING`, principal.TenantID, idempotencyKey)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		var existing struct {
			Operation string     `gorm:"column:operation"`
			Status    string     `gorm:"column:status"`
			EntityID  *uuid.UUID `gorm:"column:entity_id"`
		}
		if err := db.Raw(`SELECT operation, status, entity_id FROM idempotency_keys WHERE tenant_id = ? AND key = ?`,
			principal.TenantID, idempotencyKey).Scan(&existing).Error; err != nil {
			return nil, err
		}
		if existing.Operation != "visit.create" {
			return nil, apperrors.ErrConflict
		}
		if existing.Status == "COMPLETED" && existing.EntityID != nil {
			return s.visits.Get(ctx, principal.TenantID, *existing.EntityID)
		}
		return nil, apperrors.ErrConflict
	}
	visitedAt := time.Now().UTC()
	if request.VisitedAt != "" {
		parsed, err := time.Parse(time.RFC3339, request.VisitedAt)
		if err != nil {
			s.failIdempotency(ctx, principal.TenantID, idempotencyKey, err.Error())
			return nil, apperrors.New("INVALID_DATE", "visited_at must be RFC3339", 422)
		}
		visitedAt = parsed
	}
	recordID := uuid.New()
	if request.ID != nil {
		recordID = *request.ID
	}
	visit := &Visit{
		ID: recordID, TenantID: principal.TenantID, EmployeeID: request.EmployeeID,
		TourPlanLineID: request.TourPlanLineID, TerritoryID: request.TerritoryID,
		EntityType: request.EntityType, EntityID: request.EntityID,
		ClientReferenceID: request.ClientReferenceID, VisitedAt: visitedAt,
		CheckInLat: request.CheckInLat, CheckInLng: request.CheckInLng,
		CheckInAccuracy: request.CheckInAccuracy, Status: "DRAFT", Remarks: request.Remarks,
	}
	if err := s.visits.Create(ctx, visit); err != nil {
		s.failIdempotency(ctx, principal.TenantID, idempotencyKey, err.Error())
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	payload, _ := json.Marshal(map[string]interface{}{"id": visit.ID, "status": visit.Status})
	if err := db.Exec(`UPDATE idempotency_keys SET status = 'COMPLETED', entity_id = ?, result = ?::jsonb
		WHERE tenant_id = ? AND key = ?`, visit.ID, string(payload), principal.TenantID, idempotencyKey).Error; err != nil {
		return nil, err
	}
	s.record(ctx, principal, "VISIT_CREATE", visit)
	return visit, nil
}

func (s *Service) List(ctx context.Context, principal auth.Principal, page query.Page, employeeID, territoryID uuid.UUID, entityType, status, from, to string) ([]Visit, int64, error) {
	db := scope.Apply(s.db.WithContext(ctx), principal, "employee_id", "", "territory_id")
	db = db.Model(&Visit{}).Where("tenant_id = ?", principal.TenantID)
	if employeeID != uuid.Nil {
		db = db.Where("employee_id = ?", employeeID)
	}
	if territoryID != uuid.Nil {
		db = db.Where("territory_id = ?", territoryID)
	}
	if entityType != "" {
		db = db.Where("entity_type = ?", entityType)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	if from != "" {
		parsed, _, err := parseTimeFilter(from)
		if err != nil {
			return nil, 0, apperrors.New("INVALID_DATE", "from must be RFC3339 or YYYY-MM-DD", 422)
		}
		db = db.Where("visited_at >= ?", parsed)
	}
	if to != "" {
		parsed, dateOnly, err := parseTimeFilter(to)
		if err != nil {
			return nil, 0, apperrors.New("INVALID_DATE", "to must be RFC3339 or YYYY-MM-DD", 422)
		}
		if dateOnly {
			parsed = parsed.AddDate(0, 0, 1)
		}
		db = db.Where("visited_at < ?", parsed)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Visit
	err := db.Order(page.Sort + " " + page.Order).Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) Get(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Visit, error) {
	visit, err := s.visits.Get(ctx, principal.TenantID, id)
	if err != nil {
		return nil, err
	}
	if err := scope.EmployeeInScopeError(s.db.WithContext(ctx), principal, visit.EmployeeID); err != nil {
		return nil, err
	}
	return visit, nil
}

func (s *Service) Submit(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Visit, error) {
	visit, err := s.Get(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if visit.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_STATE", "Only draft visits can be submitted", 422)
	}
	now := time.Now().UTC()
	if err := s.visits.Update(ctx, principal.TenantID, id, map[string]interface{}{
		"status": "SUBMITTED", "submitted_at": now,
	}); err != nil {
		return nil, err
	}
	visit.Status, visit.SubmittedAt = "SUBMITTED", &now
	s.record(ctx, principal, "VISIT_SUBMIT", visit)
	return visit, nil
}

func (s *Service) AddAttachment(ctx context.Context, principal auth.Principal, visitID uuid.UUID, request AttachmentRequest) (*Attachment, error) {
	visit, err := s.Get(ctx, principal, visitID)
	if err != nil {
		return nil, err
	}
	if visit.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_STATE", "Attachments can only be added to draft visits", 422)
	}
	attachmentID := uuid.New()
	if request.ID != nil {
		attachmentID = *request.ID
	}
	attachment := &Attachment{
		ID: attachmentID, TenantID: principal.TenantID, VisitID: visitID,
		FileName: request.FileName, ContentType: request.ContentType,
		StorageKey: request.StorageKey, Checksum: request.Checksum,
	}
	err = s.db.WithContext(ctx).Create(attachment).Error
	return attachment, err
}

func (s *Service) failIdempotency(ctx context.Context, tenantID, key uuid.UUID, message string) {
	_ = s.db.WithContext(ctx).Exec(`UPDATE idempotency_keys SET status = 'FAILED', error_message = ?
		WHERE tenant_id = ? AND key = ?`, message, tenantID, key).Error
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action string, visit *Visit) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID, Action: action,
		Module: "visits", EntityType: "visit", EntityID: visit.ID.String(), New: visit,
	})
}

func validateCoordinates(lat, lng *float64) error {
	if (lat == nil) != (lng == nil) {
		return apperrors.New("INVALID_COORDINATES", "latitude and longitude must be supplied together", 422)
	}
	if lat != nil && (*lat < -90 || *lat > 90 || *lng < -180 || *lng > 180) {
		return apperrors.New("INVALID_COORDINATES", "coordinates are out of range", 422)
	}
	return nil
}

func parseTimeFilter(value string) (time.Time, bool, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, false, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	return parsed, true, err
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
