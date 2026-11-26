package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/server"
)

func TestTenantIsolation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	_, err := database.MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	db, err := database.Connect(dsn, 10, 5, gormlogger.Silent)
	require.NoError(t, err)
	defer db.Close()

	suffix := strings.ToUpper(uuid.NewString()[:8])
	tenantA, userA := createTenantUser(t, ctx, db, "TENA"+suffix, "a-"+suffix+"@example.test")
	_, userB := createTenantUser(t, ctx, db, "TENB"+suffix, "b-"+suffix+"@example.test")

	cfg := config.Config{
		Environment: "test", DatabaseURL: dsn, JWTSecret: "isolation-test-secret",
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour,
		PasswordHashCost: 10, RateLimitPerMin: 10000, CORSOrigins: []string{},
		MigrationsDir: "../../migrations",
	}
	app := httptest.NewServer(server.New(cfg, db, zap.NewNop()).Engine)
	defer app.Close()

	tokenA := login(t, app.URL, "TENA"+suffix, "a-"+suffix+"@example.test")
	tokenB := login(t, app.URL, "TENB"+suffix, "b-"+suffix+"@example.test")

	list := request(t, app.URL, http.MethodGet, "/api/v1/users", tokenA, nil)
	require.Equal(t, http.StatusOK, list.StatusCode)
	var listed struct {
		Data []struct {
			ID       uuid.UUID `json:"id"`
			TenantID uuid.UUID `json:"tenant_id"`
		} `json:"data"`
	}
	decode(t, list, &listed)
	require.Len(t, listed.Data, 1)
	require.Equal(t, userA, listed.Data[0].ID)
	require.Equal(t, tenantA, listed.Data[0].TenantID)

	cross := request(t, app.URL, http.MethodGet, "/api/v1/users/"+userB.String(), tokenA, nil)
	require.Equal(t, http.StatusNotFound, cross.StatusCode, "tenant A must not read tenant B user")

	forbidden := request(t, app.URL, http.MethodPost, "/api/v1/users", tokenA, map[string]interface{}{
		"tenant_id": userB.String(), "email": "x-" + suffix + "@example.test", "password": "StrongPassword-123",
		"first_name": "Cross", "last_name": "Tenant",
	})
	require.Equal(t, http.StatusForbidden, forbidden.StatusCode, "non-privileged user must not create users in other tenants")

	unauthorized := request(t, app.URL, http.MethodGet, "/api/v1/users", "", nil)
	require.Equal(t, http.StatusUnauthorized, unauthorized.StatusCode)

	probe := db.WithContext(ctx).Begin()
	require.NoError(t, probe.Exec(`SET LOCAL ROLE pharmaos_app`).Error)
	require.NoError(t, probe.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantA.String()).Error)
	var visible int
	var currentTenant, privileged string
	require.NoError(t, probe.Raw(`SELECT pharmaos_current_tenant_id()::text, pharmaos_is_privileged()::text`).Row().Scan(&currentTenant, &privileged))
	require.NoError(t, probe.Raw(`SELECT count(*) FROM users`).Scan(&visible).Error)
	t.Logf("RLS probe: tenant=%s privileged=%s visible=%d expected=%s", currentTenant, privileged, visible, tenantA.String())
	require.Equal(t, 1, visible, "RLS must hide other tenants' users")
	probe.Rollback()

	_ = tokenB
}

func createTenantUser(t *testing.T, ctx context.Context, db *database.DB, code, email string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error)
	tenantID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO tenants (id, code, name, tenant_type) VALUES (?, ?, ?, 'PHARMA_COMPANY')`, tenantID, code, "Isolation Tenant").Error)
	hash, err := bcrypt.GenerateFromPassword([]byte("TestPassword-123"), 10)
	require.NoError(t, err)
	userID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status)
		VALUES (?, ?, ?, ?, 'Isolation', 'Tester', 'ACTIVE')`, userID, tenantID, email, string(hash)).Error)
	var roleID string
	require.NoError(t, tx.Raw(`SELECT id::text FROM roles WHERE code = 'COMPANY_ADMIN'`).Scan(&roleID).Error)
	require.NoError(t, tx.Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from)
		VALUES (?, ?, 'TENANT', '00000000-0000-0000-0000-000000000000', CURRENT_DATE)`, userID, roleID).Error)
	require.NoError(t, tx.Commit().Error)
	return tenantID, userID
}

func login(t *testing.T, baseURL, tenantCode, email string) string {
	t.Helper()
	response := request(t, baseURL, http.MethodPost, "/api/v1/auth/login", "", map[string]interface{}{
		"tenant_code": tenantCode, "email": email, "password": "TestPassword-123",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %s", readBody(t, response))
	}
	var payload struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	decode(t, response, &payload)
	require.NotEmpty(t, payload.Data.Tokens.AccessToken)
	return payload.Data.Tokens.AccessToken
}

func request(t *testing.T, baseURL, method, path, token string, body interface{}) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return response
}

func decode(t *testing.T, response *http.Response, target interface{}) {
	t.Helper()
	defer response.Body.Close()
	require.NoError(t, json.NewDecoder(response.Body).Decode(target))
}

func readBody(t *testing.T, response *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	response.Body = io.NopCloser(bytes.NewReader(body))
	return string(body)
}
