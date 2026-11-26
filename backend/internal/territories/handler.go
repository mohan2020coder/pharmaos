package territories

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
	regions := router.Group("/regions")
	regions.GET("", middleware.RequirePermission("employee.view"), h.listRegions)
	regions.POST("", middleware.RequirePermission("employee.manage"), h.createRegion)
	territories := router.Group("/territories")
	territories.GET("", middleware.RequirePermission("employee.view"), h.listTerritories)
	territories.POST("", middleware.RequirePermission("employee.manage"), h.createTerritory)
}

func (h *Handler) listRegions(c *gin.Context) {
	page := query.Parse(c)
	divisionID := optionalUUID(c.Query("division_id"))
	items, total, err := h.service.ListRegions(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page, divisionID)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Regions retrieved", items, query.Meta(page, total))
}

func (h *Handler) createRegion(c *gin.Context) {
	var request RegionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	item, err := h.service.CreateRegion(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Region created successfully", item)
}

func (h *Handler) listTerritories(c *gin.Context) {
	page := query.Parse(c)
	divisionID := optionalUUID(c.Query("division_id"))
	regionID := optionalUUID(c.Query("region_id"))
	items, total, err := h.service.ListTerritories(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page, divisionID, regionID)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Territories retrieved", items, query.Meta(page, total))
}

func (h *Handler) createTerritory(c *gin.Context) {
	var request TerritoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httputil.BadJSON(c, err)
		return
	}
	if errors := validate.Struct(request); errors != nil {
		httputil.Invalid(c, errors)
		return
	}
	item, err := h.service.CreateTerritory(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		httputil.FailError(c, err)
		return
	}
	response.Created(c, "Territory created successfully", item)
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
