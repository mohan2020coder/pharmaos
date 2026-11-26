#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${RESTORE_DATABASE_URL:?RESTORE_DATABASE_URL is required}"

BACKUP_FILE="${1:-./pharmaos-backup.dump}"
pg_dump -Fc -f "$BACKUP_FILE" "$DATABASE_URL"
psql -v ON_ERROR_STOP=1 "$RESTORE_DATABASE_URL" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
pg_restore --no-owner --no-privileges -d "$RESTORE_DATABASE_URL" "$BACKUP_FILE"
psql -v ON_ERROR_STOP=1 "$RESTORE_DATABASE_URL" -c "SELECT count(*) AS migrations FROM schema_migrations;"
