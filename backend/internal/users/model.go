package users

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID            uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Email               string         `gorm:"type:citext;not null" json:"email"`
	PasswordHash        string         `gorm:"not null" json:"-"`
	FirstName           string         `gorm:"not null" json:"first_name"`
	LastName            string         `gorm:"not null" json:"last_name"`
	Phone               *string        `json:"phone,omitempty"`
	Status              string         `gorm:"not null;default:ACTIVE" json:"status"`
	EmailVerified       bool           `gorm:"not null;default:false" json:"email_verified"`
	MustChangePassword  bool           `gorm:"not null;default:false" json:"must_change_password"`
	FailedLoginAttempts int            `gorm:"not null;default:0" json:"-"`
	LockedUntil         *time.Time     `json:"locked_until,omitempty"`
	LastLoginAt         *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }

type RoleAssignment struct {
	UserID    uuid.UUID  `json:"user_id"`
	RoleID    uuid.UUID  `json:"role_id"`
	ScopeType string     `json:"scope_type"`
	ScopeID   *uuid.UUID `json:"scope_id,omitempty"`
	ValidFrom time.Time  `json:"valid_from"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
}
