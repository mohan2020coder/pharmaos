import * as SQLite from 'expo-sqlite';

import type { Change } from '../types';

let databasePromise: Promise<SQLite.SQLiteDatabase> | null = null;

const SCHEMA = `
CREATE TABLE IF NOT EXISTS outbox_operations (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  entity TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',
  attempts INTEGER NOT NULL DEFAULT 0,
  last_error TEXT,
  error_code TEXT,
  resolution_hint TEXT,
  next_attempt_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER,
  server_id TEXT,
  UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_outbox_ready
  ON outbox_operations (tenant_id, status, next_attempt_at);
CREATE TABLE IF NOT EXISTS sync_state (
  tenant_id TEXT PRIMARY KEY,
  last_cursor TEXT,
  last_synced_at INTEGER,
  bootstrap_complete INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS read_models (
  tenant_id TEXT NOT NULL,
  entity TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  op TEXT NOT NULL,
  PRIMARY KEY (tenant_id, entity, entity_id)
);
CREATE INDEX IF NOT EXISTS idx_read_models_entity
  ON read_models (tenant_id, entity, updated_at);
`;

async function ensureColumns(database: SQLite.SQLiteDatabase): Promise<void> {
  const migrations = [
    'ALTER TABLE outbox_operations ADD COLUMN error_code TEXT',
    'ALTER TABLE outbox_operations ADD COLUMN resolution_hint TEXT',
    'ALTER TABLE outbox_operations ADD COLUMN updated_at INTEGER'
  ];
  for (const statement of migrations) {
    try {
      await database.runAsync(statement);
    } catch {}
  }
}

export function getDatabase(): Promise<SQLite.SQLiteDatabase> {
  if (!databasePromise) {
    databasePromise = SQLite.openDatabaseAsync('pharmaos.db').then(async (database) => {
      await database.execAsync(SCHEMA);
      await ensureColumns(database);
      return database;
    });
  }
  return databasePromise;
}

export type SyncState = {
  last_cursor: string;
  last_synced_at: number | null;
  bootstrap_complete: boolean;
};

export async function getSyncState(tenantId: string): Promise<SyncState> {
  const database = await getDatabase();
  const row = await database.getFirstAsync<{
    last_cursor: string | null;
    last_synced_at: number | null;
    bootstrap_complete: number;
  }>('SELECT last_cursor, last_synced_at, bootstrap_complete FROM sync_state WHERE tenant_id = ?', tenantId);
  return {
    last_cursor: row?.last_cursor || '',
    last_synced_at: row?.last_synced_at ?? null,
    bootstrap_complete: Boolean(row?.bootstrap_complete),
  };
}

export async function saveChanges(tenantId: string, changes: Change[], nextCursor: string): Promise<void> {
  const database = await getDatabase();
  await database.withTransactionAsync(async () => {
    for (const change of changes) {
      if (change.op === 'DELETE') {
        await database.runAsync(
          'DELETE FROM read_models WHERE tenant_id = ? AND entity = ? AND entity_id = ?',
          tenantId,
          change.entity,
          change.entity_id,
        );
        continue;
      }
      await database.runAsync(
        `INSERT INTO read_models (tenant_id, entity, entity_id, payload_json, updated_at, op)
         VALUES (?, ?, ?, ?, ?, 'UPSERT')
         ON CONFLICT(tenant_id, entity, entity_id) DO UPDATE SET
           payload_json = excluded.payload_json,
           updated_at = excluded.updated_at,
           op = excluded.op`,
        tenantId,
        change.entity,
        change.entity_id,
        JSON.stringify(change.payload),
        change.updated_at,
      );
    }
    await database.runAsync(
      `INSERT INTO sync_state (tenant_id, last_cursor, last_synced_at, bootstrap_complete)
       VALUES (?, ?, ?, 1)
       ON CONFLICT(tenant_id) DO UPDATE SET
         last_cursor = excluded.last_cursor,
         last_synced_at = excluded.last_synced_at,
         bootstrap_complete = 1`,
      tenantId,
      nextCursor,
      Date.now(),
    );
  });
}

export async function clearTenantData(tenantId: string): Promise<void> {
  const database = await getDatabase();
  await database.withTransactionAsync(async () => {
    await database.runAsync('DELETE FROM outbox_operations WHERE tenant_id = ?', tenantId);
    await database.runAsync('DELETE FROM read_models WHERE tenant_id = ?', tenantId);
    await database.runAsync('DELETE FROM sync_state WHERE tenant_id = ?', tenantId);
  });
}

export async function countReadModels(tenantId: string, entity?: string): Promise<number> {
  const database = await getDatabase();
  const row = await database.getFirstAsync<{ total: number }>(
    entity
      ? 'SELECT COUNT(*) AS total FROM read_models WHERE tenant_id = ? AND entity = ?'
      : 'SELECT COUNT(*) AS total FROM read_models WHERE tenant_id = ?',
    entity ? [tenantId, entity] : [tenantId],
  );
  return row?.total ?? 0;
}
