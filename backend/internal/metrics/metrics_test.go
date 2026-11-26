package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCollectorRecordsRoutesAndSecurityEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	collector := New()
	router := gin.New()
	router.Use(collector.Middleware())
	router.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/denied", func(c *gin.Context) { c.Status(http.StatusForbidden) })
	router.GET("/metrics", collector.Handler())

	for path, expected := range map[string]int{
		"/ping":   http.StatusOK,
		"/denied": http.StatusForbidden,
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, expected, response.Code)
	}

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := response.Body.String()
	require.Contains(t, body, `pharmaos_http_requests_total{route="/ping",method="GET",status="2xx"} 1`)
	require.Contains(t, body, `pharmaos_security_events_total{event="forbidden"} 1`)
	require.Contains(t, body, "pharmaos_http_request_duration_seconds_count")
}
