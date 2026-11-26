package tenants

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Create(ctx context.Context, tenant *Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

func (r *Repository) List(ctx context.Context, page query.Page, tenantIDs []uuid.UUID, all bool) ([]Tenant, int64, error) {
	db := r.db.WithContext(ctx).Model(&Tenant{})
	if !all {
		if len(tenantIDs) == 0 {
			return []Tenant{}, 0, nil
		}
		db = db.Where("id IN ?", tenantIDs)
	}
	if page.Search != "" {
		search := "%" + page.Search + "%"
		db = db.Where("name ILIKE ? OR code ILIKE ?", search, search)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var tenants []Tenant
	err := db.Order(page.Sort + " " + page.Order).
		Limit(page.Size).Offset(page.Offset()).Find(&tenants).Error
	return tenants, total, err
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var tenant Tenant
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *Repository) FindByCode(ctx context.Context, code string) (*Tenant, error) {
	var tenant Tenant
	if err := r.db.WithContext(ctx).Where("code = ?", code).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *Repository) Update(ctx context.Context, tenant *Tenant) error {
	return r.db.WithContext(ctx).Model(tenant).Select("name", "status", "timezone", "currency_code", "settings").Updates(tenant).Error
}

func (r *Repository) UpsertSetting(ctx context.Context, tenantID uuid.UUID, key string, value map[string]any) error {
	setting := Setting{TenantID: tenantID, Key: key, Value: jsontypes.Map(value)}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_at": gorm.Expr("now()")}),
	}).Create(&setting).Error
}
