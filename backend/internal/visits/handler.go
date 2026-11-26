package visits

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
	group := router.Group("/visits")
	group.GET("", middleware.RequirePermission("visit.view"), h.list)
	group.POST("", middleware.RequirePermission("visit.create"), h.create)
	group.GET("/:id", middleware.RequirePermission("visit.view"), h.get)
	group.POST("/:id/submit", middleware.RequirePermission("visit.create"), h.submit)
	group.POST("/:id/attachments", middleware.RequirePermission("visit.create"), h.addAttachment)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	employeeID := optionalUUID(c.Query("employee_id"))
	territoryID := optionalUUID(c.Query("territory_id"))
	items, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page,
		employeeID, territoryID, c.Query("entity_type"), c.Query("status"), c.Query("from"), c.Query("to"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Visits retrieved", items, query.Meta(page, total))
}

func (h *Handler) create(c *gin.Context) {
	idempotencyKey, err := uuid.Parse(c.GetHeader("Idempotency-Key"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid or missing Idempotency-Key header", map[string]string{"Idempotency-Key": "must be a UUID"})
		return
	}
	var request CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	visit, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request, idempotencyKey)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Visit created successfully", visit)
}

func (h *Handler) get(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	visit, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, visit)
}

func (h *Handler) submit(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	visit, err := h.service.Submit(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, visit)
}

func (h *Handler) addAttachment(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	var request AttachmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	attachment, err := h.service.AddAttachment(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Attachment recorded", attachment)
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
