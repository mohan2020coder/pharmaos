package roles

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     *uuid.UUID     `gorm:"type:uuid" json:"tenant_id,omitempty"`
	Code         string         `gorm:"type:citext;not null" json:"code"`
	Name         string         `gorm:"not null" json:"name"`
	Description  string         `json:"description"`
	IsSystem     bool           `gorm:"not null;default:false" json:"is_system"`
	IsPrivileged bool           `gorm:"not null;default:false" json:"is_privileged"`
	DataScope    string         `gorm:"not null;default:TENANT" json:"data_scope"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Role) TableName() string { return "roles" }

type RoleWithPermissions struct {
	Role
	Permissions []string `json:"permissions"`
}
