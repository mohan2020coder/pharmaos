export type TokenPair = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  refresh_expires_in: number;
};

export type SessionUser = {
  id: string;
  tenant_id: string;
  email: string;
  roles: string[];
  permissions: string[];
  data_scope: string;
  is_privileged?: boolean;
  division_ids?: string[];
  territory_ids?: string[];
};

export * from './types/domain';

export type LoginResponse = {
  tokens: TokenPair;
  user: SessionUser;
};

export type AuthMe = SessionUser & {
  is_privileged: boolean;
  division_ids: string[];
  territory_ids: string[];
};

export type Change = {
  entity: string;
  entity_id: string;
  op: 'UPSERT' | 'DELETE';
  updated_at: string;
  payload: Record<string, unknown>;
};

export type ChangesResponse = {
  next_cursor: string;
  has_more: boolean;
  changes: Change[];
};

export type SyncOperation = {
  id: string;
  entity: string;
  entity_id: string;
  operation: string;
  payload: Record<string, unknown>;
  idempotency_key: string;
};

export type SyncOperationResult = {
  client_operation_id: string;
  status: 'APPLIED' | 'REPLAYED' | 'CONFLICT' | 'REJECTED';
  entity: string;
  entity_id?: string;
  error_code?: string;
  error_message?: string;
  retryable: boolean;
};

export type PushResponse = {
  operations: SyncOperationResult[];
};

export type SyncStatus = {
  server_time: string;
  latest_cursor: string;
  pending_operations: number;
  supported_entities: string[];
  entity_counts: Record<string, number>;
};

export type OutboxRow = {
  id: string;
  tenant_id: string;
  user_id: string;
  entity: string;
  entity_id: string;
  operation: string;
  payload_json: string;
  idempotency_key: string;
  status: 'PENDING' | 'FAILED' | 'CONFLICT' | 'SYNCED';
  attempts: number;
  last_error: string | null;
  error_code: string | null;
  resolution_hint: string | null;
  next_attempt_at: number | null;
  created_at: number;
  updated_at: number | null;
  server_id: string | null;
};
