package divisions

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
)

type Division struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID      uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Code          string         `gorm:"not null" json:"code"`
	Name          string         `gorm:"not null" json:"name"`
	Description   string         `json:"description"`
	Status        string         `gorm:"not null;default:ACTIVE" json:"status"`
	ManagerUserID *uuid.UUID     `gorm:"type:uuid" json:"manager_user_id,omitempty"`
	Settings      jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Division) TableName() string { return "business_divisions" }
