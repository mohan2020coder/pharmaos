package reporting

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
	group := router.Group("/reports")
	group.GET("/field-activity", middleware.RequirePermission("reports.view"), h.fieldActivity)
	group.GET("/distribution", middleware.RequirePermission("reports.view"), h.distribution)
	group.GET("/audit", middleware.RequirePermission("audit.view"), h.audit)
	group.GET("/definitions", middleware.RequirePermission("reports.view"), h.listDefinitions)
	group.POST("/definitions", middleware.RequirePermission("report.manage"), h.createDefinition)
	group.GET("/snapshots", middleware.RequirePermission("reports.view"), h.listSnapshots)
	group.GET("/snapshots/:id", middleware.RequirePermission("reports.view"), h.getSnapshot)
	group.POST("/snapshots", middleware.RequirePermission("report.manage"), h.createSnapshot)
}

func (h *Handler) fieldActivity(c *gin.Context) {
	report, err := h.service.FieldActivity(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		c.Query("from"), c.Query("to"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, report)
}

func (h *Handler) distribution(c *gin.Context) {
	report, err := h.service.DistributionSummary(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		c.Query("from"), c.Query("to"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, report)
}

func (h *Handler) audit(c *gin.Context) {
	page := query.Parse(c)
	entries, total, err := h.service.AuditTrail(c.Request.Context(), auth.MustPrincipal(c.Request.Context()),
		page, c.Query("from"), c.Query("to"), c.Query("module"), c.Query("action"))
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Audit trail retrieved", entries, query.Meta(page, total))
}

func (h *Handler) listDefinitions(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.ListDefinitions(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Report definitions retrieved", items, query.Meta(page, total))
}

func (h *Handler) createDefinition(c *gin.Context) {
	var request ReportDefinitionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	record, err := h.service.CreateDefinition(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Report definition created", record)
}

func (h *Handler) listSnapshots(c *gin.Context) {
	page := query.Parse(c)
	items, total, err := h.service.ListSnapshots(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Report snapshots retrieved", items, query.Meta(page, total))
}

func (h *Handler) getSnapshot(c *gin.Context) {
	id, ok := httputil.UUID(c, "id")
	if !ok {
		return
	}
	record, err := h.service.GetSnapshot(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.OK(c, record)
}

func (h *Handler) createSnapshot(c *gin.Context) {
	var request SnapshotRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if fields := validate.Struct(request); fields != nil {
		httputil.Invalid(c, fields)
		return
	}
	record, err := h.service.CreateSnapshot(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Report snapshot created", record)
}
