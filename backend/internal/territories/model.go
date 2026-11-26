package territories

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Region struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID   uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	DivisionID uuid.UUID      `gorm:"type:uuid;not null" json:"division_id"`
	Code       string         `gorm:"not null" json:"code"`
	Name       string         `gorm:"not null" json:"name"`
	Status     string         `gorm:"not null;default:ACTIVE" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Region) TableName() string { return "regions" }

type Territory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID   uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	DivisionID uuid.UUID      `gorm:"type:uuid;not null" json:"division_id"`
	RegionID   uuid.UUID      `gorm:"type:uuid;not null" json:"region_id"`
	BranchID   *uuid.UUID     `gorm:"type:uuid" json:"branch_id,omitempty"`
	Code       string         `gorm:"not null" json:"code"`
	Name       string         `gorm:"not null" json:"name"`
	City       string         `json:"city"`
	Status     string         `gorm:"not null;default:ACTIVE" json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Territory) TableName() string { return "territories" }
