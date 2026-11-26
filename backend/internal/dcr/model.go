package dcr

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DailyCallReport struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID  uuid.UUID      `gorm:"type:uuid;not null" json:"employee_id"`
	ReportDate  time.Time      `gorm:"type:date;not null" json:"report_date"`
	TotalVisits int            `gorm:"not null;default:0" json:"total_visits"`
	Status      string         `gorm:"not null;default:DRAFT" json:"status"`
	SubmittedAt *time.Time     `json:"submitted_at,omitempty"`
	ApprovedBy  *uuid.UUID     `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt  *time.Time     `json:"approved_at,omitempty"`
	Remarks     string         `json:"remarks,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DailyCallReport) TableName() string { return "daily_call_reports" }

type ProductDetail struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	DCRID       uuid.UUID `gorm:"type:uuid;not null" json:"dcr_id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Quantity    int       `gorm:"not null;default:0" json:"quantity"`
	SampleCount int       `gorm:"not null;default:0" json:"sample_count"`
	Outcome     string    `json:"outcome,omitempty"`
	Remarks     string    `json:"remarks,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ProductDetail) TableName() string { return "dcr_product_details" }
