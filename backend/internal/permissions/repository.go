package permissions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code        string    `gorm:"type:citext;uniqueIndex;not null" json:"code"`
	Module      string    `gorm:"not null" json:"module"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Permission) TableName() string { return "permissions" }

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	err := r.db.WithContext(ctx).Order("module, code").Find(&permissions).Error
	return permissions, err
}
