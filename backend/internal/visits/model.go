package visits

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Visit struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID          uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	EmployeeID        uuid.UUID      `gorm:"type:uuid;not null" json:"employee_id"`
	TourPlanLineID    *uuid.UUID     `gorm:"type:uuid" json:"tour_plan_line_id,omitempty"`
	TerritoryID       *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	EntityType        string         `gorm:"not null" json:"entity_type"`
	EntityID          uuid.UUID      `gorm:"type:uuid;not null" json:"entity_id"`
	ClientReferenceID *uuid.UUID     `gorm:"type:uuid" json:"client_reference_id,omitempty"`
	VisitedAt         time.Time      `gorm:"not null" json:"visited_at"`
	CheckInLat        *float64       `json:"check_in_lat,omitempty"`
	CheckInLng        *float64       `json:"check_in_lng,omitempty"`
	CheckInAccuracy   *float64       `json:"check_in_accuracy,omitempty"`
	CheckOutLat       *float64       `json:"check_out_lat,omitempty"`
	CheckOutLng       *float64       `json:"check_out_lng,omitempty"`
	Status            string         `gorm:"not null;default:DRAFT" json:"status"`
	Remarks           string         `json:"remarks,omitempty"`
	SubmittedAt       *time.Time     `json:"submitted_at,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Visit) TableName() string { return "visits" }

type Attachment struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	VisitID     uuid.UUID `gorm:"type:uuid;not null" json:"visit_id"`
	FileName    string    `gorm:"not null" json:"file_name"`
	ContentType string    `json:"content_type,omitempty"`
	StorageKey  string    `gorm:"not null" json:"storage_key"`
	Checksum    string    `json:"checksum,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Attachment) TableName() string { return "visit_attachments" }
