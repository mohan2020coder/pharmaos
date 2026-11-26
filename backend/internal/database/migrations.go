package database

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

var migrationName = regexp.MustCompile(`^(\d+)__([a-z0-9_]+)\.(up|down)\.sql$`)

type Migration struct {
	Version int
	Name    string
	Up      string
	Down    string
}

type appliedMigration struct {
	Name     string
	Checksum string
}

func checksum(sql string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(sql)))
}

func LoadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory %q: %w", dir, err)
	}
	byVersion := map[int]*Migration{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationName.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, err
		}
		migration := byVersion[version]
		if migration == nil {
			migration = &Migration{Version: version, Name: matches[2]}
			byVersion[version] = migration
		}
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if matches[3] == "up" {
			migration.Up = string(body)
		} else {
			migration.Down = string(body)
		}
	}
	versions := make([]int, 0, len(byVersion))
	for version := range byVersion {
		versions = append(versions, version)
	}
	sort.Ints(versions)
	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		migration := byVersion[version]
		if migration.Up == "" {
			return nil, fmt.Errorf("migration %d has no up file", version)
		}
		migrations = append(migrations, *migration)
	}
	return migrations, nil
}

func MigrateUp(ctx context.Context, dsn, dir string, target int) (int, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return 0, fmt.Errorf("connect for migrations: %w", err)
	}
	defer conn.Close(ctx)

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(76381922451)`); err != nil {
		return 0, err
	}
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock(76381922451)`)

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version BIGINT PRIMARY KEY,
		name TEXT NOT NULL,
		checksum TEXT NOT NULL,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return 0, err
	}
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return 0, err
	}
	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		return 0, err
	}
	last := 0
	for _, migration := range migrations {
		if target > 0 && migration.Version > target {
			break
		}
		if record, done := applied[migration.Version]; done {
			if record.Checksum != checksum(migration.Up) {
				return last, fmt.Errorf("migration %d (%s) was modified after it was applied", migration.Version, migration.Name)
			}
			last = migration.Version
			continue
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return last, err
		}
		if _, err = tx.Exec(ctx, migration.Up); err != nil {
			tx.Rollback(ctx)
			return last, fmt.Errorf("apply migration %d (%s): %w", migration.Version, migration.Name, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, md5($3))`,
			migration.Version, migration.Name, migration.Up); err != nil {
			tx.Rollback(ctx)
			return last, err
		}
		if err = tx.Commit(ctx); err != nil {
			return last, err
		}
		last = migration.Version
	}
	return last, nil
}

func MigrateDown(ctx context.Context, dsn, dir string, steps int) (int, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return 0, err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(76381922451)`); err != nil {
		return 0, err
	}
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock(76381922451)`)
	migrations, err := LoadMigrations(dir)
	if err != nil {
		return 0, err
	}
	byVersion := make(map[int]Migration, len(migrations))
	for _, m := range migrations {
		byVersion[m.Version] = m
	}
	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		return 0, err
	}
	for i := 0; i < steps; i++ {
		var version int
		var name string
		err = conn.QueryRow(ctx, `SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version, &name)
		if err != nil {
			return 0, err
		}
		migration := byVersion[version]
		if record, ok := applied[version]; !ok || record.Checksum != checksum(migration.Up) {
			return version, fmt.Errorf("migration %d (%s) was modified after it was applied", version, name)
		}
		if migration.Down == "" {
			return version, fmt.Errorf("migration %d has no down file", version)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return version, err
		}
		if _, err := tx.Exec(ctx, migration.Down); err != nil {
			tx.Rollback(ctx)
			return version, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version); err != nil {
			tx.Rollback(ctx)
			return version, err
		}
		if err := tx.Commit(ctx); err != nil {
			return version, err
		}
	}
	return 0, nil
}

func AppliedVersions(ctx context.Context, dsn string) ([]int, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, len(applied))
	for version := range applied {
		out = append(out, version)
	}
	sort.Ints(out)
	return out, nil
}

func appliedVersions(ctx context.Context, conn *pgx.Conn) (map[int]appliedMigration, error) {
	rows, err := conn.Query(ctx, `SELECT version, name, checksum FROM schema_migrations`)
	if err != nil {
		if strings.Contains(err.Error(), "schema_migrations") {
			return map[int]appliedMigration{}, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := map[int]appliedMigration{}
	for rows.Next() {
		var version int
		var record appliedMigration
		if err := rows.Scan(&version, &record.Name, &record.Checksum); err != nil {
			return nil, err
		}
		out[version] = record
	}
	return out, rows.Err()
}
