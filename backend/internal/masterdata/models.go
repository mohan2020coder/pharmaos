package masterdata

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/jsontypes"
)

type Doctor struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ExternalID     *string        `gorm:"type:citext" json:"external_id,omitempty"`
	Name           string         `gorm:"not null" json:"name"`
	Specialty      string         `json:"specialty,omitempty"`
	Qualification  string         `json:"qualification,omitempty"`
	RegistrationNo string         `json:"registration_no,omitempty"`
	Email          string         `gorm:"type:citext" json:"email,omitempty"`
	Phone          string         `json:"phone,omitempty"`
	Address        string         `json:"address,omitempty"`
	TerritoryID    *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	DivisionID     *uuid.UUID     `gorm:"type:uuid" json:"division_id,omitempty"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	Settings       jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Doctor) TableName() string { return "crm_doctors" }

type Chemist struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ExternalID  *string        `gorm:"type:citext" json:"external_id,omitempty"`
	Name        string         `gorm:"not null" json:"name"`
	OwnerName   string         `json:"owner_name,omitempty"`
	LicenseNo   string         `json:"license_no,omitempty"`
	Email       string         `gorm:"type:citext" json:"email,omitempty"`
	Phone       string         `json:"phone,omitempty"`
	Address     string         `json:"address,omitempty"`
	TerritoryID *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	DivisionID  *uuid.UUID     `gorm:"type:uuid" json:"division_id,omitempty"`
	IsActive    bool           `gorm:"not null;default:true" json:"is_active"`
	Settings    jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Chemist) TableName() string { return "crm_chemists" }

type Stockist struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ExternalID       *string        `gorm:"type:citext" json:"external_id,omitempty"`
	Name             string         `gorm:"not null" json:"name"`
	LicenseNo        string         `json:"license_no,omitempty"`
	GSTNumber        string         `json:"gst_number,omitempty"`
	Email            string         `gorm:"type:citext" json:"email,omitempty"`
	Phone            string         `json:"phone,omitempty"`
	Address          string         `json:"address,omitempty"`
	TerritoryID      *uuid.UUID     `gorm:"type:uuid" json:"territory_id,omitempty"`
	DivisionID       *uuid.UUID     `gorm:"type:uuid" json:"division_id,omitempty"`
	CreditLimit      float64        `gorm:"not null;default:0" json:"credit_limit"`
	PaymentTermsDays int            `gorm:"not null;default:0" json:"payment_terms_days"`
	IsActive         bool           `gorm:"not null;default:true" json:"is_active"`
	Settings         jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Stockist) TableName() string { return "crm_stockists" }

type DoctorTerritory struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	DoctorID    uuid.UUID  `gorm:"type:uuid;not null" json:"doctor_id"`
	TerritoryID uuid.UUID  `gorm:"type:uuid;not null" json:"territory_id"`
	ValidFrom   time.Time  `gorm:"type:date;not null" json:"valid_from"`
	ValidTo     *time.Time `gorm:"type:date" json:"valid_to,omitempty"`
	IsPrimary   bool       `gorm:"not null;default:true" json:"is_primary"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (DoctorTerritory) TableName() string { return "crm_doctor_territories" }

type ProductCategory struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Code             string         `gorm:"type:citext;not null" json:"code"`
	Name             string         `gorm:"not null" json:"name"`
	ParentCategoryID *uuid.UUID     `gorm:"type:uuid" json:"parent_category_id,omitempty"`
	IsActive         bool           `gorm:"not null;default:true" json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ProductCategory) TableName() string { return "cat_product_categories" }

type Product struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID     uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	CategoryID   *uuid.UUID     `gorm:"type:uuid" json:"category_id,omitempty"`
	Code         string         `gorm:"type:citext;not null" json:"code"`
	Name         string         `gorm:"not null" json:"name"`
	Brand        string         `json:"brand,omitempty"`
	Manufacturer string         `json:"manufacturer,omitempty"`
	Composition  string         `json:"composition,omitempty"`
	PackSize     string         `json:"pack_size,omitempty"`
	Unit         string         `json:"unit,omitempty"`
	MRP          float64        `gorm:"not null;default:0" json:"mrp"`
	IsActive     bool           `gorm:"not null;default:true" json:"is_active"`
	Settings     jsontypes.Map  `gorm:"type:jsonb;not null;default:'{}'" json:"settings"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Product) TableName() string { return "cat_products" }
