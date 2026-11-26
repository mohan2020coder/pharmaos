package organization

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Branch struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	DivisionID  *uuid.UUID     `gorm:"type:uuid" json:"division_id,omitempty"`
	Code        string         `gorm:"not null" json:"code"`
	Name        string         `gorm:"not null" json:"name"`
	Address     string         `json:"address"`
	City        string         `json:"city"`
	State       string         `json:"state"`
	PostalCode  string         `json:"postal_code"`
	CountryCode *string        `gorm:"type:char(2)" json:"country_code,omitempty"`
	Status      string         `gorm:"not null;default:ACTIVE" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Branch) TableName() string { return "branches" }
