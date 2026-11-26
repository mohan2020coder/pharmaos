package auth

import (
	"time"

	"github.com/google/uuid"
)

type AuthUser struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID            uuid.UUID `gorm:"type:uuid;not null"`
	Email               string    `gorm:"type:citext;not null"`
	PasswordHash        string    `gorm:"not null"`
	FirstName           string
	LastName            string
	Phone               *string
	Status              string
	FailedLoginAttempts int
	LockedUntil         *time.Time
	MustChangePassword  bool
	EmailVerified       bool
	LastLoginAt         *time.Time
	DeletedAt           *time.Time `gorm:"column:deleted_at"`
}

func (AuthUser) TableName() string { return "users" }

type RoleRecord struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	TenantID     *uuid.UUID
	Code         string
	DataScope    string
	IsPrivileged bool
}

type Session struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID `gorm:"type:uuid;not null"`
	TenantID         uuid.UUID `gorm:"type:uuid;not null"`
	FamilyID         uuid.UUID `gorm:"type:uuid;not null"`
	RefreshTokenHash string    `gorm:"not null"`
	IPAddress        *string
	UserAgent        *string
	DeviceID         *string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
}

func (Session) TableName() string { return "auth_sessions" }

type PasswordReset struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	TokenHash string    `gorm:"not null"`
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (PasswordReset) TableName() string { return "password_reset_tokens" }

type UserRoleGrant struct {
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	RoleID       uuid.UUID `gorm:"type:uuid;not null"`
	ScopeType    string
	ScopeID      *uuid.UUID
	ValidFrom    time.Time
	ValidTo      *time.Time
	Code         string
	DataScope    string
	IsPrivileged bool
	TenantID     *uuid.UUID
}
