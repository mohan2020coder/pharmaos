package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	apperrors "github.com/pharmaos/pharmaos/backend/internal/platform/errors"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
	"github.com/pharmaos/pharmaos/backend/internal/platform/validate"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterPublic(router *gin.RouterGroup) {
	group := router.Group("/auth")
	group.POST("/login", h.login)
	group.POST("/refresh", h.refresh)
	group.POST("/forgot-password", h.forgotPassword)
	group.POST("/reset-password", h.resetPassword)
}

func (h *Handler) RegisterProtected(router *gin.RouterGroup) {
	group := router.Group("/auth")
	group.POST("/logout", h.logout)
	group.GET("/me", h.me)
	group.GET("/sessions", h.sessions)
	group.POST("/change-password", h.changePassword)
}

func (h *Handler) login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	pair, principal, err := h.service.Login(c.Request.Context(), request, clientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, gin.H{"tokens": pair, "user": gin.H{
		"id": principal.UserID, "tenant_id": principal.TenantID, "email": principal.Email,
		"roles": principal.Roles, "permissions": keys(principal.Permissions), "data_scope": principal.DataScope,
	}})
}

func (h *Handler) refresh(c *gin.Context) {
	var request struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	pair, principal, err := h.service.Refresh(c.Request.Context(), request.RefreshToken, clientIP(c), c.GetHeader("User-Agent"))
	if err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, gin.H{"tokens": pair, "user": gin.H{"id": principal.UserID, "tenant_id": principal.TenantID, "roles": principal.Roles}})
}

func (h *Handler) logout(c *gin.Context) {
	principal := authPrincipal(c)
	if err := h.service.Logout(c.Request.Context(), principal.SessionID); err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, gin.H{"logged_out": true})
}

func (h *Handler) me(c *gin.Context) {
	principal := authPrincipal(c)
	response.OK(c, gin.H{
		"id": principal.UserID, "tenant_id": principal.TenantID, "email": principal.Email,
		"roles": principal.Roles, "permissions": keys(principal.Permissions), "data_scope": principal.DataScope,
		"is_privileged": principal.IsPrivileged, "division_ids": principal.DivisionIDs, "territory_ids": principal.TerritoryIDs,
	})
}

func (h *Handler) sessions(c *gin.Context) {
	sessions, err := h.service.Sessions(c.Request.Context(), authPrincipal(c).UserID)
	if err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, sessions)
}

func (h *Handler) changePassword(c *gin.Context) {
	var request struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=10"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), authPrincipal(c).UserID, request.CurrentPassword, request.NewPassword); err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, gin.H{"password_changed": true})
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var request struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	_ = h.service.ForgotPassword(c.Request.Context(), request.Email)
	response.OK(c, gin.H{"accepted": true, "message": "If the account exists, reset instructions have been sent"})
}

func (h *Handler) resetPassword(c *gin.Context) {
	var request struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=10"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid JSON payload", gin.H{"body": err.Error()})
		return
	}
	if errors := validate.Struct(request); errors != nil {
		response.Fail(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), request.Token, request.NewPassword); err != nil {
		failAuth(c, err)
		return
	}
	response.OK(c, gin.H{"password_reset": true})
}

func authPrincipal(c *gin.Context) Principal {
	if principal, ok := PrincipalFrom(c.Request.Context()); ok {
		return principal
	}
	return Principal{UserID: uuid.Nil}
}

func failAuth(c *gin.Context, err error) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		response.Fail(c, appErr.Status, appErr.Message, appErr.Details)
		return
	}
	response.Fail(c, http.StatusInternalServerError, "Internal server error", nil)
}

func keys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}

func clientIP(c *gin.Context) string {
	if value := c.GetString("ip"); value != "" {
		return value
	}
	return c.ClientIP()
}
