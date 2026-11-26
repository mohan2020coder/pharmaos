package roles

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, page query.Page) ([]Role, int64, error) {
	db := r.db.WithContext(ctx).Model(&Role{})
	if page.Search != "" {
		term := "%" + page.Search + "%"
		db = db.Where("name ILIKE ? OR code ILIKE ?", term, term)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var roles []Role
	err := db.Order("is_system DESC, name ASC").Limit(page.Size).Offset(page.Offset()).Find(&roles).Error
	return roles, total, err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	var role Role
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) Create(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

func (r *Repository) Update(ctx context.Context, role *Role) error {
	return r.db.WithContext(ctx).Model(role).Select("name", "description", "data_scope", "is_privileged").Updates(role).Error
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Role{}, "id = ?", id).Error
}

func (r *Repository) Permissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`SELECT p.code FROM permissions p JOIN role_permissions rp ON rp.permission_id = p.id WHERE rp.role_id = ? ORDER BY p.code`, roleID).Scan(&codes).Error
	return codes, err
}

func (r *Repository) PermissionIDs(ctx context.Context, codes []string) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if len(codes) == 0 {
		return ids, nil
	}
	err := r.db.WithContext(ctx).Raw(`SELECT id FROM permissions WHERE code = ANY(?)`, codes).Scan(&ids).Error
	return ids, err
}

func (r *Repository) ReplacePermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	tx := r.db.WithContext(ctx)
	if err := tx.Exec(`DELETE FROM role_permissions WHERE role_id = ?`, roleID).Error; err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	values := make([]interface{}, 0, len(permissionIDs)*2)
	for _, id := range permissionIDs {
		values = append(values, roleID, id)
	}
	return tx.Exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES `+placeholders(len(permissionIDs)), values...).Error
}

func placeholders(count int) string {
	out := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			out += ","
		}
		out += "(?,?)"
	}
	return out
}

func IsNotFound(err error) bool { return errors.Is(err, gorm.ErrRecordNotFound) }
