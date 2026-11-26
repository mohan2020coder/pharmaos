package masterdata

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pharmaos/pharmaos/backend/internal/platform/httputil"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
)

func partyFilters(c *gin.Context) ([]func(*gorm.DB) *gorm.DB, bool) {
	var filters []func(*gorm.DB) *gorm.DB
	if territoryID := optionalUUIDFromQuery(c.Query("territory_id")); territoryID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("territory_id = ?", territoryID) })
	}
	if divisionID := optionalUUIDFromQuery(c.Query("division_id")); divisionID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("division_id = ?", divisionID) })
	}
	active, ok := activeFilter(c)
	if !ok {
		return nil, false
	}
	if active != nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("is_active = ?", *active) })
	}
	return filters, true
}

func (h *Handler) listChemists(c *gin.Context) {
	page := query.Parse(c)
	filters, ok := partyFilters(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "Invalid is_active filter", nil)
		return
	}
	items, total, err := h.chemists.List(c.Request.Context(), tenant(c), page, []string{"name", "owner_name", "license_no"}, filters...)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Chemists retrieved", items, query.Meta(page, total))
}

func (h *Handler) createChemist(c *gin.Context) {
	var request ChemistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	chemist := &Chemist{
		ID: uuid.New(), TenantID: tenant(c), ExternalID: externalID(request.ExternalID),
		Name: strings.TrimSpace(request.Name), OwnerName: request.OwnerName, LicenseNo: request.LicenseNo,
		Email: request.Email, Phone: request.Phone, Address: request.Address,
		TerritoryID: request.TerritoryID, DivisionID: request.DivisionID,
		IsActive: boolOr(request.IsActive, true), Settings: settingsOrEmpty(request.Settings),
	}
	if err := conflict(h.chemists.Create(c.Request.Context(), chemist)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Chemist created successfully", chemist)
}

func (h *Handler) getChemist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	chemist, err := h.chemists.Get(c.Request.Context(), tenant(c), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, chemist)
}

func (h *Handler) updateChemist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request ChemistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	values := map[string]interface{}{
		"name": strings.TrimSpace(request.Name), "owner_name": request.OwnerName,
		"license_no": request.LicenseNo, "email": request.Email, "phone": request.Phone,
		"address": request.Address, "territory_id": request.TerritoryID,
		"division_id": request.DivisionID, "settings": settingsOrEmpty(request.Settings),
	}
	if request.IsActive != nil {
		values["is_active"] = *request.IsActive
	}
	if err := conflict(h.chemists.Update(c.Request.Context(), tenant(c), id, values)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) deleteChemist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.chemists.Delete(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) listStockists(c *gin.Context) {
	page := query.Parse(c)
	filters, ok := partyFilters(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "Invalid is_active filter", nil)
		return
	}
	items, total, err := h.stockists.List(c.Request.Context(), tenant(c), page, []string{"name", "license_no", "gst_number"}, filters...)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Stockists retrieved", items, query.Meta(page, total))
}

func (h *Handler) createStockist(c *gin.Context) {
	var request StockistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	stockist := &Stockist{
		ID: uuid.New(), TenantID: tenant(c), ExternalID: externalID(request.ExternalID),
		Name: strings.TrimSpace(request.Name), LicenseNo: request.LicenseNo, GSTNumber: request.GSTNumber,
		Email: request.Email, Phone: request.Phone, Address: request.Address,
		TerritoryID: request.TerritoryID, DivisionID: request.DivisionID,
		CreditLimit: request.CreditLimit, PaymentTermsDays: request.PaymentTermsDays,
		IsActive: boolOr(request.IsActive, true), Settings: settingsOrEmpty(request.Settings),
	}
	if err := conflict(h.stockists.Create(c.Request.Context(), stockist)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Stockist created successfully", stockist)
}

func (h *Handler) getStockist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	stockist, err := h.stockists.Get(c.Request.Context(), tenant(c), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, stockist)
}

func (h *Handler) updateStockist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request StockistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	values := map[string]interface{}{
		"name": strings.TrimSpace(request.Name), "license_no": request.LicenseNo,
		"gst_number": request.GSTNumber, "email": request.Email, "phone": request.Phone,
		"address": request.Address, "territory_id": request.TerritoryID,
		"division_id": request.DivisionID, "credit_limit": request.CreditLimit,
		"payment_terms_days": request.PaymentTermsDays,
		"settings":           settingsOrEmpty(request.Settings),
	}
	if request.IsActive != nil {
		values["is_active"] = *request.IsActive
	}
	if err := conflict(h.stockists.Update(c.Request.Context(), tenant(c), id, values)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) deleteStockist(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.stockists.Delete(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
