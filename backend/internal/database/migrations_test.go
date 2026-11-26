package database

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadMigrations(t *testing.T) {
	migrations, err := LoadMigrations("../../migrations")
	require.NoError(t, err)
	require.NotEmpty(t, migrations)
	require.Equal(t, 1, migrations[0].Version)
	require.Contains(t, migrations[0].Up, "pgcrypto")
	require.Contains(t, migrations[0].Down, "DROP FUNCTION")
	for _, migration := range migrations {
		require.NotEmpty(t, migration.Up, "migration %d must have an up script", migration.Version)
	}
}

func TestMigrateUpAgainstTestDatabase(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	version, err := MigrateUp(ctx, dsn, "../../migrations", 0)
	require.NoError(t, err)
	require.GreaterOrEqual(t, version, 1)
	applied, err := AppliedVersions(ctx, dsn)
	require.NoError(t, err)
	require.Contains(t, applied, 1)
}
