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

func (h *Handler) listCategories(c *gin.Context) {
	page := query.Parse(c)
	active, ok := activeFilter(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "Invalid is_active filter", nil)
		return
	}
	var filters []func(*gorm.DB) *gorm.DB
	if active != nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("is_active = ?", *active) })
	}
	items, total, err := h.categories.List(c.Request.Context(), tenant(c), page, []string{"code", "name"}, filters...)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Product categories retrieved", items, query.Meta(page, total))
}

func (h *Handler) createCategory(c *gin.Context) {
	var request ProductCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	category := &ProductCategory{
		ID: uuid.New(), TenantID: tenant(c),
		Code: strings.ToUpper(strings.TrimSpace(request.Code)), Name: strings.TrimSpace(request.Name),
		ParentCategoryID: request.ParentCategoryID, IsActive: boolOr(request.IsActive, true),
	}
	if err := conflict(h.categories.Create(c.Request.Context(), category)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Product category created successfully", category)
}

func (h *Handler) listProducts(c *gin.Context) {
	page := query.Parse(c)
	var filters []func(*gorm.DB) *gorm.DB
	if categoryID := optionalUUIDFromQuery(c.Query("category_id")); categoryID != uuid.Nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("category_id = ?", categoryID) })
	}
	active, ok := activeFilter(c)
	if !ok {
		response.Fail(c, http.StatusBadRequest, "Invalid is_active filter", nil)
		return
	}
	if active != nil {
		filters = append(filters, func(db *gorm.DB) *gorm.DB { return db.Where("is_active = ?", *active) })
	}
	items, total, err := h.products.List(c.Request.Context(), tenant(c), page,
		[]string{"code", "name", "brand", "manufacturer", "composition"}, filters...)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Products retrieved", items, query.Meta(page, total))
}

func (h *Handler) createProduct(c *gin.Context) {
	var request ProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	product := &Product{
		ID: uuid.New(), TenantID: tenant(c), CategoryID: request.CategoryID,
		Code: strings.ToUpper(strings.TrimSpace(request.Code)), Name: strings.TrimSpace(request.Name),
		Brand: request.Brand, Manufacturer: request.Manufacturer, Composition: request.Composition,
		PackSize: request.PackSize, Unit: request.Unit, MRP: request.MRP,
		IsActive: boolOr(request.IsActive, true), Settings: settingsOrEmpty(request.Settings),
	}
	if err := conflict(h.products.Create(c.Request.Context(), product)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Product created successfully", product)
}

func (h *Handler) getProduct(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	product, err := h.products.Get(c.Request.Context(), tenant(c), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, product)
}

func (h *Handler) updateProduct(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request ProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	values := map[string]interface{}{
		"category_id": request.CategoryID, "code": strings.ToUpper(strings.TrimSpace(request.Code)),
		"name": strings.TrimSpace(request.Name), "brand": request.Brand,
		"manufacturer": request.Manufacturer, "composition": request.Composition,
		"pack_size": request.PackSize, "unit": request.Unit, "mrp": request.MRP,
		"settings": settingsOrEmpty(request.Settings),
	}
	if request.IsActive != nil {
		values["is_active"] = *request.IsActive
	}
	if err := conflict(h.products.Update(c.Request.Context(), tenant(c), id, values)); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) deleteProduct(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.products.Delete(c.Request.Context(), tenant(c), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
