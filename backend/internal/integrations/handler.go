package integrations

import (
	"net/http"

	"github.com/gin-gonic/gin"

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
	connections := router.Group("/integrations/connections")
	connections.GET("", middleware.RequirePermission("integration.view"), h.listConnections)
	connections.POST("", middleware.RequirePermission("integration.manage"), h.createConnection)
	connections.PATCH("/:id/status", middleware.RequirePermission("integration.manage"), h.setConnectionStatus)
	connections.DELETE("/:id", middleware.RequirePermission("integration.manage"), h.deleteConnection)

	events := router.Group("/integrations/events")
	events.POST("", middleware.RequirePermission("integration.manage"), h.publishEvent)

	deliveries := router.Group("/integrations/deliveries")
	deliveries.GET("", middleware.RequirePermission("integration.view"), h.listDeliveries)
	deliveries.POST("/:id/retry", middleware.RequirePermission("webhook.retry"), h.retryDelivery)
}

func (h *Handler) createConnection(c *gin.Context) {
	var request ConnectionCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	record, err := h.service.CreateConnection(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Integration connection created", record)
}

func (h *Handler) listConnections(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.ListConnections(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		page, c.Query("status"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Integration connections retrieved", items, query.Meta(page, total))
}

func (h *Handler) setConnectionStatus(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request ConnectionStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	if err := h.service.SetConnectionStatus(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request.Status); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) deleteConnection(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteConnection(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) publishEvent(c *gin.Context) {
	var request EventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	deliveries, err := h.service.Publish(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Webhook event queued", deliveries)
}

func (h *Handler) listDeliveries(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.ListDeliveries(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		page, c.Query("status"), c.Query("event_type"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Webhook deliveries retrieved", items, query.Meta(page, total))
}

func (h *Handler) retryDelivery(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.RetryDelivery(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"retried": true})
}
