package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment           string
	HTTPAddr              string
	DatabaseURL           string
	RedisURL              string
	JWTSecret             string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	PasswordHashCost      int
	CORSOrigins           []string
	RateLimitPerMin       int
	MigrationsDir         string
	LogLevel              string
	ShutdownTimeout       time.Duration
	WebhookWorkerInterval time.Duration
	WebhookMaxAttempts    int
	WebhookTimeout        time.Duration
	AllowInsecureWebhooks bool
	RateLimitBackend      string
	TrustedProxies        []string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:           env("APP_ENV", "development"),
		HTTPAddr:              env("HTTP_ADDR", ":8080"),
		DatabaseURL:           env("DATABASE_URL", "postgres://pharmaos:pharmaos@localhost:5432/pharmaos?sslmode=disable"),
		RedisURL:              env("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:             env("JWT_SECRET", ""),
		AccessTokenTTL:        durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL:       durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		PasswordHashCost:      intEnv("PASSWORD_HASH_COST", 12),
		CORSOrigins:           splitList(env("CORS_ORIGINS", "http://localhost:3000,http://localhost:19006")),
		RateLimitPerMin:       intEnv("RATE_LIMIT_PER_MIN", 300),
		MigrationsDir:         env("MIGRATIONS_DIR", "migrations"),
		LogLevel:              env("LOG_LEVEL", "info"),
		ShutdownTimeout:       durationEnv("SHUTDOWN_TIMEOUT", 15*time.Second),
		WebhookWorkerInterval: durationEnv("WEBHOOK_WORKER_INTERVAL", 5*time.Second),
		WebhookMaxAttempts:    intEnv("WEBHOOK_MAX_ATTEMPTS", 8),
		WebhookTimeout:        durationEnv("WEBHOOK_TIMEOUT", 10*time.Second),
		RateLimitBackend:      env("RATE_LIMIT_BACKEND", "memory"),
		TrustedProxies:        splitList(env("TRUSTED_PROXIES", "")),
	}
	if cfg.Environment == "production" && os.Getenv("RATE_LIMIT_BACKEND") == "" {
		cfg.RateLimitBackend = "postgres"
	}
	cfg.AllowInsecureWebhooks = boolEnv("ALLOW_INSECURE_WEBHOOKS", cfg.Environment != "production")

	if cfg.JWTSecret == "" && cfg.Environment == "production" {
		return Config{}, fmt.Errorf("JWT_SECRET must be set in production")
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = "development-only-secret-change-me"
	}
	if cfg.PasswordHashCost < 10 || cfg.PasswordHashCost > 15 {
		return Config{}, fmt.Errorf("PASSWORD_HASH_COST must be between 10 and 15")
	}
	if cfg.WebhookMaxAttempts < 1 || cfg.WebhookMaxAttempts > 20 {
		return Config{}, fmt.Errorf("WEBHOOK_MAX_ATTEMPTS must be between 1 and 20")
	}
	if cfg.WebhookWorkerInterval <= 0 || cfg.WebhookTimeout <= 0 {
		return Config{}, fmt.Errorf("WEBHOOK_WORKER_INTERVAL and WEBHOOK_TIMEOUT must be positive")
	}
	if cfg.RateLimitBackend != "memory" && cfg.RateLimitBackend != "postgres" {
		return Config{}, fmt.Errorf("RATE_LIMIT_BACKEND must be memory or postgres")
	}
	if cfg.RateLimitPerMin <= 0 {
		return Config{}, fmt.Errorf("RATE_LIMIT_PER_MIN must be positive")
	}
	if cfg.Environment == "production" {
		if len(cfg.JWTSecret) < 32 {
			return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 bytes in production")
		}
		if cfg.RateLimitBackend != "postgres" {
			return Config{}, fmt.Errorf("RATE_LIMIT_BACKEND=postgres is required in production")
		}
		if cfg.AllowInsecureWebhooks {
			return Config{}, fmt.Errorf("ALLOW_INSECURE_WEBHOOKS must be false in production")
		}
		if len(cfg.CORSOrigins) == 0 || containsWildcard(cfg.CORSOrigins) {
			return Config{}, fmt.Errorf("CORS_ORIGINS must be an explicit allowlist in production")
		}
	}
	for _, value := range cfg.TrustedProxies {
		if _, _, err := net.ParseCIDR(value); err != nil {
			if net.ParseIP(value) == nil {
				return Config{}, fmt.Errorf("TRUSTED_PROXIES contains invalid CIDR or IP: %s", value)
			}
		}
	}
	return cfg, nil
}

func containsWildcard(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "*" {
			return true
		}
	}
	return false
}

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	value, err := strconv.ParseBool(env(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(env(key, ""))
	if err != nil {
		return fallback
	}
	return value
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
