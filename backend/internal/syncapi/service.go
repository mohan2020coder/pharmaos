package syncapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/dcr"
	"github.com/pharmaos/pharmaos/backend/internal/distribution"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/scope"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
	"github.com/pharmaos/pharmaos/backend/internal/tours"
	"github.com/pharmaos/pharmaos/backend/internal/visits"
)

type Service struct {
	db           *gorm.DB
	visits       *visits.Service
	dcr          *dcr.Service
	tours        *tours.Service
	distribution *distribution.Service
}

func NewService(db *gorm.DB, visitService *visits.Service, dcrService *dcr.Service, tourService *tours.Service, distributionService *distribution.Service) *Service {
	return &Service{db: db, visits: visitService, dcr: dcrService, tours: tourService, distribution: distributionService}
}

func (s *Service) Changes(ctx context.Context, principal auth.Principal, since string, entityTypes string, limit int) (*ChangesResponse, error) {
	cursor, err := decodeCursor(since)
	if err != nil {
		return nil, apperrors.New("INVALID_CURSOR", "since is not a valid sync cursor", 400)
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	entities, err := selectedEntities(entityTypes)
	if err != nil {
		return nil, err
	}

	changes := make([]Change, 0, limit+len(entities))
	for _, entity := range entities {
		spec := entitySpecs[entity]
		db := s.baseQuery(ctx, principal, entity, spec)
		if !cursor.UpdatedAt.IsZero() {
			switch {
			case entity < cursor.Entity:
				db = db.Where(spec.UpdatedColumn+" > ?", cursor.UpdatedAt)
			case entity == cursor.Entity:
				db = db.Where("("+spec.UpdatedColumn+" > ? OR ("+spec.UpdatedColumn+" = ? AND id > ?))",
					cursor.UpdatedAt, cursor.UpdatedAt, cursor.ID)
			default:
				db = db.Where(spec.UpdatedColumn+" >= ?", cursor.UpdatedAt)
			}
		}
		var rows []map[string]interface{}
		if err := db.Order(spec.UpdatedColumn + " ASC, id ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			changes = append(changes, changeFromRow(entity, spec, row))
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		if !changes[i].UpdatedAt.Equal(changes[j].UpdatedAt) {
			return changes[i].UpdatedAt.Before(changes[j].UpdatedAt)
		}
		if changes[i].Entity != changes[j].Entity {
			return changes[i].Entity < changes[j].Entity
		}
		return changes[i].EntityID < changes[j].EntityID
	})

	hasMore := len(changes) > limit
	if hasMore {
		changes = changes[:limit]
	}
	nextCursor := ""
	if len(changes) > 0 {
		last := changes[len(changes)-1]
		id, _ := uuid.Parse(last.EntityID)
		nextCursor = encodeCursor(cursorData{UpdatedAt: last.UpdatedAt, Entity: last.Entity, ID: id})
	}
	return &ChangesResponse{NextCursor: nextCursor, HasMore: hasMore, Changes: changes}, nil
}

func (s *Service) Status(ctx context.Context, principal auth.Principal) (*StatusResponse, error) {
	serverTime := time.Now().UTC()
	entities := make([]string, 0, len(entitySpecs))
	for entity := range entitySpecs {
		entities = append(entities, entity)
	}
	sort.Strings(entities)

	counts := make(map[string]int64, len(entities))
	var latest cursorData
	for _, entity := range entities {
		spec := entitySpecs[entity]
		base := s.baseQuery(ctx, principal, entity, spec)
		var count int64
		if err := base.Model(&struct{}{}).Count(&count).Error; err != nil {
			return nil, err
		}
		counts[entity] = count

		var row struct {
			UpdatedAt time.Time `gorm:"column:updated_at"`
			ID        string    `gorm:"column:id"`
		}
		err := s.baseQuery(ctx, principal, entity, spec).
			Select(spec.UpdatedColumn + " AS updated_at, id::text AS id").
			Order(spec.UpdatedColumn + " DESC, id DESC").Limit(1).Scan(&row).Error
		if err != nil {
			return nil, err
		}
		if !row.UpdatedAt.IsZero() && (latest.UpdatedAt.IsZero() || row.UpdatedAt.After(latest.UpdatedAt)) {
			id, _ := uuid.Parse(row.ID)
			latest = cursorData{UpdatedAt: row.UpdatedAt, Entity: entity, ID: id}
		}
	}

	var pending int64
	if err := s.db.WithContext(ctx).Table("idempotency_keys").
		Where("tenant_id = ? AND operation = ? AND status = ?", principal.TenantID, "sync.operation", "PENDING").
		Count(&pending).Error; err != nil {
		return nil, err
	}

	latestCursor := ""
	if !latest.UpdatedAt.IsZero() {
		latestCursor = encodeCursor(latest)
	}
	return &StatusResponse{
		ServerTime: serverTime, LatestCursor: latestCursor, PendingOperations: pending,
		SupportedEntities: entities, EntityCounts: counts,
	}, nil
}

func (s *Service) Push(ctx context.Context, principal auth.Principal, request PushRequest) (*PushResponse, error) {
	if len(request.Operations) == 0 {
		return &PushResponse{Operations: []OperationResult{}}, nil
	}
	if len(request.Operations) > 100 {
		return nil, apperrors.New("BATCH_TOO_LARGE", "A sync batch may contain at most 100 operations", 422)
	}
	results := make([]OperationResult, 0, len(request.Operations))
	for _, operation := range request.Operations {
		results = append(results, s.applyOperation(ctx, principal, operation))
	}
	return &PushResponse{Operations: results}, nil
}

func (s *Service) baseQuery(ctx context.Context, principal auth.Principal, entity string, spec entitySpec) *gorm.DB {
	db := s.db.WithContext(ctx).Table(spec.Table)
	switch spec.Visibility {
	case "distribution_link":
		db = db.Where("tenant_id = ? OR distributor_tenant_id = ?", principal.TenantID, principal.TenantID)
	case "distributor_profile":
		db = db.Where(`tenant_id = ? OR EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.distributor_tenant_id = dist_distributors.tenant_id
			  AND l.tenant_id = ?
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE))`, principal.TenantID, principal.TenantID)
	case "order":
		db = db.Where(`tenant_id = ? OR (distributor_tenant_id = ? AND EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.tenant_id = dist_orders.tenant_id
			  AND l.distributor_tenant_id = dist_orders.distributor_tenant_id
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))`, principal.TenantID, principal.TenantID)
	case "order_child":
		db = db.Where(`EXISTS (
			SELECT 1 FROM dist_orders o
			WHERE o.id = dist_order_items.order_id
			  AND (o.tenant_id = ? OR (o.distributor_tenant_id = ? AND EXISTS (
				SELECT 1 FROM dist_company_distributor_links l
				WHERE l.tenant_id = o.tenant_id
				  AND l.distributor_tenant_id = o.distributor_tenant_id
				  AND l.status = 'ACTIVE'
				  AND l.effective_from <= CURRENT_DATE
				  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))))`, principal.TenantID, principal.TenantID)
	case "distributor_data":
		db = db.Where(`tenant_id = ? OR EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.distributor_tenant_id = `+spec.Table+`.tenant_id
			  AND l.tenant_id = ?
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE))`, principal.TenantID, principal.TenantID)
	case "secondary_sale":
		db = db.Where(`tenant_id = ? OR (company_tenant_id = ? AND EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.tenant_id = dist_secondary_sales.company_tenant_id
			  AND l.distributor_tenant_id = dist_secondary_sales.tenant_id
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))`, principal.TenantID, principal.TenantID)
	default:
		db = db.Where("tenant_id = ?", principal.TenantID)
	}
	if spec.ParentTable != "" && spec.Visibility == "" {
		sub := s.db.WithContext(ctx).Table(spec.ParentTable).Select("id").Where("tenant_id = ?", principal.TenantID)
		sub = scope.Apply(sub, principal, spec.ParentEmployeeColumn, "", "")
		return db.Where(spec.ParentIDColumn+" IN (?)", sub)
	}
	if spec.EmployeeColumn != "" {
		db = scope.Apply(db, principal, spec.EmployeeColumn, spec.DivisionColumn, spec.TerritoryColumn)
	}
	if spec.TerritoryColumn != "" && len(principal.TerritoryIDs) > 0 {
		db = db.Where(spec.TerritoryColumn+" IN ?", principal.TerritoryIDs)
	}
	_ = entity
	return db
}

func (s *Service) applyOperation(ctx context.Context, principal auth.Principal, operation Operation) OperationResult {
	result := OperationResult{ClientOperationID: operation.ID, Entity: operation.Entity, Status: "REJECTED"}
	if operation.ID == uuid.Nil || operation.IdempotencyKey == uuid.Nil {
		result.ErrorCode, result.ErrorMessage = "INVALID_OPERATION", "id and idempotency_key are required"
		return result
	}
	if operation.Entity == "" || operation.Operation == "" {
		result.ErrorCode, result.ErrorMessage = "INVALID_OPERATION", "entity and operation are required"
		return result
	}

	hash := hashOperation(operation)
	replay, retryable := s.replayOperation(ctx, principal, operation, hash)
	if replay != nil {
		return *replay
	}
	if retryable {
		result.ErrorCode, result.ErrorMessage, result.Retryable = "IDEMPOTENCY_BUSY", "Operation is already in progress", true
		return result
	}

	if err := s.reserveOperation(ctx, principal, operation, hash); err != nil {
		mapped := mapError(err, operation)
		mapped.ErrorCode, mapped.ErrorMessage, mapped.Retryable = "IDEMPOTENCY_ERROR", err.Error(), true
		return mapped
	}

	entityID, err := s.executeOperation(ctx, principal, operation)
	if err != nil {
		mapped := mapError(err, operation)
		_ = s.db.WithContext(ctx).Exec(`DELETE FROM idempotency_keys WHERE tenant_id = ? AND key = ? AND operation = ?`,
			principal.TenantID, operation.ID, "sync.operation").Error
		return mapped
	}

	result.Status, result.EntityID = "APPLIED", entityID
	stored, _ := json.Marshal(storedSyncResult{Hash: hash, Result: result})
	_ = s.db.WithContext(ctx).Exec(`UPDATE idempotency_keys SET status = 'COMPLETED', entity_id = ?, result = ?::jsonb
		WHERE tenant_id = ? AND key = ? AND operation = ?`,
		result.EntityID, string(stored), principal.TenantID, operation.ID, "sync.operation").Error
	return result
}

func (s *Service) executeOperation(ctx context.Context, principal auth.Principal, operation Operation) (uuid.UUID, error) {
	clientID := operation.EntityID
	switch operation.Operation {
	case "visit.create":
		var request visits.CreateRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.visits.Create(ctx, principal, request, operation.IdempotencyKey)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "visit.submit":
		entity, err := s.visits.Submit(ctx, principal, operation.EntityID)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "visit.attachment":
		var request visits.AttachmentRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.visits.AddAttachment(ctx, principal, operation.EntityID, request)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "dcr.create":
		var request dcr.CreateRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.dcr.Create(ctx, principal, request)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "dcr.product.add":
		var request dcr.ProductRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.dcr.AddProduct(ctx, principal, operation.EntityID, request)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "dcr.submit":
		entity, err := s.dcr.Submit(ctx, principal, operation.EntityID)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "dcr.approve", "dcr.reject":
		if !principal.HasPermission("dcr.review") {
			return uuid.Nil, apperrors.ErrForbidden
		}
		var request dcr.DecisionRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		entity, err := s.dcr.Decide(ctx, principal, operation.EntityID, operation.Operation == "dcr.approve", request.Remarks)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "tour.create":
		var request tours.CreateRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.tours.Create(ctx, principal, request)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "tour.line.add":
		var request tours.LineRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		entity, err := s.tours.AddLine(ctx, principal, operation.EntityID, request)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "tour.submit":
		entity, err := s.tours.Submit(ctx, principal, operation.EntityID)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "tour.approve", "tour.reject":
		if !principal.HasPermission("tourplan.approve") {
			return uuid.Nil, apperrors.ErrForbidden
		}
		var request tours.DecisionRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		entity, err := s.tours.Decide(ctx, principal, operation.EntityID, operation.Operation == "tour.approve", request.Comments)
		if err != nil {
			return uuid.Nil, err
		}
		return entity.ID, nil
	case "order.create":
		var request distribution.OrderCreateRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		order, err := s.distribution.CreateOrder(ctx, principal, request, operation.IdempotencyKey)
		if err != nil {
			return uuid.Nil, err
		}
		return order.ID, nil
	case "order.submit":
		order, err := s.distribution.SubmitOrder(ctx, principal, operation.EntityID)
		if err != nil {
			return uuid.Nil, err
		}
		return order.ID, nil
	case "order.approve":
		var request distribution.OrderDecisionRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		order, err := s.distribution.ApproveOrder(ctx, principal, operation.EntityID, request.Remarks)
		if err != nil {
			return uuid.Nil, err
		}
		return order.ID, nil
	case "secondary_sale.create":
		var request distribution.SecondarySaleRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		sale, err := s.distribution.CreateSecondarySale(ctx, principal, request, operation.IdempotencyKey)
		if err != nil {
			return uuid.Nil, err
		}
		return sale.ID, nil
	case "inventory.movement":
		var request distribution.InventoryAdjustmentRequest
		if err := decodePayload(operation.Payload, &request); err != nil {
			return uuid.Nil, err
		}
		request.ID = &clientID
		if fields := validate.Struct(request); fields != nil {
			return uuid.Nil, invalidPayload(fields)
		}
		movement, err := s.distribution.AdjustInventory(ctx, principal, request, operation.IdempotencyKey)
		if err != nil {
			return uuid.Nil, err
		}
		return movement.ID, nil
	default:
		return uuid.Nil, apperrors.New("UNSUPPORTED_OPERATION", "Unsupported sync operation", 422)
	}
}

type storedSyncResult struct {
	Hash   string          `json:"hash"`
	Result OperationResult `json:"result"`
}

func (s *Service) replayOperation(ctx context.Context, principal auth.Principal, operation Operation, hash string) (*OperationResult, bool) {
	var record struct {
		Status string `gorm:"column:status"`
		Result []byte `gorm:"column:result"`
	}
	err := s.db.WithContext(ctx).Raw(`SELECT status, COALESCE(result, '{}'::jsonb)::text AS result
		FROM idempotency_keys WHERE tenant_id = ? AND key = ? AND operation = ?`,
		principal.TenantID, operation.ID, "sync.operation").Scan(&record).Error
	if err != nil || record.Status == "" {
		return nil, false
	}
	var stored storedSyncResult
	_ = json.Unmarshal(record.Result, &stored)
	if stored.Hash != "" && stored.Hash != hash {
		result := OperationResult{
			ClientOperationID: operation.ID, Entity: operation.Entity, Status: "REJECTED",
			ErrorCode: "IDEMPOTENCY_CONFLICT", ErrorMessage: "Idempotency key reused with a different payload",
		}
		return &result, false
	}
	if record.Status == "PENDING" {
		_ = s.db.WithContext(ctx).Exec(`DELETE FROM idempotency_keys WHERE tenant_id = ? AND key = ? AND operation = ?`,
			principal.TenantID, operation.ID, "sync.operation").Error
		return nil, false
	}
	if record.Status != "COMPLETED" {
		_ = s.db.WithContext(ctx).Exec(`DELETE FROM idempotency_keys WHERE tenant_id = ? AND key = ? AND operation = ?`,
			principal.TenantID, operation.ID, "sync.operation").Error
		return nil, false
	}
	stored.Result.Status = "REPLAYED"
	stored.Result.Retryable = false
	return &stored.Result, false
}

func (s *Service) reserveOperation(ctx context.Context, principal auth.Principal, operation Operation, hash string) error {
	stored, _ := json.Marshal(storedSyncResult{Hash: hash})
	result := s.db.WithContext(ctx).Exec(`INSERT INTO idempotency_keys (tenant_id, key, operation, entity_id, status, result)
		VALUES (?, ?, 'sync.operation', ?, 'PENDING', ?::jsonb)
		ON CONFLICT (tenant_id, key) DO NOTHING`,
		principal.TenantID, operation.ID, operation.EntityID, string(stored))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrConflict
	}
	return nil
}

func selectedEntities(selector string) ([]string, error) {
	if strings.TrimSpace(selector) == "" {
		entities := make([]string, 0, len(entitySpecs))
		for entity := range entitySpecs {
			entities = append(entities, entity)
		}
		sort.Strings(entities)
		return entities, nil
	}
	seen := map[string]bool{}
	entities := []string{}
	for _, value := range strings.Split(selector, ",") {
		entity := strings.TrimSpace(value)
		if entity == "" {
			continue
		}
		if _, ok := entitySpecs[entity]; !ok {
			return nil, apperrors.New("UNKNOWN_ENTITY", "Unsupported sync entity: "+entity, 422)
		}
		if !seen[entity] {
			seen[entity] = true
			entities = append(entities, entity)
		}
	}
	sort.Strings(entities)
	return entities, nil
}

func changeFromRow(entity string, spec entitySpec, row map[string]interface{}) Change {
	entityID := fmt.Sprint(row["id"])
	updatedAt := rowTime(row[spec.UpdatedColumn])
	deleted := false
	if spec.DeletedColumn != "" {
		deleted = row[spec.DeletedColumn] != nil && fmt.Sprint(row[spec.DeletedColumn]) != ""
	}
	op := "UPSERT"
	if deleted {
		op = "DELETE"
	}
	return Change{Entity: entity, EntityID: entityID, Operation: op, UpdatedAt: updatedAt, Payload: row}
}

func rowTime(value interface{}) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case *time.Time:
		if typed != nil {
			return *typed
		}
	case string:
		if parsed, err := time.Parse(time.RFC3339, typed); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func decodeCursor(value string) (cursorData, error) {
	var cursor cursorData
	if strings.TrimSpace(value) == "" {
		return cursor, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor, err
	}
	if err := json.Unmarshal(raw, &cursor); err != nil {
		return cursor, err
	}
	return cursor, nil
}

func encodeCursor(cursor cursorData) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func hashOperation(operation Operation) string {
	payload := operation.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s", operation.ID, operation.Entity, operation.Operation, payload)))
	return hex.EncodeToString(sum[:])
}

func decodePayload(raw json.RawMessage, target interface{}) error {
	if len(raw) == 0 {
		raw = json.RawMessage("{}")
	}
	return json.Unmarshal(raw, target)
}

func invalidPayload(fields map[string]string) error {
	encoded, _ := json.Marshal(fields)
	return apperrors.New("VALIDATION_FAILED", "Invalid sync payload: "+string(encoded), http.StatusUnprocessableEntity)
}

func mapError(err error, operation Operation) OperationResult {
	result := OperationResult{ClientOperationID: operation.ID, Entity: operation.Entity, Status: "REJECTED"}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		result.ErrorCode, result.ErrorMessage = appErr.Code, appErr.Message
		if appErr.Status == http.StatusConflict {
			result.Status = "CONFLICT"
		}
		result.Retryable = appErr.Status >= http.StatusInternalServerError
		return result
	}
	result.ErrorCode, result.ErrorMessage, result.Retryable = "INTERNAL", "Internal server error", true
	return result
}

func intParam(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
