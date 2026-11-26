package distribution

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DistributionLink struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID            uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	DistributorTenantID uuid.UUID  `gorm:"type:uuid;not null" json:"distributor_tenant_id"`
	Status              string     `gorm:"not null;default:ACTIVE" json:"status"`
	EffectiveFrom       time.Time  `gorm:"type:date;not null" json:"effective_from"`
	EffectiveTo         *time.Time `gorm:"type:date" json:"effective_to,omitempty"`
	CreditLimit         float64    `gorm:"not null;default:0" json:"credit_limit"`
	PaymentTermsDays    int        `gorm:"not null;default:0" json:"payment_terms_days"`
	CreatedBy           *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func (DistributionLink) TableName() string { return "dist_company_distributor_links" }

type Distributor struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Code      string         `gorm:"type:citext;not null" json:"code"`
	Name      string         `gorm:"not null" json:"name"`
	LicenseNo string         `json:"license_no,omitempty"`
	Email     string         `gorm:"type:citext" json:"email,omitempty"`
	Phone     string         `json:"phone,omitempty"`
	Address   string         `json:"address,omitempty"`
	Status    string         `gorm:"not null;default:ACTIVE" json:"status"`
	Settings  []byte         `gorm:"type:jsonb" json:"settings,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Distributor) TableName() string { return "dist_distributors" }

type Order struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID            uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	DistributorTenantID uuid.UUID      `gorm:"type:uuid;not null" json:"distributor_tenant_id"`
	LinkID              uuid.UUID      `gorm:"type:uuid;not null" json:"link_id"`
	OrderNumber         string         `gorm:"type:citext;not null" json:"order_number"`
	Status              string         `gorm:"not null;default:DRAFT" json:"status"`
	CurrencyCode        string         `gorm:"not null;default:USD" json:"currency_code"`
	TotalAmount         float64        `gorm:"not null;default:0" json:"total_amount"`
	IdempotencyKey      uuid.UUID      `gorm:"type:uuid;not null" json:"idempotency_key"`
	RequestedBy         *uuid.UUID     `gorm:"type:uuid" json:"requested_by,omitempty"`
	ApprovedBy          *uuid.UUID     `gorm:"type:uuid" json:"approved_by,omitempty"`
	OrderDate           time.Time      `gorm:"type:date;not null" json:"order_date"`
	RequiredBy          *time.Time     `gorm:"type:date" json:"required_by,omitempty"`
	SubmittedAt         *time.Time     `json:"submitted_at,omitempty"`
	ApprovedAt          *time.Time     `json:"approved_at,omitempty"`
	CancelledAt         *time.Time     `json:"cancelled_at,omitempty"`
	Remarks             string         `json:"remarks,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Order) TableName() string { return "dist_orders" }

type OrderItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	UnitPrice float64   `gorm:"not null;default:0" json:"unit_price"`
	LineTotal float64   `gorm:"not null;default:0" json:"line_total"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (OrderItem) TableName() string { return "dist_order_items" }

type Inventory struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductID        uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	WarehouseCode    string    `gorm:"not null;default:MAIN" json:"warehouse_code"`
	QuantityOnHand   int       `gorm:"not null;default:0" json:"quantity_on_hand"`
	QuantityReserved int       `gorm:"not null;default:0" json:"quantity_reserved"`
	Version          int64     `gorm:"not null;default:1" json:"version"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (Inventory) TableName() string { return "dist_inventory" }

type InventoryMovement struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductID      uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	WarehouseCode  string     `gorm:"not null;default:MAIN" json:"warehouse_code"`
	MovementType   string     `gorm:"not null" json:"movement_type"`
	QuantityDelta  int        `gorm:"not null" json:"quantity_delta"`
	ReferenceType  string     `json:"reference_type,omitempty"`
	ReferenceID    *uuid.UUID `gorm:"type:uuid" json:"reference_id,omitempty"`
	IdempotencyKey uuid.UUID  `gorm:"type:uuid;not null" json:"idempotency_key"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	OccurredAt     time.Time  `json:"occurred_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (InventoryMovement) TableName() string { return "dist_inventory_movements" }

type SecondarySale struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	OrderID         *uuid.UUID `gorm:"type:uuid" json:"order_id,omitempty"`
	CompanyTenantID *uuid.UUID `gorm:"type:uuid" json:"company_tenant_id,omitempty"`
	ProductID       uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	Quantity        int        `gorm:"not null" json:"quantity"`
	UnitPrice       float64    `gorm:"not null;default:0" json:"unit_price"`
	TotalAmount     float64    `gorm:"not null;default:0" json:"total_amount"`
	SaleDate        time.Time  `gorm:"type:date;not null" json:"sale_date"`
	InvoiceNo       string     `gorm:"type:citext" json:"invoice_no,omitempty"`
	CustomerCode    string     `json:"customer_code,omitempty"`
	IdempotencyKey  uuid.UUID  `gorm:"type:uuid;not null" json:"idempotency_key"`
	CreatedBy       *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (SecondarySale) TableName() string { return "dist_secondary_sales" }

type LinkCreateRequest struct {
	ID                  *uuid.UUID `json:"id"`
	DistributorTenantID uuid.UUID  `json:"distributor_tenant_id" validate:"required,uuid"`
	Status              string     `json:"status"`
	EffectiveFrom       string     `json:"effective_from"`
	EffectiveTo         string     `json:"effective_to"`
	CreditLimit         float64    `json:"credit_limit"`
	PaymentTermsDays    int        `json:"payment_terms_days"`
}

type LinkStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=ACTIVE SUSPENDED ENDED"`
}

type DistributorCreateRequest struct {
	ID        *uuid.UUID `json:"id"`
	Code      string     `json:"code" validate:"required"`
	Name      string     `json:"name" validate:"required"`
	LicenseNo string     `json:"license_no"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	Address   string     `json:"address"`
	Status    string     `json:"status"`
}

type OrderItemRequest struct {
	ID        *uuid.UUID `json:"id"`
	ProductID uuid.UUID  `json:"product_id" validate:"required,uuid"`
	Quantity  int        `json:"quantity" validate:"required,min=1"`
	UnitPrice float64    `json:"unit_price" validate:"min=0"`
}

type OrderCreateRequest struct {
	ID                  *uuid.UUID         `json:"id"`
	CompanyTenantID     *uuid.UUID         `json:"company_tenant_id"`
	DistributorTenantID uuid.UUID          `json:"distributor_tenant_id" validate:"required,uuid"`
	OrderNumber         string             `json:"order_number"`
	CurrencyCode        string             `json:"currency_code"`
	OrderDate           string             `json:"order_date"`
	RequiredBy          string             `json:"required_by"`
	Remarks             string             `json:"remarks"`
	Items               []OrderItemRequest `json:"items" validate:"required,min=1,dive"`
}

type OrderDecisionRequest struct {
	Remarks string `json:"remarks"`
}

type SecondarySaleRequest struct {
	ID              *uuid.UUID `json:"id"`
	CompanyTenantID *uuid.UUID `json:"company_tenant_id"`
	OrderID         *uuid.UUID `json:"order_id"`
	ProductID       uuid.UUID  `json:"product_id" validate:"required,uuid"`
	Quantity        int        `json:"quantity" validate:"required,min=1"`
	UnitPrice       float64    `json:"unit_price" validate:"min=0"`
	SaleDate        string     `json:"sale_date"`
	InvoiceNo       string     `json:"invoice_no"`
	CustomerCode    string     `json:"customer_code"`
}

type InventoryAdjustmentRequest struct {
	ID            *uuid.UUID `json:"id"`
	ProductID     uuid.UUID  `json:"product_id" validate:"required,uuid"`
	WarehouseCode string     `json:"warehouse_code"`
	MovementType  string     `json:"movement_type" validate:"required,oneof=RECEIPT SALE ADJUSTMENT"`
	QuantityDelta int        `json:"quantity_delta" validate:"required,ne=0"`
	ReferenceType string     `json:"reference_type"`
	ReferenceID   *uuid.UUID `json:"reference_id"`
	OccurredAt    string     `json:"occurred_at"`
}
