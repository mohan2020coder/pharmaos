package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
	"github.com/pharmaos/pharmaos/backend/internal/platform/response"
)

func Authenticate(tokens *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Fail(c, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		claims, err := tokens.ParseAccess(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Invalid or expired token", nil)
			return
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Invalid token subject", nil)
			return
		}
		tenantID, err := uuid.Parse(claims.TenantID)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Invalid tenant claim", nil)
			return
		}
		sessionID, err := uuid.Parse(claims.SessionID)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, "Invalid session claim", nil)
			return
		}
		principal := auth.Principal{
			UserID: userID, TenantID: tenantID, SessionID: sessionID, Email: claims.Email,
			Roles: claims.Roles, Permissions: map[string]bool{}, DataScope: claims.DataScope,
			DivisionIDs: parseUUIDs(claims.DivisionIDs), TerritoryIDs: parseUUIDs(claims.TerritoryIDs),
			IsPrivileged: claims.IsPrivileged || contains(claims.Roles, "SUPER_ADMIN"),
		}
		for _, permission := range claims.Permissions {
			principal.Permissions[permission] = true
		}
		c.Request = c.Request.WithContext(auth.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFrom(c.Request.Context())
		if !ok {
			response.Fail(c, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		if !principal.HasPermission(permission) {
			response.Fail(c, http.StatusForbidden, "Permission denied", gin.H{"required_permission": permission})
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := auth.PrincipalFrom(c.Request.Context())
		if !ok {
			response.Fail(c, http.StatusUnauthorized, "Authentication required", nil)
			return
		}
		for _, permission := range permissions {
			if principal.HasPermission(permission) {
				c.Next()
				return
			}
		}
		response.Fail(c, http.StatusForbidden, "Permission denied", nil)
	}
}

func parseUUIDs(values []string) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if parsed, err := uuid.Parse(value); err == nil {
			out = append(out, parsed)
		}
	}
	return out
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
