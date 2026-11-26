package tours

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
	group := router.Group("/tour-plans")
	group.GET("", middleware.RequirePermission("tourplan.view"), h.list)
	group.POST("", middleware.RequirePermission("tourplan.create"), h.create)
	group.GET("/:id", middleware.RequirePermission("tourplan.view"), h.get)
	group.POST("/:id/submit", middleware.RequirePermission("tourplan.create"), h.submit)
	group.POST("/:id/approve", middleware.RequirePermission("tourplan.approve"), h.approve)
	group.POST("/:id/reject", middleware.RequirePermission("tourplan.approve"), h.reject)
	group.GET("/:id/lines", middleware.RequirePermission("tourplan.view"), h.lines)
	group.POST("/:id/lines", middleware.RequirePermission("tourplan.create"), h.addLine)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	employeeID := optionalUUID(c.Query("employee_id"))
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page, employeeID, c.Query("status"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Tour plans retrieved", items, query.Meta(page, total))
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
	plan, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Tour plan created successfully", plan)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	plan, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, plan)
}

func (h *Handler) submit(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	plan, err := h.service.Submit(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, plan)
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
	plan, err := h.service.Decide(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, approve, request.Comments)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, plan)
}

func (h *Handler) lines(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	lines, err := h.service.Lines(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, lines)
}

func (h *Handler) addLine(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request LineRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	line, err := h.service.AddLine(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Tour plan line added", line)
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
