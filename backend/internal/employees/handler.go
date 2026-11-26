package employees

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
	group := router.Group("/employees")
	group.GET("", middleware.RequirePermission("employee.view"), h.list)
	group.POST("", middleware.RequirePermission("employee.manage"), h.create)
	group.GET("/me", h.me)
	group.GET("/:id", middleware.RequirePermission("employee.view"), h.get)
	group.GET("/:id/history", middleware.RequirePermission("employee.view"), h.history)
	group.POST("/:id/transfer", middleware.RequirePermission("employee.manage"), h.transfer)

	positions := router.Group("/positions")
	positions.GET("", middleware.RequirePermission("employee.view"), h.listPositions)
	positions.POST("", middleware.RequirePermission("employee.manage"), h.createPosition)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	divisionID := optionalUUID(c.Query("division_id"))
	territoryID := optionalUUID(c.Query("territory_id"))
	managerID := optionalUUID(c.Query("manager_id"))
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page, divisionID, territoryID, managerID)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Employees retrieved", items, query.Meta(page, total))
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
	employee, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Employee created successfully", employee)
}

func (h *Handler) me(c *gin.Context) {
	employee, err := h.service.Current(c.Request.Context(), auth.MustPrincipal(c.Request.Context()))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, employee)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	employee, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, employee)
}

func (h *Handler) history(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	history, err := h.service.History(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, history)
}

func (h *Handler) transfer(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request TransferRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	if err := h.service.Transfer(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request); err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, gin.H{"transferred": true})
}

func (h *Handler) listPositions(c *gin.Context) {
	positions, err := h.service.ListPositions(c.Request.Context(), auth.MustPrincipal(c.Request.Context()))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, positions)
}

func (h *Handler) createPosition(c *gin.Context) {
	var request struct {
		Code         string     `json:"code" validate:"required,min=2,max=40"`
		Title        string     `json:"title" validate:"required,min=2,max=120"`
		Level        int        `json:"level" validate:"required,min=1,max=12"`
		IsManagement bool       `json:"is_management"`
		ParentID     *uuid.UUID `json:"parent_position_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	position, err := h.service.CreatePosition(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		request.Code, request.Title, request.Level, request.IsManagement, request.ParentID)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Position created successfully", position)
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
