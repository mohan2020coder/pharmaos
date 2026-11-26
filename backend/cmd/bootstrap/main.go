package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pharmaos/pharmaos/backend/internal/config"
	"github.com/pharmaos/pharmaos/backend/internal/database"
	"github.com/pharmaos/pharmaos/backend/internal/platform/logging"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger, err := logging.New(cfg.LogLevel, cfg.Environment == "production")
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")))
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if email == "" || password == "" || len(password) < 12 {
		fmt.Fprintln(os.Stderr, "BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD (>= 12 chars) are required")
		os.Exit(1)
	}
	code := strings.ToUpper(strings.TrimSpace(env("BOOTSTRAP_TENANT_CODE", "PHARMAOS")))
	name := env("BOOTSTRAP_TENANT_NAME", "PharmaOS Platform")

	db, err := database.Connect(cfg.DatabaseURL, 5, 2, gormlogger.Warn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := context.Background()
	tx := db.WithContext(ctx).Begin()
	if err := tx.Exec(`SELECT set_config('app.is_privileged', 'true', true)`).Error; err != nil {
		panic(err)
	}
	var tenantRaw string
	err = tx.Raw(`SELECT id::text FROM tenants WHERE code = ?`, code).Scan(&tenantRaw).Error
	tenantID, err2 := uuid.Parse(tenantRaw)
	if err != nil || err2 != nil || tenantID == uuid.Nil {
		tenantID = uuid.New()
		if err := tx.Exec(`INSERT INTO tenants (id, code, name, tenant_type, status, is_platform, settings)
			VALUES (?, ?, ?, 'PHARMA_COMPANY', 'ACTIVE', true, '{}'::jsonb)`, tenantID, code, name).Error; err != nil {
			panic(err)
		}
	}
	if err := tx.Exec(`SELECT set_config('app.current_tenant_id', ?, true)`, tenantID.String()).Error; err != nil {
		panic(err)
	}
	var userRaw string
	err = tx.Raw(`SELECT id::text FROM users WHERE tenant_id = ? AND email = ?`, tenantID, email).Scan(&userRaw).Error
	userID, err2 := uuid.Parse(userRaw)
	if err != nil || err2 != nil || userID == uuid.Nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), cfg.PasswordHashCost)
		if err != nil {
			panic(err)
		}
		userID = uuid.New()
		if err := tx.Exec(`INSERT INTO users (id, tenant_id, email, password_hash, first_name, last_name, status, email_verified)
			VALUES (?, ?, ?, ?, 'Platform', 'Administrator', 'ACTIVE', true)`, userID, tenantID, email, string(hash)).Error; err != nil {
			panic(err)
		}
	}
	var roleRaw string
	if err := tx.Raw(`SELECT id::text FROM roles WHERE code = 'SUPER_ADMIN'`).Scan(&roleRaw).Error; err != nil {
		panic(err)
	}
	roleID, err := uuid.Parse(roleRaw)
	if err != nil || roleID == uuid.Nil {
		panic("SUPER_ADMIN role not found; run migrations first")
	}
	if err := tx.Exec(`INSERT INTO user_roles (user_id, role_id, scope_type, scope_id, valid_from)
		VALUES (?, ?, 'TENANT', '00000000-0000-0000-0000-000000000000', CURRENT_DATE)
		ON CONFLICT (user_id, role_id, scope_type, scope_id) DO NOTHING`, userID, roleID).Error; err != nil {
		panic(err)
	}
	if err := tx.Commit().Error; err != nil {
		panic(err)
	}
	logger.Info("bootstrap complete", zap.String("tenant_code", code), zap.String("admin_email", email))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
