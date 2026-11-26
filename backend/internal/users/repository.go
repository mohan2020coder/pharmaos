package users

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) List(ctx context.Context, page query.Page, tenantID uuid.UUID, all bool) ([]User, int64, error) {
	db := r.db.WithContext(ctx).Model(&User{})
	if !all {
		db = db.Where("tenant_id = ?", tenantID)
	}
	if page.Search != "" {
		term := "%" + page.Search + "%"
		db = db.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", term, term, term)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := db.Order(page.Sort + " " + page.Order).Limit(page.Size).Offset(page.Offset()).Find(&users).Error
	return users, total, err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) Update(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Model(user).Select("first_name", "last_name", "phone", "status", "email_verified", "must_change_password").Updates(user).Error
}

func (r *Repository) Deactivate(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`UPDATE users SET status = 'INACTIVE' WHERE id = ?`, id).Error
}

func (r *Repository) AssignRole(ctx context.Context, grant RoleAssignment) error {
	return r.db.WithContext(ctx).Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from, valid_to)
		VALUES (?, ?, ?, COALESCE(?, '00000000-0000-0000-0000-000000000000'), ?, ?) ON CONFLICT (user_id, role_id, scope_type, scope_id) DO UPDATE
		SET valid_from = EXCLUDED.valid_from, valid_to = EXCLUDED.valid_to`,
		grant.UserID, grant.RoleID, grant.ScopeType, grant.ScopeID, grant.ValidFrom, grant.ValidTo).Error
}

func (r *Repository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID, scopeType string, scopeID *uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`DELETE FROM user_roles WHERE user_id = ? AND role_id = ? AND scope_type = ?
		AND scope_id = COALESCE(?, '00000000-0000-0000-0000-000000000000')`,
		userID, roleID, scopeType, scopeID).Error
}

func (r *Repository) RolesForUser(ctx context.Context, userID uuid.UUID) ([]RoleAssignment, error) {
	var grants []RoleAssignment
	err := r.db.WithContext(ctx).Raw(`SELECT user_id, role_id, scope_type, scope_id, valid_from, valid_to FROM user_roles WHERE user_id = ?`, userID).Scan(&grants).Error
	return grants, err
}

func (r *Repository) RoleBelongsToTenant(ctx context.Context, roleID, tenantID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Raw(`SELECT count(*) FROM roles WHERE id = ? AND (tenant_id IS NULL OR tenant_id = ?) AND deleted_at IS NULL`, roleID, tenantID).Scan(&count).Error
	return count > 0, err
}
