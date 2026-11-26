package distribution

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/middleware"
	"github.com/pharmaos/pharmaos/backend/internal/platform/httputil"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *gin.RouterGroup) {
	links := router.Group("/distribution-links")
	links.GET("", middleware.RequirePermission("distributor.view"), h.listLinks)
	links.POST("", middleware.RequirePermission("link.manage"), h.createLink)
	links.PATCH("/:id/status", middleware.RequirePermission("link.manage"), h.updateLinkStatus)

	distributors := router.Group("/distributors")
	distributors.GET("", middleware.RequirePermission("distributor.view"), h.listDistributors)
	distributors.POST("", middleware.RequirePermission("distributor.manage"), h.createDistributor)
	distributors.GET("/:id", middleware.RequirePermission("distributor.view"), h.getDistributor)

	orders := router.Group("/orders")
	orders.GET("", middleware.RequirePermission("order.view"), h.listOrders)
	orders.POST("", middleware.RequirePermission("order.manage"), h.createOrder)
	orders.GET("/:id", middleware.RequirePermission("order.view"), h.getOrder)
	orders.POST("/:id/submit", middleware.RequirePermission("order.manage"), h.submitOrder)
	orders.POST("/:id/approve", middleware.RequirePermission("order.approve"), h.approveOrder)
	orders.POST("/:id/cancel", middleware.RequirePermission("order.manage"), h.cancelOrder)

	inventory := router.Group("/inventory")
	inventory.GET("", middleware.RequirePermission("inventory.view"), h.listInventory)
	inventory.GET("/movements", middleware.RequirePermission("inventory.view"), h.listInventoryMovements)
	inventory.POST("/movements", middleware.RequirePermission("inventory.manage"), h.adjustInventory)

	sales := router.Group("/secondary-sales")
	sales.GET("", middleware.RequirePermission("secondary_sales.view"), h.listSecondarySales)
	sales.POST("", middleware.RequirePermission("secondary_sales.import"), h.createSecondarySale)
}

func (h *Handler) listLinks(c *gin.Context) {
	items, total, err := h.service.ListLinks(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), c.Query("status"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Distribution links retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) createLink(c *gin.Context) {
	var request LinkCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	link, err := h.service.CreateLink(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Distribution link created", link)
}

func (h *Handler) updateLinkStatus(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request LinkStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	if err := h.service.UpdateLinkStatus(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) listDistributors(c *gin.Context) {
	items, total, err := h.service.ListDistributors(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), c.Query("status"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Distributors retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) createDistributor(c *gin.Context) {
	var request DistributorCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	record, err := h.service.CreateDistributor(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Distributor created", record)
}

func (h *Handler) getDistributor(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	record, err := h.service.GetDistributor(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, record)
}

func (h *Handler) listOrders(c *gin.Context) {
	items, total, err := h.service.ListOrders(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), c.Query("status"), optionalUUID(c.Query("distributor_tenant_id")))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Orders retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) createOrder(c *gin.Context) {
	key, ok := idempotencyKey(c)
	if !ok {
		return
	}
	var request OrderCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	order, err := h.service.CreateOrder(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request, key)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Order created", order)
}

func (h *Handler) getOrder(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	order, err := h.service.GetOrder(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Handler) submitOrder(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	order, err := h.service.SubmitOrder(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Handler) approveOrder(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request OrderDecisionRequest
	if err := c.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		httputil.BadJSON(c, err)
		return
	}
	order, err := h.service.ApproveOrder(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request.Remarks)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Handler) cancelOrder(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request OrderDecisionRequest
	if err := c.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		httputil.BadJSON(c, err)
		return
	}
	order, err := h.service.CancelOrder(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request.Remarks)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, order)
}

func (h *Handler) listSecondarySales(c *gin.Context) {
	items, total, err := h.service.ListSecondarySales(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), c.Query("from"), c.Query("to"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Secondary sales retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) createSecondarySale(c *gin.Context) {
	key, ok := idempotencyKey(c)
	if !ok {
		return
	}
	var request SecondarySaleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	sale, err := h.service.CreateSecondarySale(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request, key)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Secondary sale recorded", sale)
}

func (h *Handler) listInventory(c *gin.Context) {
	items, total, err := h.service.ListInventory(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), optionalUUID(c.Query("product_id")), c.Query("warehouse_code"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Inventory retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) listInventoryMovements(c *gin.Context) {
	items, total, err := h.service.ListInventoryMovements(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		query.Parse(c), optionalUUID(c.Query("product_id")))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Inventory movements retrieved", items, query.Meta(query.Parse(c), total))
}

func (h *Handler) adjustInventory(c *gin.Context) {
	key, ok := idempotencyKey(c)
	if !ok {
		return
	}
	var request InventoryAdjustmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	movement, err := h.service.AdjustInventory(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request, key)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Inventory adjusted", movement)
}

func idempotencyKey(c *gin.Context) (uuid.UUID, bool) {
	key, err := uuid.Parse(c.GetHeader("Idempotency-Key"))
	if err == nil {
		return key, true
	}
	response.Fail(c, http.StatusBadRequest, "Invalid or missing Idempotency-Key header", map[string]string{"Idempotency-Key": "must be a UUID"})
	return uuid.Nil, false
}

func optionalUUID(value string) uuid.UUID {
	if value == "" {
		return uuid.Nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}
