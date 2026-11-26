package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/server"
)

func TestDistributedRateLimitAndMetrics(t *testing.T) {
	dsn := envOrSkip(t)
	ctx := context.Background()
	_, err := database.MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	db, err := database.Connect(dsn, 10, 5, gormlogger.Silent)
	require.NoError(t, err)
	defer db.Close()

	key := "test:" + t.Name()
	for i := 0; i < 2; i++ {
		var allowed bool
		require.NoError(t, db.WithContext(ctx).Raw(`SELECT pharmaos_rate_limit(?, ?, 60)`, key, 2).Scan(&allowed).Error)
		require.True(t, allowed)
	}
	var allowed bool
	require.NoError(t, db.WithContext(ctx).Raw(`SELECT pharmaos_rate_limit(?, ?, 60)`, key, 2).Scan(&allowed).Error)
	require.False(t, allowed)

	concurrentKey := "test:concurrent:" + t.Name()
	const limit = 5
	const attempts = 20
	results := make(chan bool, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var ok bool
			if err := db.WithContext(ctx).Raw(`SELECT pharmaos_rate_limit(?, ?, 60)`, concurrentKey, limit).Scan(&ok).Error; err != nil {
				errs <- err
				return
			}
			results <- ok
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	allowedCount := 0
	for ok := range results {
		if ok {
			allowedCount++
		}
	}
	require.Equal(t, limit, allowedCount)

	cfg := config.Config{
		Environment: "test", DatabaseURL: dsn, JWTSecret: "hardening-test-secret",
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour,
		PasswordHashCost: 10, RateLimitPerMin: 10000, RateLimitBackend: "memory",
		CORSOrigins: []string{}, MigrationsDir: "../../migrations",
	}
	app := httptest.NewServer(server.New(cfg, db, zap.NewNop()).Engine)
	defer app.Close()

	response, err := http.Get(app.URL + "/metrics")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
}
