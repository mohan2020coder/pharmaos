package distribution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/auth"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
)

type OrderDetail struct {
	Order
	Items []OrderItem `json:"items"`
}

type CompanyOrderRef struct {
	CompanyTenantID uuid.UUID `gorm:"column:company_tenant_id"`
}

type Service struct {
	db    *gorm.DB
	audit *audit.Service
}

func NewService(db *gorm.DB, auditService *audit.Service) *Service {
	return &Service{db: db, audit: auditService}
}

func (s *Service) CreateLink(ctx context.Context, principal auth.Principal, request LinkCreateRequest) (*DistributionLink, error) {
	if !principal.HasPermission("link.manage") {
		return nil, apperrors.ErrForbidden
	}
	effectiveFrom, err := parseDateOrDefault(request.EffectiveFrom)
	if err != nil {
		return nil, err
	}
	if effectiveFrom.IsZero() {
		effectiveFrom = today()
	}
	var effectiveTo *time.Time
	if request.EffectiveTo != "" {
		parsed, err := parseDate(request.EffectiveTo)
		if err != nil {
			return nil, err
		}
		effectiveTo = &parsed
		if effectiveTo.Before(effectiveFrom) {
			return nil, apperrors.New("INVALID_DATE_RANGE", "effective_to must not precede effective_from", 422)
		}
	}
	status := request.Status
	if status == "" {
		status = "ACTIVE"
	}
	if !validStatus(status, "ACTIVE", "SUSPENDED", "ENDED") {
		return nil, apperrors.New("INVALID_STATUS", "Unsupported link status", 422)
	}
	id := uuid.New()
	if request.ID != nil {
		id = *request.ID
	}
	link := &DistributionLink{
		ID: id, TenantID: principal.TenantID, DistributorTenantID: request.DistributorTenantID,
		Status: status, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo,
		CreditLimit: request.CreditLimit, PaymentTermsDays: request.PaymentTermsDays,
		CreatedBy: &principal.UserID,
	}
	if err := s.withTenant(ctx, principal.TenantID, func(tx *gorm.DB) error { return tx.Create(link).Error }); err != nil {
		if isUniqueError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "DISTRIBUTION_LINK_CREATE", "DISTRIBUTION_LINK", link.ID, link)
	return link, nil
}

func (s *Service) ListLinks(ctx context.Context, principal auth.Principal, page query.Page, status string) ([]DistributionLink, int64, error) {
	db := s.db.WithContext(ctx).Model(&DistributionLink{})
	if !principal.IsPrivileged {
		db = db.Where("tenant_id = ? OR distributor_tenant_id = ?", principal.TenantID, principal.TenantID)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []DistributionLink
	err := db.Order("updated_at DESC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) UpdateLinkStatus(ctx context.Context, principal auth.Principal, id uuid.UUID, request LinkStatusRequest) error {
	if !principal.HasPermission("link.manage") {
		return apperrors.ErrForbidden
	}
	result := s.db.WithContext(ctx).Model(&DistributionLink{}).
		Where("id = ? AND tenant_id = ?", id, principal.TenantID).
		Update("status", request.Status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrNotFound
	}
	s.record(ctx, principal, "DISTRIBUTION_LINK_STATUS", "DISTRIBUTION_LINK", id, request)
	return nil
}

func (s *Service) CreateDistributor(ctx context.Context, principal auth.Principal, request DistributorCreateRequest) (*Distributor, error) {
	if !principal.HasPermission("distributor.manage") {
		return nil, apperrors.ErrForbidden
	}
	status := request.Status
	if status == "" {
		status = "ACTIVE"
	}
	if !validStatus(status, "ACTIVE", "INACTIVE") {
		return nil, apperrors.New("INVALID_STATUS", "Unsupported distributor status", 422)
	}
	id := uuid.New()
	if request.ID != nil {
		id = *request.ID
	}
	record := &Distributor{
		ID: id, TenantID: principal.TenantID, Code: request.Code, Name: request.Name,
		LicenseNo: request.LicenseNo, Email: request.Email, Phone: request.Phone,
		Address: request.Address, Status: status,
	}
	if err := s.withTenant(ctx, principal.TenantID, func(tx *gorm.DB) error { return tx.Create(record).Error }); err != nil {
		if isUniqueError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "DISTRIBUTOR_CREATE", "DISTRIBUTOR", record.ID, record)
	return record, nil
}

func (s *Service) ListDistributors(ctx context.Context, principal auth.Principal, page query.Page, status string) ([]Distributor, int64, error) {
	db := s.db.WithContext(ctx).Model(&Distributor{}).Where("deleted_at IS NULL")
	if !principal.IsPrivileged {
		db = db.Where(`tenant_id = ? OR EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.distributor_tenant_id = dist_distributors.tenant_id
			  AND l.tenant_id = ?
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE))`, principal.TenantID, principal.TenantID)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Distributor
	err := db.Order("name ASC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) GetDistributor(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Distributor, error) {
	var record Distributor
	err := s.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if !principal.IsPrivileged && record.TenantID != principal.TenantID {
		_, err := s.activeLink(ctx, principal.TenantID, record.TenantID)
		if err != nil {
			return nil, apperrors.ErrNotFound
		}
	}
	return &record, nil
}

func (s *Service) CreateOrder(ctx context.Context, principal auth.Principal, request OrderCreateRequest, idempotencyKey uuid.UUID) (*OrderDetail, error) {
	if !principal.HasPermission("order.manage") {
		return nil, apperrors.ErrForbidden
	}
	companyTenantID, distributorTenantID, link, err := s.resolveLink(ctx, principal, request.CompanyTenantID, request.DistributorTenantID)
	if err != nil {
		return nil, err
	}
	orderDate, err := parseDateOrDefault(request.OrderDate)
	if err != nil {
		return nil, err
	}
	var requiredBy *time.Time
	if request.RequiredBy != "" {
		parsed, err := parseDate(request.RequiredBy)
		if err != nil {
			return nil, err
		}
		requiredBy = &parsed
	}
	orderID := uuid.New()
	if request.ID != nil {
		orderID = *request.ID
	}
	existingID, replay, err := s.claimIdempotency(ctx, companyTenantID, idempotencyKey, "order.create")
	if err != nil {
		return nil, err
	}
	if replay {
		if existingID == nil {
			return nil, apperrors.ErrConflict
		}
		return s.GetOrder(ctx, principal, *existingID)
	}
	currency := strings.ToUpper(strings.TrimSpace(request.CurrencyCode))
	if currency == "" {
		currency = "USD"
	}
	orderNumber := strings.TrimSpace(request.OrderNumber)
	if orderNumber == "" {
		orderNumber = fmt.Sprintf("ORD-%d-%s", time.Now().UTC().Unix(), orderID.String()[:8])
	}
	order := &Order{
		ID: orderID, TenantID: companyTenantID, DistributorTenantID: distributorTenantID,
		LinkID: link.ID, OrderNumber: orderNumber, Status: "DRAFT", CurrencyCode: currency,
		IdempotencyKey: idempotencyKey, RequestedBy: &principal.UserID,
		OrderDate: orderDate, RequiredBy: requiredBy, Remarks: request.Remarks,
	}
	err = s.withTenant(ctx, companyTenantID, func(tx *gorm.DB) error {
		items, total, err := s.buildOrderItems(tx, companyTenantID, orderID, request.Items)
		if err != nil {
			return err
		}
		order.TotalAmount = total
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return s.completeIdempotency(tx, companyTenantID, idempotencyKey, orderID)
	})
	if err != nil {
		s.failIdempotency(ctx, companyTenantID, idempotencyKey, err.Error())
		if isUniqueError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "ORDER_CREATE", "ORDER", order.ID, order)
	return s.GetOrder(ctx, principal, order.ID)
}

func (s *Service) GetOrder(ctx context.Context, principal auth.Principal, id uuid.UUID) (*OrderDetail, error) {
	var order Order
	err := s.orderQuery(ctx, principal).Where("dist_orders.id = ? AND dist_orders.deleted_at IS NULL", id).First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var items []OrderItem
	if err := s.orderItemQuery(ctx, principal).Where("dist_order_items.order_id = ?", id).Find(&items).Error; err != nil {
		return nil, err
	}
	return &OrderDetail{Order: order, Items: items}, nil
}

func (s *Service) ListOrders(ctx context.Context, principal auth.Principal, page query.Page, status string, distributorTenantID uuid.UUID) ([]Order, int64, error) {
	db := s.orderQuery(ctx, principal).Where("dist_orders.deleted_at IS NULL")
	if status != "" {
		db = db.Where("dist_orders.status = ?", status)
	}
	if distributorTenantID != uuid.Nil {
		db = db.Where("dist_orders.distributor_tenant_id = ?", distributorTenantID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Order
	err := db.Order("dist_orders.updated_at DESC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) SubmitOrder(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Order, error) {
	order, err := s.mutableOrder(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if order.Status != "DRAFT" {
		return nil, apperrors.New("INVALID_ORDER_STATE", "Only draft orders can be submitted", 422)
	}
	now := time.Now().UTC()
	err = s.withTenant(ctx, order.TenantID, func(tx *gorm.DB) error {
		return tx.Model(&Order{}).Where("id = ? AND status = 'DRAFT'", id).
			Updates(map[string]interface{}{"status": "SUBMITTED", "submitted_at": now}).Error
	})
	if err != nil {
		return nil, err
	}
	order.Status, order.SubmittedAt = "SUBMITTED", &now
	s.record(ctx, principal, "ORDER_SUBMIT", "ORDER", id, order)
	return order, nil
}

func (s *Service) ApproveOrder(ctx context.Context, principal auth.Principal, id uuid.UUID, remarks string) (*Order, error) {
	if !principal.HasPermission("order.approve") {
		return nil, apperrors.ErrForbidden
	}
	order, err := s.mutableOrder(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if order.Status != "SUBMITTED" {
		return nil, apperrors.New("INVALID_ORDER_STATE", "Only submitted orders can be approved", 422)
	}
	var items []OrderItem
	if err := s.db.WithContext(ctx).Where("order_id = ?", id).Find(&items).Error; err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, apperrors.New("EMPTY_ORDER", "Order has no items", 422)
	}
	now := time.Now().UTC()
	movementKey := uuid.NewSHA1(uuid.NameSpaceOID, []byte(order.ID.String()+":approve"))
	err = s.withTenant(ctx, order.TenantID, func(tx *gorm.DB) error {
		if err := setTenantContext(tx, order.DistributorTenantID); err != nil {
			return err
		}
		for _, item := range items {
			if err := s.reserveStock(tx, order.DistributorTenantID, item.ProductID, item.Quantity, movementKey, order.ID, principal.UserID); err != nil {
				return err
			}
		}
		if err := setTenantContext(tx, order.TenantID); err != nil {
			return err
		}
		return tx.Model(&Order{}).Where("id = ? AND status = 'SUBMITTED'", id).
			Updates(map[string]interface{}{"status": "APPROVED", "approved_by": principal.UserID, "approved_at": now, "remarks": remarks}).Error
	})
	if err != nil {
		return nil, err
	}
	order.Status, order.ApprovedBy, order.ApprovedAt = "APPROVED", &principal.UserID, &now
	s.record(ctx, principal, "ORDER_APPROVE", "ORDER", id, order)
	return order, nil
}

func (s *Service) CancelOrder(ctx context.Context, principal auth.Principal, id uuid.UUID, remarks string) (*Order, error) {
	order, err := s.mutableOrder(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if order.Status != "DRAFT" && order.Status != "SUBMITTED" {
		return nil, apperrors.New("INVALID_ORDER_STATE", "Only draft or submitted orders can be cancelled", 422)
	}
	now := time.Now().UTC()
	err = s.withTenant(ctx, order.TenantID, func(tx *gorm.DB) error {
		return tx.Model(&Order{}).Where("id = ? AND status IN ?", id, []string{"DRAFT", "SUBMITTED"}).
			Updates(map[string]interface{}{"status": "CANCELLED", "cancelled_at": now, "remarks": remarks}).Error
	})
	if err != nil {
		return nil, err
	}
	order.Status, order.CancelledAt = "CANCELLED", &now
	s.record(ctx, principal, "ORDER_CANCEL", "ORDER", id, order)
	return order, nil
}

func (s *Service) CreateSecondarySale(ctx context.Context, principal auth.Principal, request SecondarySaleRequest, idempotencyKey uuid.UUID) (*SecondarySale, error) {
	if !principal.HasPermission("secondary_sales.import") {
		return nil, apperrors.ErrForbidden
	}
	companyTenantID := request.CompanyTenantID
	if companyTenantID == nil && request.OrderID != nil {
		var order CompanyOrderRef
		if err := s.db.WithContext(ctx).Table("dist_orders").Select("tenant_id AS company_tenant_id").
			Where("id = ? AND distributor_tenant_id = ?", *request.OrderID, principal.TenantID).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.New("INVALID_ORDER", "Order is not linked to this distributor", 422)
			}
			return nil, err
		}
		companyTenantID = &order.CompanyTenantID
	}
	company, distributorTenantID, err := s.resolveTenants(ctx, principal, companyTenantID, principal.TenantID)
	if err != nil {
		return nil, err
	}
	if distributorTenantID != principal.TenantID {
		return nil, apperrors.ErrForbidden
	}
	if _, err := s.activeLink(ctx, company, distributorTenantID); err != nil {
		return nil, err
	}
	if request.OrderID != nil {
		var order Order
		if err := s.db.WithContext(ctx).Where("id = ? AND tenant_id = ? AND distributor_tenant_id = ?", *request.OrderID, company, distributorTenantID).First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, apperrors.New("INVALID_ORDER", "Order is not linked to this distributor", 422)
			}
			return nil, err
		}
	}
	saleDate, err := parseDateOrDefault(request.SaleDate)
	if err != nil {
		return nil, err
	}
	id := uuid.New()
	if request.ID != nil {
		id = *request.ID
	}
	existingID, replay, err := s.claimIdempotency(ctx, distributorTenantID, idempotencyKey, "secondary_sale.create")
	if err != nil {
		return nil, err
	}
	if replay {
		if existingID == nil {
			return nil, apperrors.ErrConflict
		}
		return s.GetSecondarySale(ctx, principal, *existingID)
	}
	sale := &SecondarySale{
		ID: id, TenantID: distributorTenantID, OrderID: request.OrderID,
		CompanyTenantID: &company, ProductID: request.ProductID,
		Quantity: request.Quantity, UnitPrice: request.UnitPrice,
		TotalAmount: float64(request.Quantity) * request.UnitPrice, SaleDate: saleDate,
		InvoiceNo: request.InvoiceNo, CustomerCode: request.CustomerCode,
		IdempotencyKey: idempotencyKey, CreatedBy: &principal.UserID,
	}
	err = s.withTenant(ctx, distributorTenantID, func(tx *gorm.DB) error {
		if err := tx.Create(sale).Error; err != nil {
			return err
		}
		return s.completeIdempotency(tx, distributorTenantID, idempotencyKey, id)
	})
	if err != nil {
		s.failIdempotency(ctx, distributorTenantID, idempotencyKey, err.Error())
		if isUniqueError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "SECONDARY_SALE_CREATE", "SECONDARY_SALE", sale.ID, sale)
	return sale, nil
}

func (s *Service) GetSecondarySale(ctx context.Context, principal auth.Principal, id uuid.UUID) (*SecondarySale, error) {
	var sale SecondarySale
	db := s.db.WithContext(ctx).Model(&SecondarySale{})
	if !principal.IsPrivileged {
		db = db.Where(`tenant_id = ? OR (company_tenant_id = ? AND EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.tenant_id = dist_secondary_sales.company_tenant_id
			  AND l.distributor_tenant_id = dist_secondary_sales.tenant_id
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))`, principal.TenantID, principal.TenantID)
	}
	if err := db.Where("id = ?", id).First(&sale).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &sale, nil
}

func (s *Service) ListSecondarySales(ctx context.Context, principal auth.Principal, page query.Page, from, to string) ([]SecondarySale, int64, error) {
	db := s.db.WithContext(ctx).Model(&SecondarySale{})
	if !principal.IsPrivileged {
		db = db.Where(`tenant_id = ? OR (company_tenant_id = ? AND EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.tenant_id = dist_secondary_sales.company_tenant_id
			  AND l.distributor_tenant_id = dist_secondary_sales.tenant_id
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))`, principal.TenantID, principal.TenantID)
	}
	if from != "" {
		parsed, err := parseDate(from)
		if err != nil {
			return nil, 0, err
		}
		db = db.Where("sale_date >= ?", parsed)
	}
	if to != "" {
		parsed, err := parseDate(to)
		if err != nil {
			return nil, 0, err
		}
		db = db.Where("sale_date <= ?", parsed)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []SecondarySale
	err := db.Order("sale_date DESC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) ListInventory(ctx context.Context, principal auth.Principal, page query.Page, productID uuid.UUID, warehouseCode string) ([]Inventory, int64, error) {
	db := s.db.WithContext(ctx).Model(&Inventory{})
	if !principal.IsPrivileged {
		db = db.Where(`tenant_id = ? OR EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.distributor_tenant_id = dist_inventory.tenant_id
			  AND l.tenant_id = ?
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE))`, principal.TenantID, principal.TenantID)
	}
	if productID != uuid.Nil {
		db = db.Where("product_id = ?", productID)
	}
	if warehouseCode != "" {
		db = db.Where("warehouse_code = ?", warehouseCode)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Inventory
	err := db.Order("updated_at DESC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) AdjustInventory(ctx context.Context, principal auth.Principal, request InventoryAdjustmentRequest, idempotencyKey uuid.UUID) (*InventoryMovement, error) {
	if !principal.HasPermission("inventory.manage") {
		return nil, apperrors.ErrForbidden
	}
	occurredAt, err := parseTimeOrDefault(request.OccurredAt)
	if err != nil {
		return nil, err
	}
	warehouse := strings.TrimSpace(request.WarehouseCode)
	if warehouse == "" {
		warehouse = "MAIN"
	}
	id := uuid.New()
	if request.ID != nil {
		id = *request.ID
	}
	movement := &InventoryMovement{
		ID: id, TenantID: principal.TenantID, ProductID: request.ProductID,
		WarehouseCode: warehouse, MovementType: request.MovementType,
		QuantityDelta: request.QuantityDelta, ReferenceType: request.ReferenceType,
		ReferenceID: request.ReferenceID, IdempotencyKey: idempotencyKey,
		CreatedBy: &principal.UserID, OccurredAt: occurredAt,
	}
	err = s.withTenant(ctx, principal.TenantID, func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("idempotency_keys").
			Where("tenant_id = ? AND key = ? AND operation = ?", principal.TenantID, idempotencyKey, "inventory.movement").
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return apperrors.ErrConflict
		}
		if err := tx.Exec(`INSERT INTO idempotency_keys (tenant_id, key, operation, status)
			VALUES (?, ?, 'inventory.movement', 'PENDING')`, principal.TenantID, idempotencyKey).Error; err != nil {
			return err
		}
		if err := s.applyMovement(tx, principal.TenantID, request.ProductID, warehouse, request.QuantityDelta); err != nil {
			return err
		}
		if err := tx.Create(movement).Error; err != nil {
			return err
		}
		return tx.Exec(`UPDATE idempotency_keys SET status = 'COMPLETED', entity_id = ? WHERE tenant_id = ? AND key = ? AND operation = ?`,
			id, principal.TenantID, idempotencyKey, "inventory.movement").Error
	})
	if err != nil {
		if isUniqueError(err) {
			return nil, apperrors.ErrConflict
		}
		return nil, err
	}
	s.record(ctx, principal, "INVENTORY_ADJUST", "INVENTORY_MOVEMENT", movement.ID, movement)
	return movement, nil
}

func (s *Service) ListInventoryMovements(ctx context.Context, principal auth.Principal, page query.Page, productID uuid.UUID) ([]InventoryMovement, int64, error) {
	db := s.db.WithContext(ctx).Model(&InventoryMovement{})
	if !principal.IsPrivileged {
		db = db.Where(`tenant_id = ? OR EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.distributor_tenant_id = dist_inventory_movements.tenant_id
			  AND l.tenant_id = ?
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE))`, principal.TenantID, principal.TenantID)
	}
	if productID != uuid.Nil {
		db = db.Where("product_id = ?", productID)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []InventoryMovement
	err := db.Order("occurred_at DESC").Limit(page.Size).Offset(page.Offset()).Find(&items).Error
	return items, total, err
}

func (s *Service) activeLink(ctx context.Context, companyTenantID, distributorTenantID uuid.UUID) (*DistributionLink, error) {
	var link DistributionLink
	err := s.db.WithContext(ctx).Where(`tenant_id = ? AND distributor_tenant_id = ? AND status = 'ACTIVE'
		AND effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)`,
		companyTenantID, distributorTenantID).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.New("LINK_NOT_ACTIVE", "No active link exists for these tenants", 422)
	}
	if err != nil {
		return nil, err
	}
	return &link, nil
}

func (s *Service) resolveLink(ctx context.Context, principal auth.Principal, companyTenantID *uuid.UUID, distributorTenantID uuid.UUID) (uuid.UUID, uuid.UUID, *DistributionLink, error) {
	company, distributor, err := s.resolveTenants(ctx, principal, companyTenantID, distributorTenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}
	if distributor == uuid.Nil || distributor == company {
		return uuid.Nil, uuid.Nil, nil, apperrors.New("INVALID_DISTRIBUTOR", "A distinct distributor tenant is required", 422)
	}
	link, err := s.activeLink(ctx, company, distributor)
	if err != nil {
		return uuid.Nil, uuid.Nil, nil, err
	}
	return company, distributor, link, nil
}

func (s *Service) resolveTenants(ctx context.Context, principal auth.Principal, companyTenantID *uuid.UUID, distributorTenantID uuid.UUID) (uuid.UUID, uuid.UUID, error) {
	company := principal.TenantID
	if companyTenantID != nil {
		company = *companyTenantID
	}
	distributor := distributorTenantID
	if distributor == uuid.Nil {
		distributor = principal.TenantID
	}
	if !principal.IsPrivileged && principal.TenantID != company && principal.TenantID != distributor {
		return uuid.Nil, uuid.Nil, apperrors.ErrForbidden
	}
	_ = ctx
	return company, distributor, nil
}

func (s *Service) orderQuery(ctx context.Context, principal auth.Principal) *gorm.DB {
	db := s.db.WithContext(ctx).Model(&Order{})
	if principal.IsPrivileged {
		return db
	}
	return db.Where(`dist_orders.tenant_id = ? OR (dist_orders.distributor_tenant_id = ? AND EXISTS (
		SELECT 1 FROM dist_company_distributor_links l
		WHERE l.tenant_id = dist_orders.tenant_id
		  AND l.distributor_tenant_id = dist_orders.distributor_tenant_id
		  AND l.status = 'ACTIVE'
		  AND l.effective_from <= CURRENT_DATE
		  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))`, principal.TenantID, principal.TenantID)
}

func (s *Service) orderItemQuery(ctx context.Context, principal auth.Principal) *gorm.DB {
	db := s.db.WithContext(ctx).Model(&OrderItem{})
	if principal.IsPrivileged {
		return db
	}
	return db.Where(`EXISTS (
		SELECT 1 FROM dist_orders o
		WHERE o.id = dist_order_items.order_id
		  AND (o.tenant_id = ? OR (o.distributor_tenant_id = ? AND EXISTS (
			SELECT 1 FROM dist_company_distributor_links l
			WHERE l.tenant_id = o.tenant_id
			  AND l.distributor_tenant_id = o.distributor_tenant_id
			  AND l.status = 'ACTIVE'
			  AND l.effective_from <= CURRENT_DATE
			  AND (l.effective_to IS NULL OR l.effective_to >= CURRENT_DATE)))))`, principal.TenantID, principal.TenantID)
}

func (s *Service) mutableOrder(ctx context.Context, principal auth.Principal, id uuid.UUID) (*Order, error) {
	order, err := s.GetOrder(ctx, principal, id)
	if err != nil {
		return nil, err
	}
	if !principal.IsPrivileged && order.TenantID != principal.TenantID {
		return nil, apperrors.ErrForbidden
	}
	if _, err := s.activeLink(ctx, order.TenantID, order.DistributorTenantID); err != nil {
		return nil, err
	}
	return &order.Order, nil
}

func (s *Service) buildOrderItems(tx *gorm.DB, companyTenantID, orderID uuid.UUID, requests []OrderItemRequest) ([]OrderItem, float64, error) {
	items := make([]OrderItem, 0, len(requests))
	total := 0.0
	seen := map[uuid.UUID]bool{}
	for _, request := range requests {
		if seen[request.ProductID] {
			return nil, 0, apperrors.New("DUPLICATE_PRODUCT", "Each product may appear once per order", 422)
		}
		seen[request.ProductID] = true
		var product struct {
			MRP float64 `gorm:"column:mrp"`
		}
		if err := tx.Table("cat_products").Select("mrp").
			Where("tenant_id = ? AND id = ? AND is_active = true AND deleted_at IS NULL", companyTenantID, request.ProductID).
			First(&product).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, 0, apperrors.New("INVALID_PRODUCT", "Product is not available for this company", 422)
			}
			return nil, 0, err
		}
		unitPrice := request.UnitPrice
		if unitPrice <= 0 {
			unitPrice = product.MRP
		}
		id := uuid.New()
		if request.ID != nil {
			id = *request.ID
		}
		lineTotal := float64(request.Quantity) * unitPrice
		items = append(items, OrderItem{
			ID: id, TenantID: companyTenantID, OrderID: orderID, ProductID: request.ProductID,
			Quantity: request.Quantity, UnitPrice: unitPrice, LineTotal: lineTotal,
		})
		total += lineTotal
	}
	return items, total, nil
}

func (s *Service) reserveStock(tx *gorm.DB, distributorTenantID, productID uuid.UUID, quantity int, movementKey, orderID, userID uuid.UUID) error {
	var inventory Inventory
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND product_id = ? AND warehouse_code = ?", distributorTenantID, productID, "MAIN").
		First(&inventory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return apperrors.New("INSUFFICIENT_STOCK", "No inventory record exists for an ordered product", 422)
	}
	if err != nil {
		return err
	}
	if inventory.QuantityOnHand-inventory.QuantityReserved < quantity {
		return apperrors.New("INSUFFICIENT_STOCK", "Insufficient stock to approve order", 422)
	}
	result := tx.Model(&Inventory{}).
		Where("id = ? AND version = ?", inventory.ID, inventory.Version).
		Updates(map[string]interface{}{
			"quantity_reserved": gorm.Expr("quantity_reserved + ?", quantity),
			"version":           gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.New("STOCK_CONFLICT", "Inventory changed during approval; retry", 409)
	}
	lineKey := uuid.NewSHA1(uuid.NameSpaceOID, []byte(movementKey.String()+":"+productID.String()))
	movement := InventoryMovement{
		ID:       lineKey,
		TenantID: distributorTenantID, ProductID: productID, WarehouseCode: "MAIN",
		MovementType: "ORDER_RESERVE", QuantityDelta: -quantity,
		ReferenceType: "ORDER", ReferenceID: &orderID, IdempotencyKey: lineKey,
		CreatedBy: &userID, OccurredAt: time.Now().UTC(),
	}
	if err := tx.Create(&movement).Error; err != nil {
		return err
	}
	return nil
}

func (s *Service) applyMovement(tx *gorm.DB, tenantID, productID uuid.UUID, warehouse string, delta int) error {
	var inventory Inventory
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND product_id = ? AND warehouse_code = ?", tenantID, productID, warehouse).
		First(&inventory).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		inventory = Inventory{
			ID: uuid.New(), TenantID: tenantID, ProductID: productID,
			WarehouseCode: warehouse, QuantityOnHand: 0, QuantityReserved: 0, Version: 1,
		}
		if err := tx.Create(&inventory).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	onHand := inventory.QuantityOnHand + delta
	if onHand < inventory.QuantityReserved {
		return apperrors.New("INVALID_INVENTORY", "Adjustment cannot reduce stock below reserved quantity", 422)
	}
	result := tx.Model(&Inventory{}).Where("id = ? AND version = ?", inventory.ID, inventory.Version).
		Updates(map[string]interface{}{
			"quantity_on_hand": gorm.Expr("quantity_on_hand + ?", delta),
			"version":          gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.New("STOCK_CONFLICT", "Inventory changed during adjustment; retry", 409)
	}
	return nil
}

func (s *Service) claimIdempotency(ctx context.Context, tenantID, key uuid.UUID, operation string) (*uuid.UUID, bool, error) {
	if key == uuid.Nil {
		return nil, false, apperrors.New("INVALID_IDEMPOTENCY_KEY", "Idempotency key is required", 422)
	}
	db := s.db.WithContext(ctx)
	result := db.Exec(`INSERT INTO idempotency_keys (tenant_id, key, operation, status)
		VALUES (?, ?, ?, 'PENDING') ON CONFLICT (tenant_id, key) DO NOTHING`, tenantID, key, operation)
	if result.Error != nil {
		return nil, false, result.Error
	}
	if result.RowsAffected > 0 {
		return nil, false, nil
	}
	var existing struct {
		Operation string     `gorm:"column:operation"`
		Status    string     `gorm:"column:status"`
		EntityID  *uuid.UUID `gorm:"column:entity_id"`
	}
	if err := db.Raw(`SELECT operation, status, entity_id FROM idempotency_keys WHERE tenant_id = ? AND key = ?`,
		tenantID, key).Scan(&existing).Error; err != nil {
		return nil, false, err
	}
	if existing.Operation != operation {
		return nil, false, apperrors.ErrConflict
	}
	if existing.Status == "COMPLETED" {
		return existing.EntityID, true, nil
	}
	return nil, false, apperrors.New("IDEMPOTENCY_BUSY", "Operation is already in progress", 409)
}

func (s *Service) completeIdempotency(tx *gorm.DB, tenantID, key, entityID uuid.UUID) error {
	return tx.Exec(`UPDATE idempotency_keys SET status = 'COMPLETED', entity_id = ? WHERE tenant_id = ? AND key = ?`,
		entityID, tenantID, key).Error
}

func (s *Service) failIdempotency(ctx context.Context, tenantID, key uuid.UUID, reason string) {
	_ = s.db.WithContext(ctx).Exec(`UPDATE idempotency_keys SET status = 'FAILED', result = ?::jsonb WHERE tenant_id = ? AND key = ?`,
		`{"error":`+quote(reason)+`}`, tenantID, key).Error
}

func (s *Service) withTenant(ctx context.Context, tenantID uuid.UUID, fn func(*gorm.DB) error) error {
	tx := s.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if tx.Error != nil {
			tx.Rollback()
		}
	}()
	if err := setTenantContext(tx, tenantID); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'false', true)`).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func (s *Service) record(ctx context.Context, principal auth.Principal, action, entityType string, entityID uuid.UUID, value interface{}) {
	_ = s.audit.Record(ctx, audit.Entry{
		TenantID: &principal.TenantID, UserID: &principal.UserID,
		Action: action, Module: "distribution", EntityType: entityType,
		EntityID: entityID.String(), New: value,
	})
}

func setTenantContext(tx *gorm.DB, tenantID uuid.UUID) error {
	return tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantID.String()).Error
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, apperrors.New("INVALID_DATE", "Date must be YYYY-MM-DD", 422)
	}
	return parsed, nil
}

func parseDateOrDefault(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return today(), nil
	}
	return parseDate(value)
}

func parseTimeOrDefault(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now().UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, apperrors.New("INVALID_DATE", "Timestamp must be RFC3339", 422)
	}
	return parsed, nil
}

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func validStatus(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique")
}

func quote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
