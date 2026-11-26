package tours

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TourPlan struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID  uuid.UUID      `gorm:"type:uuid;not null" json:"employee_id"`
	Title       string         `gorm:"not null" json:"title"`
	PeriodStart time.Time      `gorm:"type:date;not null" json:"period_start"`
	PeriodEnd   time.Time      `gorm:"type:date;not null" json:"period_end"`
	Status      string         `gorm:"not null;default:DRAFT" json:"status"`
	SubmittedAt *time.Time     `json:"submitted_at,omitempty"`
	ApprovedBy  *uuid.UUID     `gorm:"type:uuid" json:"approved_by,omitempty"`
	ApprovedAt  *time.Time     `json:"approved_at,omitempty"`
	Comments    string         `json:"comments,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TourPlan) TableName() string { return "tour_plans" }

type TourPlanLine struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	TourPlanID  uuid.UUID  `gorm:"type:uuid;not null" json:"tour_plan_id"`
	PlannedDate time.Time  `gorm:"type:date;not null" json:"planned_date"`
	TerritoryID *uuid.UUID `gorm:"type:uuid" json:"territory_id,omitempty"`
	EntityType  string     `gorm:"not null" json:"entity_type"`
	EntityID    uuid.UUID  `gorm:"type:uuid;not null" json:"entity_id"`
	Purpose     string     `json:"purpose,omitempty"`
	Sequence    int        `gorm:"not null;default:1" json:"sequence"`
	Status      string     `gorm:"not null;default:PLANNED" json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (TourPlanLine) TableName() string { return "tour_plan_lines" }
