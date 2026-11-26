package masterdata

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/crud"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Service[T any] struct{ repo *crud.Repository[T] }

func NewService[T any](db *gorm.DB) *Service[T] {
	return &Service[T]{repo: crud.New[T](db)}
}

func (s *Service[T]) Create(ctx context.Context, entity *T) error {
	return s.repo.Create(ctx, entity)
}

func (s *Service[T]) List(ctx context.Context, tenantID uuid.UUID, page query.Page, searchColumns []string, filters ...func(*gorm.DB) *gorm.DB) ([]T, int64, error) {
	if page.Search != "" && len(searchColumns) > 0 {
		term := "%" + page.Search + "%"
		filters = append(filters, func(db *gorm.DB) *gorm.DB {
			group := db.Where(searchColumns[0]+" ILIKE ?", term)
			for _, column := range searchColumns[1:] {
				group = group.Or(column+" ILIKE ?", term)
			}
			return group
		})
	}
	return s.repo.List(ctx, tenantID, page, filters...)
}

func (s *Service[T]) Get(ctx context.Context, tenantID, id uuid.UUID) (*T, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service[T]) Update(ctx context.Context, tenantID, id uuid.UUID, values map[string]interface{}) error {
	return s.repo.Update(ctx, tenantID, id, values)
}

func (s *Service[T]) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.Delete(ctx, tenantID, id)
}
