package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Options struct {
	Timeout          time.Duration
	MaxAttempts      int
	AllowInsecureURL bool
}

type Service struct {
	db      *gorm.DB
	audit   *audit.Service
	client  *http.Client
	options Options
}

func NewService(db *gorm.DB, auditService *audit.Service, options Options) *Service {
	if options.Timeout <= 0 {
		options.Timeout = 10 * time.Second
	}
	if options.MaxAttempts <= 0 {
		options.MaxAttempts = 8
	}
	return &Service{
		db: db, audit: auditService, options: options,
		client: &http.Client{Timeout: options.Timeout},
	}
}

func (s *Service) CreateConnection(ctx context.Context, principal auth.Principal, request ConnectionCreateRequest) (*IntegrationConnection, error) {
	if !principal.HasPermission("integration.manage") {
		return nil, apperrors.ErrForbidden
	}
	if request.Kind == "WEBHOOK" && strings.TrimSpace(request.EndpointURL) == "" {
		return nil, apperrors.New("ENDPOINT_REQUIRED", "Webhook connections require endpoint_url", 422)
	}
	if strings.TrimSpace(request.EndpointURL) != "" {
		if err := validateEndpointURL(request.EndpointURL, s.options.AllowInsecureURL); err != nil {
			return nil, apperrors.New("INVALID_ENDPOINT", err.Error(), 422)
		}
	}
	if request.Headers == nil {
		request.Headers = StringMap{}
	}
	record := IntegrationConnection{
		TenantID: principal.TenantID, Name: strings.TrimSpace(request.Name), Kind: request.Kind,
		EndpointURL: strings.TrimSpace(request.EndpointURL), SecretRef: strings.TrimSpace(request.SecretRef),
		Headers: request.Headers, Status: "ACTIVE", CreatedBy: principal.UserID,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "CREATE", "integration_connection", record.ID.String())
	return &record, nil
}

func (s *Service) ListConnections(ctx context.Context, principal auth.Principal, page query.Page, status string) ([]IntegrationConnection, int64, error) {
	base := s.db.WithContext(ctx).Model(&IntegrationConnection{}).Where("tenant_id = ?", principal.TenantID)
	if status != "" {
		base = base.Where("status = ?", status)
	}
	if page.Search != "" {
		base = base.Where("name ILIKE ? OR kind ILIKE ?", "%"+page.Search+"%", "%"+page.Search+"%")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]IntegrationConnection, 0, page.Size)
	if err := base.Order("name ASC").Offset(page.Offset()).Limit(page.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) SetConnectionStatus(ctx context.Context, principal auth.Principal, id uuid.UUID, status string) error {
	if !principal.HasPermission("integration.manage") {
		return apperrors.ErrForbidden
	}
	result := s.db.WithContext(ctx).Model(&IntegrationConnection{}).
		Where("tenant_id = ? AND id = ?", principal.TenantID, id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	s.record(ctx, principal, "UPDATE", "integration_connection", id.String())
	return nil
}

func (s *Service) DeleteConnection(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	if !principal.HasPermission("integration.manage") {
		return apperrors.ErrForbidden
	}
	result := s.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", principal.TenantID, id).Delete(&IntegrationConnection{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	s.record(ctx, principal, "DELETE", "integration_connection", id.String())
	return nil
}

func (s *Service) Publish(ctx context.Context, principal auth.Principal, request EventRequest) ([]WebhookDelivery, error) {
	deliveries, err := s.PublishSystem(ctx, principal.TenantID, request.EventType, request.EventID, request.Payload)
	if err != nil {
		return nil, err
	}
	if len(deliveries) > 0 {
		s.record(ctx, principal, "PUBLISH", "webhook_event", request.EventID)
	}
	return deliveries, nil
}

func (s *Service) PublishSystem(ctx context.Context, tenantID uuid.UUID, eventType, eventID string, payload JSONObject) ([]WebhookDelivery, error) {
	eventType = strings.TrimSpace(eventType)
	eventID = strings.TrimSpace(eventID)
	if eventType == "" || eventID == "" {
		return nil, apperrors.ErrValidation
	}
	if payload == nil {
		payload = JSONObject{}
	}
	var connections []IntegrationConnection
	if err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND kind = 'WEBHOOK' AND status = 'ACTIVE'", tenantID).
		Find(&connections).Error; err != nil {
		return nil, err
	}
	deliveries := make([]WebhookDelivery, 0, len(connections))
	for _, connection := range connections {
		delivery := WebhookDelivery{
			TenantID: tenantID, ConnectionID: connection.ID, EventType: eventType,
			EventID: eventID, Payload: payload, Status: "PENDING", NextAttemptAt: time.Now().UTC(),
		}
		err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&delivery).Error
		if err != nil {
			return nil, err
		}
		if delivery.ID != uuid.Nil {
			deliveries = append(deliveries, delivery)
		}
	}
	return deliveries, nil
}

func (s *Service) ListDeliveries(ctx context.Context, principal auth.Principal, page query.Page, status, eventType string) ([]WebhookDelivery, int64, error) {
	base := s.db.WithContext(ctx).Model(&WebhookDelivery{}).Where("tenant_id = ?", principal.TenantID)
	if status != "" {
		base = base.Where("status = ?", status)
	}
	if eventType != "" {
		base = base.Where("event_type = ?", eventType)
	}
	if page.Search != "" {
		base = base.Where("event_id ILIKE ?", "%"+page.Search+"%")
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]WebhookDelivery, 0, page.Size)
	if err := base.Order("created_at DESC").Offset(page.Offset()).Limit(page.Size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) RetryDelivery(ctx context.Context, principal auth.Principal, id uuid.UUID) error {
	if !principal.HasPermission("webhook.retry") {
		return apperrors.ErrForbidden
	}
	result := s.db.WithContext(ctx).Model(&WebhookDelivery{}).
		Where("tenant_id = ? AND id = ? AND status IN ?", principal.TenantID, id, []string{"FAILED", "DEAD"}).
		Updates(map[string]any{
			"status": "PENDING", "attempts": 0, "next_attempt_at": time.Now().UTC(),
			"last_error": nil, "response_status": nil, "delivered_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	s.record(ctx, principal, "RETRY", "webhook_delivery", id.String())
	return nil
}

func (s *Service) DispatchBatch(ctx context.Context) (int, error) {
	claimed := make([]WebhookDelivery, 0)
	err := s.withPrivileged(ctx, func(tx *gorm.DB) error {
		var acquired bool
		if err := tx.Raw(`SELECT pg_try_advisory_xact_lock(hashtext(?))`, "pharmaos:webhook-worker").Scan(&acquired).Error; err != nil {
			return err
		}
		if !acquired {
			return nil
		}
		var due []WebhookDelivery
		if err := tx.Where("status IN ? AND next_attempt_at <= ?", []string{"PENDING", "FAILED"}, time.Now().UTC()).
			Order("next_attempt_at ASC").Limit(25).Find(&due).Error; err != nil {
			return err
		}
		for _, delivery := range due {
			result := tx.Model(&WebhookDelivery{}).
				Where("id = ? AND status IN ?", delivery.ID, []string{"PENDING", "FAILED"}).
				Updates(map[string]any{
					"status": "IN_FLIGHT", "attempts": gorm.Expr("attempts + 1"), "updated_at": time.Now().UTC(),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				delivery.Attempts++
				claimed = append(claimed, delivery)
			}
		}
		return nil
	})
	if err != nil || len(claimed) == 0 {
		return len(claimed), err
	}
	for _, delivery := range claimed {
		if err := s.deliver(ctx, delivery); err != nil {
			return len(claimed), err
		}
	}
	return len(claimed), nil
}

func (s *Service) deliver(ctx context.Context, delivery WebhookDelivery) error {
	var connection IntegrationConnection
	if err := s.withPrivileged(ctx, func(tx *gorm.DB) error {
		return tx.Where("id = ? AND kind = 'WEBHOOK' AND status = 'ACTIVE'", delivery.ConnectionID).First(&connection).Error
	}); err != nil {
		return s.finishDelivery(ctx, delivery, 0, err, false)
	}
	if err := validateEndpointURL(connection.EndpointURL, s.options.AllowInsecureURL); err != nil {
		return s.finishDelivery(ctx, delivery, 0, err, false)
	}
	body, err := json.Marshal(delivery.Payload)
	if err != nil {
		return s.finishDelivery(ctx, delivery, 0, err, false)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, connection.EndpointURL, strings.NewReader(string(body)))
	if err != nil {
		return s.finishDelivery(ctx, delivery, 0, err, false)
	}
	for key, value := range connection.Headers {
		request.Header.Set(key, value)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-PharmaOS-Event", delivery.EventType)
	request.Header.Set("X-PharmaOS-Delivery", delivery.ID.String())
	request.Header.Set("X-PharmaOS-Attempt", fmt.Sprintf("%d", delivery.Attempts))
	if secret := os.Getenv(strings.TrimSpace(connection.SecretRef)); strings.TrimSpace(connection.SecretRef) != "" && secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		request.Header.Set("X-PharmaOS-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}

	response, err := s.client.Do(request)
	if err != nil {
		return s.finishDelivery(ctx, delivery, 0, err, false)
	}
	defer response.Body.Close()
	succeeded := response.StatusCode >= 200 && response.StatusCode < 300
	if response.StatusCode >= 400 && response.StatusCode < 500 && response.StatusCode != 429 {
		return s.finishDelivery(ctx, delivery, response.StatusCode, fmt.Errorf("webhook rejected with status %d", response.StatusCode), false)
	}
	return s.finishDelivery(ctx, delivery, response.StatusCode, nil, succeeded)
}

func (s *Service) finishDelivery(ctx context.Context, delivery WebhookDelivery, responseStatus int, failure error, succeeded bool) error {
	updates := map[string]any{
		"response_status": responseStatus, "updated_at": time.Now().UTC(),
	}
	if succeeded {
		now := time.Now().UTC()
		updates["status"] = "DELIVERED"
		updates["delivered_at"] = now
		updates["next_attempt_at"] = now
		updates["last_error"] = nil
	} else {
		message := "webhook delivery failed"
		if failure != nil {
			message = truncate(failure.Error(), 2000)
		}
		status := "FAILED"
		if delivery.Attempts >= s.options.MaxAttempts {
			status = "DEAD"
		}
		updates["status"] = status
		updates["last_error"] = message
		updates["next_attempt_at"] = time.Now().UTC().Add(backoff(delivery.Attempts))
	}
	return s.withPrivileged(ctx, func(tx *gorm.DB) error {
		return tx.Model(&WebhookDelivery{}).Where("id = ?", delivery.ID).Updates(updates).Error
	})
}

func (s *Service) withPrivileged(ctx context.Context, fn func(tx *gorm.DB) error) error {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			tx.Rollback()
			panic(recovered)
		}
	}()
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action, entityType, entityID string) {
	if s.audit == nil {
		return
	}
	tenantID := principal.TenantID
	userID := principal.UserID
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &tenantID, UserID: &userID, Action: action,
		Module: "integrations", EntityType: entityType, EntityID: entityID,
	})
}

func validateEndpointURL(raw string, allowInsecure bool) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("endpoint_url must be an absolute URL")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && allowInsecure {
		return nil
	}
	return fmt.Errorf("endpoint_url must use https")
}

func backoff(attempt int) time.Duration {
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 6 {
		shift = 6
	}
	return time.Duration(1<<shift) * time.Second
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
