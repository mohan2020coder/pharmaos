package organization

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
	group := router.Group("/organization")
	group.GET("/hierarchy", middleware.RequirePermission("employee.view"), h.hierarchy)
	group.GET("/branches", middleware.RequirePermission("employee.view"), h.list)
	group.POST("/branches", middleware.RequirePermission("employee.manage"), h.create)
}

func (h *Handler) hierarchy(c *gin.Context) {
	result, err := h.service.Hierarchy(c.Request.Context(), auth.MustPrincipal(c.Request.Context()))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Branches retrieved", items, query.Meta(page, total))
}

func (h *Handler) create(c *gin.Context) {
	var request BranchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	branch, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Branch created successfully", branch)
}
