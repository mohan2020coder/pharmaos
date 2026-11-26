package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/server"
)

func TestReportingAuditAndWebhookQueue(t *testing.T) {
	dsn := envOrSkip(t)
	ctx := context.Background()
	_, err := database.MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	db, err := database.Connect(dsn, 10, 5, gormlogger.Silent)
	require.NoError(t, err)
	defer db.Close()

	suffix := strings.ToUpper(uuid.NewString()[:8])
	code := "RPT" + suffix
	email := "rpt-" + suffix + "@example.test"
	tenantID, _ := createTenantUser(t, ctx, db, code, email)

	cfg := config.Config{
		Environment: "test", DatabaseURL: dsn, JWTSecret: "insights-test-secret",
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour,
		PasswordHashCost: 10, RateLimitPerMin: 10000, CORSOrigins: []string{},
		MigrationsDir: "../../migrations", AllowInsecureWebhooks: true,
		WebhookMaxAttempts: 3, WebhookTimeout: time.Second, WebhookWorkerInterval: time.Hour,
	}
	app := httptest.NewServer(server.New(cfg, db, zap.NewNop()).Engine)
	defer app.Close()
	token := login(t, app.URL, code, email)

	field := request(t, app.URL, http.MethodGet, "/api/v1/reports/field-activity", token, nil)
	require.Equal(t, http.StatusOK, field.StatusCode, readBody(t, field))
	var fieldEnvelope struct {
		Data struct {
			Scope string `json:"scope"`
		} `json:"data"`
	}
	decode(t, field, &fieldEnvelope)
	require.Equal(t, "TENANT", fieldEnvelope.Data.Scope)

	distribution := request(t, app.URL, http.MethodGet, "/api/v1/reports/distribution", token, nil)
	require.Equal(t, http.StatusOK, distribution.StatusCode, readBody(t, distribution))

	definition := request(t, app.URL, http.MethodPost, "/api/v1/reports/definitions", token, map[string]interface{}{
		"code": "DAILY_" + suffix, "name": "Daily activity", "report_type": "FIELD_ACTIVITY",
		"config": map[string]interface{}{"granularity": "day"},
	})
	require.Equal(t, http.StatusCreated, definition.StatusCode, readBody(t, definition))

	connection := request(t, app.URL, http.MethodPost, "/api/v1/integrations/connections", token, map[string]interface{}{
		"name": "Test webhook", "kind": "WEBHOOK", "endpoint_url": "http://127.0.0.1:9/hook",
		"secret_ref": "TEST_WEBHOOK_SECRET", "headers": map[string]string{"X-Test": "true"},
	})
	require.Equal(t, http.StatusCreated, connection.StatusCode, readBody(t, connection))
	var connectionEnvelope struct {
		Data struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decode(t, connection, &connectionEnvelope)
	require.NotEqual(t, uuid.Nil, connectionEnvelope.Data.ID)

	eventID := "event-" + suffix
	published := request(t, app.URL, http.MethodPost, "/api/v1/integrations/events", token, map[string]interface{}{
		"event_type": "order.approved", "event_id": eventID,
		"payload": map[string]interface{}{"tenant_id": tenantID.String()},
	})
	require.Equal(t, http.StatusCreated, published.StatusCode, readBody(t, published))
	var publishEnvelope struct {
		Data []struct {
			EventID string `json:"event_id"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	decode(t, published, &publishEnvelope)
	require.Len(t, publishEnvelope.Data, 1)
	require.Equal(t, eventID, publishEnvelope.Data[0].EventID)

	deliveries := request(t, app.URL, http.MethodGet, "/api/v1/integrations/deliveries", token, nil)
	require.Equal(t, http.StatusOK, deliveries.StatusCode, readBody(t, deliveries))
	var deliveryEnvelope struct {
		Data []jsonRaw `json:"data"`
	}
	decode(t, deliveries, &deliveryEnvelope)
	require.Len(t, deliveryEnvelope.Data, 1)

	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error)
	err = tx.Exec(`UPDATE audit_logs SET action = 'TAMPERED'
		WHERE id = (SELECT id FROM audit_logs WHERE tenant_id = ? ORDER BY id DESC LIMIT 1)`, tenantID).Error
	require.Error(t, err)
	require.Contains(t, err.Error(), "append-only")
	require.NoError(t, tx.Rollback().Error)
}

type jsonRaw map[string]interface{}
