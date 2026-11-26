package divisions

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
	group := router.Group("/divisions")
	group.GET("", middleware.RequirePermission("employee.view"), h.list)
	group.GET("/:id", middleware.RequirePermission("employee.view"), h.get)
	group.POST("", middleware.RequirePermission("employee.manage"), h.create)
	group.PATCH("/:id", middleware.RequirePermission("employee.manage"), h.update)
	group.DELETE("/:id", middleware.RequirePermission("employee.manage"), h.delete)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Divisions retrieved", items, query.Meta(page, total))
}

func (h *Handler) get(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *Handler) create(c *gin.Context) {
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	item, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Division created successfully", item)
}

func (h *Handler) update(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	if err := h.service.Update(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
