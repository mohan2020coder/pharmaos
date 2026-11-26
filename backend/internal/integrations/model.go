package integrations

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JSONObject map[string]any

func (m JSONObject) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *JSONObject) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*m = JSONObject{}
	case []byte:
		return json.Unmarshal(typed, m)
	case string:
		return json.Unmarshal([]byte(typed), m)
	default:
		return fmt.Errorf("unsupported JSONObject source %T", value)
	}
	return nil
}

type StringMap map[string]string

func (m StringMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *StringMap) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*m = StringMap{}
	case []byte:
		return json.Unmarshal(typed, m)
	case string:
		return json.Unmarshal([]byte(typed), m)
	default:
		return fmt.Errorf("unsupported StringMap source %T", value)
	}
	return nil
}

type IntegrationConnection struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	Name        string    `gorm:"not null" json:"name"`
	Kind        string    `gorm:"not null" json:"kind"`
	EndpointURL string    `json:"endpoint_url,omitempty"`
	SecretRef   string    `json:"secret_ref,omitempty"`
	Headers     StringMap `gorm:"type:jsonb;not null" json:"headers"`
	Status      string    `gorm:"not null;default:ACTIVE" json:"status"`
	CreatedBy   uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (IntegrationConnection) TableName() string { return "int_integration_connections" }

type WebhookDelivery struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	ConnectionID   uuid.UUID  `gorm:"type:uuid;not null" json:"connection_id"`
	EventType      string     `gorm:"not null" json:"event_type"`
	EventID        string     `gorm:"not null" json:"event_id"`
	Payload        JSONObject `gorm:"type:jsonb;not null" json:"payload"`
	Status         string     `gorm:"not null;default:PENDING" json:"status"`
	Attempts       int        `gorm:"not null;default:0" json:"attempts"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	LastError      string     `json:"last_error,omitempty"`
	ResponseStatus *int       `json:"response_status,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (WebhookDelivery) TableName() string { return "int_webhook_deliveries" }

type ConnectionCreateRequest struct {
	Name        string    `json:"name" validate:"required,max=120"`
	Kind        string    `json:"kind" validate:"required,oneof=WEBHOOK ERP ANALYTICS"`
	EndpointURL string    `json:"endpoint_url"`
	SecretRef   string    `json:"secret_ref" validate:"max=160"`
	Headers     StringMap `json:"headers"`
}

type ConnectionStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=ACTIVE DISABLED"`
}

type EventRequest struct {
	EventType string     `json:"event_type" validate:"required,max=120"`
	EventID   string     `json:"event_id" validate:"required,max=200"`
	Payload   JSONObject `json:"payload" validate:"required"`
}
