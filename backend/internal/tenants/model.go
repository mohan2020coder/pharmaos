package tenants

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
)

type Tenant struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code        string         `gorm:"type:citext;uniqueIndex;not null" json:"code"`
	Name        string         `gorm:"not null" json:"name"`
	TenantType  string         `gorm:"not null" json:"tenant_type"`
	Status      string         `gorm:"not null;default:ACTIVE" json:"status"`
	IsPlatform  bool           `gorm:"not null;default:false" json:"is_platform"`
	CountryCode *string        `gorm:"type:char(2)" json:"country_code,omitempty"`
	Timezone    string         `gorm:"not null;default:UTC" json:"timezone"`
	Currency    string         `gorm:"column:currency_code;type:char(3);not null;default:USD" json:"currency_code"`
	Settings    jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Tenant) TableName() string { return "tenants" }

type Setting struct {
	TenantID  uuid.UUID     `gorm:"type:uuid;primaryKey" json:"tenant_id"`
	Key       string        `gorm:"primaryKey" json:"key"`
	Value     jsontypes.Map `gorm:"type:jsonb;not null" json:"value"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (Setting) TableName() string { return "tenant_settings" }
