package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/pharmaos/pharmaos/backend/internal/config"
)

func TestLivenessAndDocs(t *testing.T) {
	app := New(config.Config{Environment: "test", RateLimitPerMin: 1000, CORSOrigins: []string{"http://localhost"}, MigrationsDir: "../../migrations"}, nil, zap.NewNop())
	for _, path := range []string{"/docs", "/docs/openapi.yaml"} {
		recorder := httptest.NewRecorder()
		app.Engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusOK, recorder.Code, path)
	}
}
