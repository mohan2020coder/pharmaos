import { get, post } from './client';
import type { ChangesResponse, PushResponse, SyncOperation, SyncStatus } from '../types';

export function getChanges(since: string, entities?: string[]): Promise<ChangesResponse> {
  const params = new URLSearchParams({ limit: '200' });
  if (since) {
    params.set('since', since);
  }
  if (entities?.length) {
    params.set('entities', entities.join(','));
  }
  return get<ChangesResponse>(`/api/v1/sync/changes?${params.toString()}`);
}

export function pushOperations(operations: SyncOperation[]): Promise<PushResponse> {
  return post<PushResponse>('/api/v1/sync/operations', { operations });
}

export function syncStatus(): Promise<SyncStatus> {
  return get<SyncStatus>('/api/v1/sync/status');
}
