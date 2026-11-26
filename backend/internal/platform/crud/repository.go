package crud

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Repository[T any] struct{ db *gorm.DB }

func New[T any](db *gorm.DB) *Repository[T] { return &Repository[T]{db: db} }

func (r *Repository[T]) Create(ctx context.Context, entity *T) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

func (r *Repository[T]) List(ctx context.Context, tenantID uuid.UUID, page query.Page, filters ...func(*gorm.DB) *gorm.DB) ([]T, int64, error) {
	db := r.db.WithContext(ctx).Model(new(T)).Where("tenant_id = ?", tenantID)
	for _, filter := range filters {
		db = filter(db)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []T
	err := db.Order(page.Sort + " " + page.Order).Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (r *Repository[T]) All(ctx context.Context, tenantID uuid.UUID, filters ...func(*gorm.DB) *gorm.DB) ([]T, error) {
	db := r.db.WithContext(ctx).Model(new(T)).Where("tenant_id = ?", tenantID)
	for _, filter := range filters {
		db = filter(db)
	}
	var items []T
	err := db.Order("name ASC").Find(&items).Error
	return items, err
}

func (r *Repository[T]) Get(ctx context.Context, tenantID, id uuid.UUID) (*T, error) {
	var entity T
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}
	return &entity, err
}

func (r *Repository[T]) Update(ctx context.Context, tenantID, id uuid.UUID, values map[string]interface{}) error {
	result := r.db.WithContext(ctx).Model(new(T)).Where("tenant_id = ? AND id = ?", tenantID, id).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *Repository[T]) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).Delete(new(T))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}
