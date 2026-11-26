package roles

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
	group := router.Group("/roles")
	group.GET("", middleware.RequirePermission("role.view"), h.list)
	group.GET("/:id", middleware.RequirePermission("role.view"), h.get)
	group.POST("", middleware.RequirePermission("role.manage"), h.create)
	group.PATCH("/:id", middleware.RequirePermission("role.manage"), h.update)
	group.DELETE("/:id", middleware.RequirePermission("role.manage"), h.delete)
	group.PUT("/:id/permissions", middleware.RequirePermission("role.manage"), h.setPermissions)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	roles, total, err := h.service.List(c.Request.Context(), page)
	if err != nil {
		fail(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Roles retrieved", roles, query.Meta(page, total))
}

func (h *Handler) get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid role id", nil)
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, role)
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
	role, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, "Role created successfully", role)
}

func (h *Handler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid role id", nil)
		return
	}
	var request UpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	role, err := h.service.Update(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, role)
}

func (h *Handler) delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid role id", nil)
		return
	}
	if err := h.service.Delete(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *Handler) setPermissions(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid role id", nil)
		return
	}
	var request struct {
		Permissions []string `json:"permissions" validate:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	role, err := h.service.SetPermissions(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request.Permissions)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, role)
}

func fail(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response.Fail(c, appErr.Status, appErr.Message, appErr.Details)
		return
	}
	response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
}
