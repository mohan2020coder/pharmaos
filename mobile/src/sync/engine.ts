import { getChanges, pushOperations, syncStatus } from '../api/sync';
import { getDatabase, getSyncState, saveChanges } from '../storage/database';
import { listPendingOperations, markConflict, markFailed, markSynced } from '../storage/outbox';
import { getUser } from '../storage/secure';
import type { OutboxRow, SyncOperation } from '../types';

export type SyncSummary = {
  changesPulled: number;
  operationsPushed: number;
  applied: number;
  conflicts: number;
  failures: number;
};

function toSyncOperation(row: OutboxRow): SyncOperation {
  return {
    id: row.id,
    entity: row.entity,
    entity_id: row.entity_id,
    operation: row.operation,
    payload: JSON.parse(row.payload_json) as Record<string, unknown>,
    idempotency_key: row.idempotency_key,
  };
}

export async function syncNow(): Promise<SyncSummary> {
  const user = await getUser();
  if (!user) {
    throw new Error('No active session');
  }
  const summary: SyncSummary = { changesPulled: 0, operationsPushed: 0, applied: 0, conflicts: 0, failures: 0 };
  const state = await getSyncState(user.tenant_id);
  let cursor = state.last_cursor;
  let pages = 0;

  while (pages < 20) {
    const response = await getChanges(cursor);
    await saveChanges(user.tenant_id, response.changes, response.next_cursor);
    summary.changesPulled += response.changes.length;
    pages += 1;
    if (!response.has_more || !response.next_cursor || response.next_cursor === cursor) {
      cursor = response.next_cursor;
      break;
    }
    cursor = response.next_cursor;
  }

  const pending = await listPendingOperations(user.tenant_id, 100);
  if (pending.length === 0) {
    return summary;
  }

  try {
    const response = await pushOperations(pending.map(toSyncOperation));
    const database = await getDatabase();
    await database.withTransactionAsync(async () => {
      for (const result of response.operations) {
        const operation = pending.find((item) => item.id === result.client_operation_id);
        if (!operation) {
          continue;
        }
        if (result.status === 'APPLIED' || result.status === 'REPLAYED') {
          await markSynced(database, operation.id, result.entity_id);
          summary.applied += 1;
          continue;
        }
        if (result.status === 'CONFLICT') {
          summary.conflicts += 1;
          await markConflict(
            database,
            operation.id,
            result.error_code || 'CONFLICT',
            result.error_message || 'The server rejected this operation as a conflict.',
          );
          continue;
        }
        summary.failures += 1;
        await markFailed(
          database,
          operation.id,
          result.error_message || result.error_code || result.status,
          result.retryable,
          result.error_code || result.status,
        );
      }
    });
    summary.operationsPushed = response.operations.length;
  } catch (error) {
    const database = await getDatabase();
    const message = error instanceof Error ? error.message : 'Push failed';
    await database.withTransactionAsync(async () => {
      for (const operation of pending) {
        await markFailed(database, operation.id, message, true, 'NETWORK');
      }
    });
    summary.failures += pending.length;
    summary.operationsPushed = pending.length;
  }
  return summary;
}

export function serverTime(): Promise<string> {
  return syncStatus().then((status) => status.server_time);
}
