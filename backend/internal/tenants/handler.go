package tenants

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/middleware"
	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/query"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(router *gin.RouterGroup) {
	group := router.Group("/tenants")
	group.GET("", middleware.RequirePermission("tenant.view"), h.list)
	group.GET("/:id", middleware.RequirePermission("tenant.view"), h.get)
	group.POST("", middleware.RequirePermission("tenant.manage"), h.create)
	group.PATCH("/:id", middleware.RequirePermission("tenant.manage"), h.update)
	group.PUT("/:id/settings/:key", middleware.RequirePermission("tenant.manage"), h.setSetting)
}

func (h *Handler) create(c *gin.Context) {
	var request CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	tenant, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Created(c, "Tenant created successfully", tenant)
}

func (h *Handler) list(c *gin.Context) {
	principal := auth.MustPrincipal(c.Request.Context())
	page := query.Parse(c)
	items, total, err := h.service.List(c.Request.Context(), principal, page)
	if err != nil {
		handleError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Tenants retrieved", items, query.Meta(page, total))
}

func (h *Handler) get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid tenant id", gin.H{"id": "must be a UUID"})
		return
	}
	tenant, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, tenant)
}

func (h *Handler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid tenant id", gin.H{"id": "must be a UUID"})
		return
	}
	var request UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if validationErrors := validate.Struct(request); validationErrors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", validationErrors)
		return
	}
	tenant, err := h.service.Update(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, tenant)
}

func (h *Handler) setSetting(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid tenant id", gin.H{"id": "must be a UUID"})
		return
	}
	var value map[string]any
	if err := c.ShouldBindJSON(&value); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if err := h.service.SetSetting(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, c.Param("key"), value); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, gin.H{"tenant_id": id, "key": c.Param("key"), "value": value})
}

func handleError(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response.Fail(c, appErr.Status, appErr.Message, appErr.Details)
		return
	}
	response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
}
