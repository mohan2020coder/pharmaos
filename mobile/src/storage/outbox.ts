import type { SQLiteDatabase } from 'expo-sqlite';

import type { OutboxRow } from '../types';
import { getDatabase } from './database';

export type EnqueueOperation = {
  id: string;
  tenantId: string;
  userId: string;
  entity: string;
  entityId: string;
  operation: string;
  payload: Record<string, unknown>;
  idempotencyKey: string;
};

export async function enqueueOperation(database: SQLiteDatabase, operation: EnqueueOperation): Promise<void> {
  await database.runAsync(
    `INSERT INTO outbox_operations
      (id, tenant_id, user_id, entity, entity_id, operation, payload_json, idempotency_key, status, attempts, created_at, updated_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'PENDING', 0, ?, ?)`,
    operation.id,
    operation.tenantId,
    operation.userId,
    operation.entity,
    operation.entityId,
    operation.operation,
    JSON.stringify(operation.payload),
    operation.idempotencyKey,
    Date.now(),
    Date.now(),
  );
}

export async function listPendingOperations(tenantId: string, limit = 100): Promise<OutboxRow[]> {
  const database = await getDatabase();
  return database.getAllAsync<OutboxRow>(
    `SELECT * FROM outbox_operations
     WHERE tenant_id = ? AND status = 'PENDING'
       AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
     ORDER BY created_at ASC LIMIT ?`,
    tenantId,
    Date.now(),
    limit,
  );
}

export async function listProblemOperations(tenantId: string, limit = 100): Promise<OutboxRow[]> {
  const database = await getDatabase();
  return database.getAllAsync<OutboxRow>(
    `SELECT * FROM outbox_operations
     WHERE tenant_id = ? AND status IN ('FAILED', 'CONFLICT')
     ORDER BY COALESCE(updated_at, created_at) DESC LIMIT ?`,
    tenantId,
    limit,
  );
}

export async function markSynced(database: SQLiteDatabase, operationId: string, serverId?: string): Promise<void> {
  await database.runAsync(
    `UPDATE outbox_operations
     SET status = 'SYNCED', server_id = COALESCE(?, server_id),
         last_error = NULL, error_code = NULL, resolution_hint = NULL, next_attempt_at = NULL,
         updated_at = ?
     WHERE id = ?`,
    serverId ?? null,
    Date.now(),
    operationId,
  );
}

export async function markFailed(
  database: SQLiteDatabase,
  operationId: string,
  error: string,
  retryable: boolean,
  errorCode = 'SYNC_ERROR',
): Promise<void> {
  const row = await database.getFirstAsync<{ attempts: number }>(
    'SELECT attempts FROM outbox_operations WHERE id = ?',
    operationId,
  );
  const attempts = (row?.attempts ?? 0) + 1;
  const terminal = !retryable || attempts >= 8;
  const backoff = Math.min(60000, 1000 * 2 ** Math.min(attempts, 6));
  await database.runAsync(
    `UPDATE outbox_operations
     SET status = ?, attempts = ?, last_error = ?, error_code = ?, resolution_hint = ?,
         next_attempt_at = ?, updated_at = ?
     WHERE id = ?`,
    terminal ? 'FAILED' : 'PENDING',
    attempts,
    error.slice(0, 500),
    errorCode,
    terminal ? 'REVIEW_OR_RETRY' : 'AUTO_RETRY',
    terminal ? null : Date.now() + backoff,
    Date.now(),
    operationId,
  );
}

export async function markConflict(
  database: SQLiteDatabase,
  operationId: string,
  errorCode: string,
  errorMessage: string,
): Promise<void> {
  const row = await database.getFirstAsync<{ attempts: number }>(
    'SELECT attempts FROM outbox_operations WHERE id = ?',
    operationId,
  );
  await database.runAsync(
    `UPDATE outbox_operations
     SET status = 'CONFLICT', attempts = ?, last_error = ?, error_code = ?,
         resolution_hint = 'USE_SERVER_OR_DISCARD', next_attempt_at = NULL, updated_at = ?
     WHERE id = ?`,
    (row?.attempts ?? 0) + 1,
    errorMessage.slice(0, 500),
    errorCode || 'CONFLICT',
    Date.now(),
    operationId,
  );
}

export async function retryOperation(database: SQLiteDatabase, operationId: string): Promise<void> {
  await database.runAsync(
    `UPDATE outbox_operations
     SET status = 'PENDING', attempts = 0, next_attempt_at = NULL, last_error = NULL, error_code = NULL,
         resolution_hint = NULL, updated_at = ?
     WHERE id = ?`,
    Date.now(),
    operationId,
  );
}

export async function resolveOperation(
  database: SQLiteDatabase,
  operationId: string,
  resolution: 'USE_SERVER' | 'DISCARD',
): Promise<void> {
  await database.runAsync(
    `UPDATE outbox_operations
     SET status = 'SYNCED', next_attempt_at = NULL, last_error = NULL, error_code = NULL,
         resolution_hint = ?, updated_at = ?
     WHERE id = ?`,
    resolution,
    Date.now(),
    operationId,
  );
}

export async function requeueWithNewIdempotencyKey(
  database: SQLiteDatabase,
  operationId: string,
  idempotencyKey: string,
): Promise<void> {
  await database.runAsync(
    `UPDATE outbox_operations
     SET idempotency_key = ?, status = 'PENDING', attempts = 0, next_attempt_at = NULL,
         last_error = NULL, error_code = NULL, resolution_hint = NULL, updated_at = ?
     WHERE id = ?`,
    idempotencyKey,
    Date.now(),
    operationId,
  );
}

export async function pendingCount(tenantId: string): Promise<number> {
  const database = await getDatabase();
  const row = await database.getFirstAsync<{ total: number }>(
    `SELECT COUNT(*) AS total FROM outbox_operations
     WHERE tenant_id = ? AND status = 'PENDING'`,
    tenantId,
  );
  return row?.total ?? 0;
}

export async function problemCount(tenantId: string): Promise<number> {
  const database = await getDatabase();
  const row = await database.getFirstAsync<{ total: number }>(
    `SELECT COUNT(*) AS total FROM outbox_operations
     WHERE tenant_id = ? AND status IN ('FAILED', 'CONFLICT')`,
    tenantId,
  );
  return row?.total ?? 0;
}
