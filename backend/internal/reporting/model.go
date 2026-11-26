package reporting

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value any) error {
	switch typed := value.(type) {
	case nil:
		*m = JSONMap{}
	case []byte:
		return json.Unmarshal(typed, m)
	case string:
		return json.Unmarshal([]byte(typed), m)
	default:
		return fmt.Errorf("unsupported JSONMap source %T", value)
	}
	return nil
}

type Period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type VisitMetrics struct {
	Total           int64 `json:"total"`
	Submitted       int64 `json:"submitted"`
	Draft           int64 `json:"draft"`
	UniqueCustomers int64 `json:"unique_customers"`
}

type DailyCallReportMetrics struct {
	Total     int64 `json:"total"`
	Submitted int64 `json:"submitted"`
	Approved  int64 `json:"approved"`
}

type FieldActivityReport struct {
	Period           Period                 `json:"period"`
	Scope            string                 `json:"scope"`
	Visits           VisitMetrics           `json:"visits"`
	DailyCallReports DailyCallReportMetrics `json:"daily_call_reports"`
}

type OrderMetrics struct {
	Total     int64   `json:"total"`
	Submitted int64   `json:"submitted"`
	Approved  int64   `json:"approved"`
	Cancelled int64   `json:"cancelled"`
	Value     float64 `json:"value"`
}

type InventoryMetrics struct {
	SKUCount int64 `json:"sku_count"`
	OnHand   int64 `json:"on_hand"`
	Reserved int64 `json:"reserved"`
}

type SecondarySalesMetrics struct {
	Records int64   `json:"records"`
	Units   int64   `json:"units"`
	Value   float64 `json:"value"`
}

type DistributionSummaryReport struct {
	Period         Period                `json:"period"`
	Orders         OrderMetrics          `json:"orders"`
	Inventory      InventoryMetrics      `json:"inventory"`
	SecondarySales SecondarySalesMetrics `json:"secondary_sales"`
	ActiveLinks    int64                 `json:"active_links"`
}

type ReportDefinition struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	Code        string    `gorm:"not null" json:"code"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description,omitempty"`
	ReportType  string    `gorm:"not null" json:"report_type"`
	Config      JSONMap   `gorm:"type:jsonb;not null" json:"config"`
	CreatedBy   uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ReportDefinition) TableName() string { return "rpt_report_definitions" }

type ReportSnapshot struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	DefinitionID *uuid.UUID `gorm:"type:uuid" json:"definition_id,omitempty"`
	PeriodFrom   *time.Time `gorm:"type:date" json:"period_from,omitempty"`
	PeriodTo     *time.Time `gorm:"type:date" json:"period_to,omitempty"`
	Payload      JSONMap    `gorm:"type:jsonb;not null" json:"payload"`
	CreatedBy    uuid.UUID  `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (ReportSnapshot) TableName() string { return "rpt_report_snapshots" }

type AuditEntry struct {
	ID         int64      `json:"id"`
	TenantID   *uuid.UUID `json:"tenant_id,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action"`
	Module     string     `json:"module"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (AuditEntry) TableName() string { return "audit_logs" }

type ReportDefinitionRequest struct {
	Code        string  `json:"code" validate:"required,max=80"`
	Name        string  `json:"name" validate:"required,max=160"`
	Description string  `json:"description"`
	ReportType  string  `json:"report_type" validate:"required,oneof=FIELD_ACTIVITY DISTRIBUTION_SUMMARY AUDIT_TRAIL"`
	Config      JSONMap `json:"config"`
}

type SnapshotRequest struct {
	DefinitionID *uuid.UUID `json:"definition_id"`
	ReportType   string     `json:"report_type" validate:"required,oneof=FIELD_ACTIVITY DISTRIBUTION_SUMMARY AUDIT_TRAIL"`
	From         string     `json:"from"`
	To           string     `json:"to"`
	Payload      JSONMap    `json:"payload" validate:"required"`
}
