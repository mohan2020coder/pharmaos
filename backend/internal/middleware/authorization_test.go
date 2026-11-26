package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/pharmaos/pharmaos/backend/internal/auth"
)

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/protected", RequirePermission("order.approve"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	principal := auth.Principal{UserID: uuid.New(), TenantID: uuid.New(), Permissions: map[string]bool{}}
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusForbidden, response.Code)

	principal.Permissions["order.approve"] = true
	request = httptest.NewRequest(http.MethodGet, "/protected", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), principal))
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code)
}
