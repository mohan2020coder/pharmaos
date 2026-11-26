package users

import (
	"net/http"
	"time"

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
	group := router.Group("/users")
	group.GET("", middleware.RequirePermission("user.view"), h.list)
	group.GET("/:id", middleware.RequirePermission("user.view"), h.get)
	group.POST("", middleware.RequirePermission("user.manage"), h.create)
	group.PATCH("/:id", middleware.RequirePermission("user.manage"), h.update)
	group.POST("/:id/deactivate", middleware.RequirePermission("user.manage"), h.deactivate)
	group.GET("/:id/roles", middleware.RequirePermission("user.view"), h.roles)
	group.POST("/:id/roles", middleware.RequirePermission("user.manage"), h.assignRole)
	group.DELETE("/:id/roles/:role_id", middleware.RequirePermission("user.manage"), h.removeRole)
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
	user, err := h.service.Create(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), request)
	if err != nil {
		fail(c, err)
		return
	}
	response.Created(c, "User created successfully", user)
}

func (h *Handler) list(c *gin.Context) {
	page := query.Parse(c)
	users, total, err := h.service.List(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), page)
	if err != nil {
		fail(c, err)
		return
	}
	response.JSON(c, http.StatusOK, "Users retrieved", users, query.Meta(page, total))
}

func (h *Handler) get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", gin.H{"id": "must be a UUID"})
		return
	}
	user, err := h.service.Get(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, user)
}

func (h *Handler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", nil)
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
	user, err := h.service.Update(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id, request)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, user)
}

func (h *Handler) deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", nil)
		return
	}
	if err := h.service.Deactivate(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"deactivated": true})
}

func (h *Handler) roles(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", nil)
		return
	}
	grants, err := h.service.RolesForUser(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), id)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, grants)
}

func (h *Handler) assignRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", nil)
		return
	}
	var request struct {
		RoleID    uuid.UUID  `json:"role_id" validate:"required,uuid"`
		ScopeType string     `json:"scope_type" validate:"required,oneof=SELF TERRITORY REGION DIVISION TENANT PARTNER"`
		ScopeID   *uuid.UUID `json:"scope_id"`
		ValidFrom string     `json:"valid_from"`
		ValidTo   string     `json:"valid_to"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	validFromValue, err := parseDate(request.ValidFrom)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid valid_from date", nil)
		return
	}
	var validFrom time.Time
	if validFromValue != nil {
		validFrom = *validFromValue
	}
	validTo, err := parseDate(request.ValidTo)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid valid_to date", nil)
		return
	}
	err = h.service.AssignRole(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), userID, request.RoleID, request.ScopeType, request.ScopeID, validFrom, validTo)
	if err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"assigned": true})
}

func (h *Handler) removeRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid user id", nil)
		return
	}
	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid role id", nil)
		return
	}
	scopeType := c.Query("scope_type")
	if scopeType == "" {
		scopeType = "TENANT"
	}
	var scopeID *uuid.UUID
	if raw := c.Query("scope_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "Invalid scope_id", nil)
			return
		}
		scopeID = &parsed
	}
	if err := h.service.RemoveRole(c.Request.Context(), auth.MustPrincipal(c.Request.Context()), userID, roleID, scopeType, scopeID); err != nil {
		fail(c, err)
		return
	}
	response.OK(c, gin.H{"removed": true})
}

func parseDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func fail(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response.Fail(c, appErr.Status, appErr.Message, appErr.Details)
		return
	}
	response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
}
