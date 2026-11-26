package auth

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/audit"
	"github.com/pharmaos/pharmaos/backend/internal/database"
)

func TestLoginIssuesRotatableTokens(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	_, err := database.MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	db, err := database.Connect(dsn, 5, 2, gormlogger.Silent)
	require.NoError(t, err)
	defer db.Close()

	code := "AUTH" + strings.ToUpper(uuid.NewString()[:8])
	email := strings.ToLower(uuid.NewString()[:8]) + "@example.test"
	tx := db.WithContext(ctx).Begin()
	require.NoError(t, tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error)
	tenantID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO tenants (id, code, name, tenant_type) VALUES (?, ?, ?, 'PHARMA_COMPANY')`, tenantID, code, "Auth Test Tenant").Error)
	hash, err := bcrypt.GenerateFromPassword([]byte("TestPassword-123"), 10)
	require.NoError(t, err)
	userID := uuid.New()
	require.NoError(t, tx.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status)
		VALUES (?, ?, ?, ?, 'Test', 'User', 'ACTIVE')`, userID, tenantID, email, string(hash)).Error)
	var roleID string
	require.NoError(t, tx.Raw(`SELECT id::text FROM roles WHERE code = 'MR'`).Scan(&roleID).Error)
	require.NoError(t, tx.Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from)
		VALUES (?, ?, 'SELF', '00000000-0000-0000-0000-000000000000', CURRENT_DATE)`, userID, roleID).Error)
	require.NoError(t, tx.Commit().Error)

	tokens := NewManager("test-secret", 15*time.Minute, 24*time.Hour)
	service := NewService(db.DB, NewRepository(), tokens, audit.NewService(db.DB), 10, nil)
	pair, principal, err := service.Login(ctx, LoginRequest{TenantCode: code, Email: email, Password: "TestPassword-123"}, "127.0.0.1", "test")
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.Equal(t, tenantID, principal.TenantID)

	rotated, _, err := service.Refresh(ctx, pair.RefreshToken, "127.0.0.1", "test")
	require.NoError(t, err)
	require.NotEmpty(t, rotated.AccessToken)

	_, _, err = service.Refresh(ctx, pair.RefreshToken, "127.0.0.1", "test")
	require.Error(t, err, "reused refresh token must be rejected")
}
