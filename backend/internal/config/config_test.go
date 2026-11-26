package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "development", cfg.Environment)
	require.Equal(t, "test-secret", cfg.JWTSecret)
	require.Positive(t, cfg.RateLimitPerMin)
}

func TestProductionRequiresJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	_, err := Load()
	require.Error(t, err)
}

func TestProductionHardeningDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "0123456789012345678901234567890123456789")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "postgres", cfg.RateLimitBackend)
	require.False(t, cfg.AllowInsecureWebhooks)
}

func TestProductionRejectsUnsafeWebhooks(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "0123456789012345678901234567890123456789")
	t.Setenv("ALLOW_INSECURE_WEBHOOKS", "true")
	_, err := Load()
	require.Error(t, err)
}

func TestTrustedProxiesValidation(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8,not-a-cidr")
	_, err := Load()
	require.Error(t, err)
}
