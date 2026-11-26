package crm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Target struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	EntityType   string         `gorm:"not null" json:"entity_type"`
	EntityID     uuid.UUID      `gorm:"type:uuid;not null" json:"entity_id"`
	EmployeeID   uuid.UUID      `gorm:"type:uuid;not null" json:"employee_id"`
	TerritoryID  *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	PeriodStart  time.Time      `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd    time.Time      `gorm:"type:date;not null" json:"period_end"`
	TargetVisits int            `gorm:"not null;default:0" json:"target_visits"`
	TargetSales  float64        `gorm:"not null;default:0" json:"target_sales"`
	Status       string         `gorm:"not null;default:ACTIVE" json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Target) TableName() string { return "crm_targets" }
