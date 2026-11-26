package dcr

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
	group := router.Group("/dcr")
	group.GET("", middleware.RequirePermission("dcr.view"), h.list)
	group.POST("", middleware.RequirePermission("dcr.create"), h.create)
	group.GET("/:id", middleware.RequirePermission("dcr.view"), h.get)
	group.GET("/:id/products", middleware.RequirePermission("dcr.view"), h.products)
	group.POST("/:id/products", middleware.RequirePermission("dcr.create"), h.addProduct)
	group.POST("/:id/submit", middleware.RequirePermission("dcr.create"), h.submit)
	group.POST("/:id/approve", middleware.RequirePermission("dcr.review"), h.approve)
	group.POST("/:id/reject", middleware.RequirePermission("dcr.review"), h.reject)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	employeeID := optionalUUID(c.Query("employee_id"))
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page,
		employeeID, c.Query("status"), c.Query("from"), c.Query("to"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Daily call reports retrieved", items, query.Meta(page, total))
}

func (h *Handler) create(c *gin.Context) {
	var request CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	report, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Daily call report created", report)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	report, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, report)
}

func (h *Handler) products(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	details, err := h.service.Products(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, details)
}

func (h *Handler) addProduct(c *gin.Context) {
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
	detail, err := h.service.AddProduct(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Product detail added", detail)
}

func (h *Handler) submit(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	report, err := h.service.Submit(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, report)
}

func (h *Handler) approve(c *gin.Context) {
	h.decision(c, true)
}

func (h *Handler) reject(c *gin.Context) {
	h.decision(c, false)
}

func (h *Handler) decision(c *gin.Context, approve bool) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request DecisionRequest
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&request)
	}
	report, err := h.service.Decide(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, approve, request.Remarks)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, report)
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
