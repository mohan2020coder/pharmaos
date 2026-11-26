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

func TestDistributionLinkIdempotencyAndInventory(t *testing.T) {
	dsn := envOrSkip(t)
	ctx := context.Background()
	_, err := database.MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	db, err := database.Connect(dsn, 10, 5, gormlogger.Silent)
	require.NoError(t, err)
	defer db.Close()

	suffix := strings.ToUpper(uuid.NewString()[:8])
	companyCode := "DCO" + suffix
	distributorCode := "DST" + suffix
	companyID, _ := createTenantUser(t, ctx, db, companyCode, "dco-"+suffix+"@example.test")
	distributorID, _ := createDistributorTenantUser(t, ctx, db, distributorCode, "dst-"+suffix+"@example.test")
	productID := uuid.New()
	seedDistributionCatalog(t, ctx, db, companyID, distributorID, productID)

	cfg := config.Config{
		Environment: "test", DatabaseURL: dsn, JWTSecret: "distribution-test-secret",
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour,
		PasswordHashCost: 10, RateLimitPerMin: 10000, CORSOrigins: []string{},
		MigrationsDir: "../../migrations",
	}
	app := httptest.NewServer(server.New(cfg, db, zap.NewNop()).Engine)
	defer app.Close()
	companyToken := login(t, app.URL, companyCode, "dco-"+suffix+"@example.test")
	distributorToken := login(t, app.URL, distributorCode, "dst-"+suffix+"@example.test")

	linkResponse := requestWithKey(t, app.URL, http.MethodPost, "/api/v1/distribution-links", companyToken, "", map[string]interface{}{
		"distributor_tenant_id": distributorID,
		"status":                "ACTIVE",
	})
	require.Equal(t, http.StatusCreated, linkResponse.StatusCode, readBody(t, linkResponse))
	var linkEnvelope struct {
		Data struct {
			ID     uuid.UUID `json:"id"`
			Status string    `json:"status"`
		} `json:"data"`
	}
	decode(t, linkResponse, &linkEnvelope)
	require.Equal(t, "ACTIVE", linkEnvelope.Data.Status)

	orderBody := map[string]interface{}{
		"distributor_tenant_id": distributorID,
		"order_number":          "ORD-" + suffix,
		"items": []map[string]interface{}{
			{"product_id": productID, "quantity": 5},
		},
	}
	idemKey := uuid.NewString()
	created := requestWithKey(t, app.URL, http.MethodPost, "/api/v1/orders", companyToken, idemKey, orderBody)
	require.Equal(t, http.StatusCreated, created.StatusCode, readBody(t, created))
	var orderEnvelope struct {
		Data struct {
			ID            uuid.UUID `json:"id"`
			Status        string    `json:"status"`
			TotalAmount   float64   `json:"total_amount"`
			DistributorID uuid.UUID `json:"distributor_tenant_id"`
		} `json:"data"`
	}
	decode(t, created, &orderEnvelope)
	require.Equal(t, "DRAFT", orderEnvelope.Data.Status)
	require.Equal(t, distributorID, orderEnvelope.Data.DistributorID)
	require.Greater(t, orderEnvelope.Data.TotalAmount, 0.0)

	replayed := requestWithKey(t, app.URL, http.MethodPost, "/api/v1/orders", companyToken, idemKey, orderBody)
	require.Equal(t, http.StatusCreated, replayed.StatusCode, readBody(t, replayed))
	var replayEnvelope struct {
		Data struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decode(t, replayed, &replayEnvelope)
	require.Equal(t, orderEnvelope.Data.ID, replayEnvelope.Data.ID)

	submitted := request(t, app.URL, http.MethodPost, "/api/v1/orders/"+orderEnvelope.Data.ID.String()+"/submit", companyToken, nil)
	require.Equal(t, http.StatusOK, submitted.StatusCode, readBody(t, submitted))
	approved := request(t, app.URL, http.MethodPost, "/api/v1/orders/"+orderEnvelope.Data.ID.String()+"/approve", companyToken, nil)
	require.Equal(t, http.StatusOK, approved.StatusCode, readBody(t, approved))
	duplicateApproval := request(t, app.URL, http.MethodPost, "/api/v1/orders/"+orderEnvelope.Data.ID.String()+"/approve", companyToken, nil)
	require.Equal(t, http.StatusUnprocessableEntity, duplicateApproval.StatusCode, readBody(t, duplicateApproval))

	stock := request(t, app.URL, http.MethodGet, "/api/v1/inventory?product_id="+productID.String(), distributorToken, nil)
	require.Equal(t, http.StatusOK, stock.StatusCode, readBody(t, stock))
	var stockEnvelope struct {
		Data []struct {
			QuantityReserved int `json:"quantity_reserved"`
		} `json:"data"`
	}
	decode(t, stock, &stockEnvelope)
	require.Len(t, stockEnvelope.Data, 1)
	require.Equal(t, 5, stockEnvelope.Data[0].QuantityReserved)

	suspended := request(t, app.URL, http.MethodPatch, "/api/v1/distribution-links/"+linkEnvelope.Data.ID.String()+"/status", companyToken, map[string]interface{}{
		"status": "SUSPENDED",
	})
	require.Equal(t, http.StatusOK, suspended.StatusCode, readBody(t, suspended))

	hidden := request(t, app.URL, http.MethodGet, "/api/v1/inventory?product_id="+productID.String(), companyToken, nil)
	require.Equal(t, http.StatusOK, hidden.StatusCode, readBody(t, hidden))
	var hiddenEnvelope struct {
		Data []json.RawMessage `json:"data"`
	}
	decode(t, hidden, &hiddenEnvelope)
	require.Empty(t, hiddenEnvelope.Data, "company must lose linked inventory access after link suspension")
}

func envOrSkip(t *testing.T) string {
	t.Helper()
	value := os.Getenv("TEST_DATABASE_URL")
	if value == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	return value
}

func createDistributorTenantUser(t *testing.T, ctx context.Context, db *database.DB, code, email string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error)
	tenantID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO tenants (id, code, name, tenant_type) VALUES (?, ?, ?, 'DISTRIBUTOR')`, tenantID, code, "Distribution Tenant").Error)
	hash, err := bcrypt.GenerateFromPassword([]byte("TestPassword-123"), 10)
	require.NoError(t, err)
	userID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status)
		VALUES (?, ?, ?, ?, 'Distributor', 'Admin', 'ACTIVE')`, userID, tenantID, email, string(hash)).Error)
	var roleID string
	require.NoError(t, tx.Raw(`SELECT id::text FROM roles WHERE code = 'DISTRIBUTOR_ADMIN'`).Scan(&roleID).Error)
	require.NoError(t, tx.Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from)
		VALUES (?, ?, 'TENANT', '00000000-0000-0000-0000-000000000000', CURRENT_DATE)`, userID, roleID).Error)
	require.NoError(t, tx.Commit().Error)
	return tenantID, userID
}

func seedDistributionCatalog(t *testing.T, ctx context.Context, db *database.DB, companyID, distributorID, productID uuid.UUID) {
	t.Helper()
	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error)
	require.NoError(t, tx.Exec(`INSERT INTO cat_products (id, tenant_id, code, name, mrp, is_active)
		VALUES (?, ?, ?, ?, 12.50, true)`, productID, companyID, "DP-"+uuid.NewString()[:8], "Distribution Product").Error)
	require.NoError(t, tx.Exec(`INSERT INTO dist_inventory
		(id, tenant_id, product_id, warehouse_code, quantity_on_hand, quantity_reserved, version)
		VALUES (?, ?, ?, 'MAIN', 100, 0, 1)`, uuid.New(), distributorID, productID).Error)
	require.NoError(t, tx.Commit().Error)
}

func requestWithKey(t *testing.T, baseURL, method, path, token, idempotencyKey string, body interface{}) *http.Response {
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
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return response
}
