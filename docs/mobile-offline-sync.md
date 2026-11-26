# Mobile Offline Sync

> **Status:** **Implemented in Phase 6 and extended in Phase 7.** The `mobile/` Expo/React Native client and `/api/v1/sync/*` endpoints implement the contract below. Tokens are stored in `expo-secure-store`; SQLite holds tenant-scoped read models, sync cursors, and a durable outbox. Phase 7 adds distribution read models, order capture, and order/secondary-sale/inventory operations.

## 1. Principles

- Field users must be able to create visits, DCR entries, tour-plan updates and orders without connectivity.
- Every device write is recorded in a **durable SQLite outbox** in the same local transaction as the domain row.
- Every operation has a **client-generated UUID** and a stable **idempotency key**, so retries and restarts are safe.
- The server is authoritative for cross-tenant distribution data, inventory quantities and workflow state.
- No real patient data or prescription-level patient information is stored locally or remotely.
- Switching tenants or logging out wipes tenant data and pending operations.

## 2. Client storage model

```sql
-- SQLite, one logical store per installed app; all rows include tenant_id
CREATE TABLE outbox_operations (
  id                TEXT PRIMARY KEY,          -- UUID v4/v7 from client
  tenant_id         TEXT NOT NULL,
  user_id           TEXT NOT NULL,
  entity            TEXT NOT NULL,             -- visit, dcr, order, ...
  entity_id         TEXT NOT NULL,
  operation         TEXT NOT NULL,             -- CREATE / UPDATE / SUBMIT
  payload_json      TEXT NOT NULL,
  idempotency_key   TEXT NOT NULL,
  status            TEXT NOT NULL DEFAULT 'PENDING',
  attempts          INTEGER NOT NULL DEFAULT 0,
  last_error        TEXT,
  next_attempt_at   INTEGER,                   -- epoch ms
  created_at        INTEGER NOT NULL,
  UNIQUE (tenant_id, idempotency_key)
);
CREATE INDEX idx_outbox_ready
  ON outbox_operations (tenant_id, status, next_attempt_at);

CREATE TABLE sync_state (
  tenant_id         TEXT PRIMARY KEY,
  last_cursor       TEXT,
  last_synced_at    INTEGER,
  bootstrap_complete INTEGER NOT NULL DEFAULT 0
);
```

Read models (`doctors`, `products`, `territories`, `tour_plans`, `visits`, `distributors`, `distribution_links`, `orders`, `order_items`, `inventory`, `inventory_movements`, `secondary_sales`) are local projections keyed by `(tenant_id, server_id)`. Local-only records keep their UUID; the server accepts that UUID as the record ID rather than reassigning it.

Use an encrypted SQLite container where the platform permits it, secure token storage, and truncation of sensitive logs. Keep the queue bounded with a dead-letter state and user-visible sync status.

## 3. Write path on device

1. User action creates/updates the local row and an outbox row atomically.
2. UI shows `PENDING` / `SYNCED` / `FAILED` state; offline edits remain editable unless the entity is immutable after submit.
3. Sync worker runs on connectivity changes, app resume and a periodic timer. It drains `PENDING` operations ordered by `created_at` per tenant.
4. Each batch operation carries `idempotency_key` and the client operation ID; successful results update the local row's server version, then mark the outbox row `SYNCED`. Direct HTTP writes use the `Idempotency-Key` header.
5. Transient failures use exponential backoff with jitter. `401` triggers token refresh; `403` marks the operation terminal and raises a user-visible permissions error; `409` invokes the conflict policy.
6. A batch partially succeeds: per-operation results are applied individually; never roll back already-applied operations on the client.

## 4. Server sync endpoints

| Route | Purpose |
| --- | --- |
| `GET /api/v1/sync/changes?since=<cursor>&limit=<n>` | Pull server changes with an opaque keyset cursor for the principal's tenant and data scope. |
| `POST /api/v1/sync/operations` | Push a batch of outbox operations; returns one result per operation. |
| `GET /api/v1/sync/status` | Queue health, latest cursor, server time for clock-skew correction. |

Pull responses use the standard envelope with `data.next_cursor`, `data.has_more`, `data.changes[]` and server timestamps. Push responses include `data.operations[]` with `status` of `APPLIED`, `REPLAYED`, `CONFLICT` or `REJECTED`. Supported operations are `visit.create`, `visit.submit`, `visit.attachment`, `dcr.create`, `dcr.product.add`, `dcr.submit`, `dcr.approve`, `dcr.reject`, `tour.create`, `tour.line.add`, `tour.submit`, `tour.approve`, `tour.reject`, `order.create`, `order.submit`, `order.approve`, `secondary_sale.create`, and `inventory.movement`. Batch entries must match the token's tenant.

## 5. Conflict policy

| Entity | Rule |
| --- | --- |
| Visits / DCR | Immutable after `SUBMIT`; duplicate submit is `REPLAYED` with the original ID, not a second record. Edits before submit use device timestamp and last-write-wins within the owner's scope. |
| Tour plans | Server version check; `409 STALE_VERSION` requires a pull and user-mediated merge. |
| Orders | Server-authoritative; idempotency key guarantees at most one order. Validation failures are terminal, not retried blindly. |
| Doctors / chemists / stockists / products / targets | Server-owned master data; device edits are read-only and overwritten on pull. |
| Inventory | Send movement deltas, never absolute stock snapshots; server serializes per tenant/distributor/SKU. |
| Secondary sales | Idempotent create; duplicates resolve to the existing server record. |

## 6. Sync sequence

```text
login/token refresh
  -> bootstrap reference data (territories, products, users in scope)
  -> GET /sync/changes (repeat until has_more=false)
  -> POST /sync/operations (drain outbox, apply per-item results)
  -> reconcile local IDs/versions and cursor
  -> background: repeat on connectivity, app resume, timer
```

Use the server's `sync/status` timestamp rather than device time for cursors; rely on `(updated_at, id)` keyset pagination so equal timestamps remain stable. deletions are tombstoned with a `deleted_at` marker until all devices advance past them.

## 7. Security and privacy

- Tokens in platform secure storage; never in SQLite plaintext or logs.
- Tenant ID in every local row; on tenant switch/logout, delete local data, outbox, cursors and cached attachments.
- TLS only; `Authorization` header sent over HTTPS; consider certificate pinning for production builds.
- GPS check-in for visits requires explicit permission and is stored only with the visit business record.
- Request IDs from sync failures should be surfaced to support with user consent; no credentials or patient data in diagnostic bundles.

## 8. Observability

Track per tenant/device: queue depth, oldest pending age, attempt count, success/replay/conflict/reject rates, pull latency and cursor lag. Log structured client events with request ID and operation ID; alert on dead-letter growth. The server should record idempotency hits and sync batch sizes in metrics.

## 9. Test matrix

| Case | Expected |
| --- | --- |
| Create visit offline, sync twice | One server visit; second call is a replay |
| App killed mid-batch | Unacknowledged operations remain pending and are retried safely |
| Duplicate idempotency key with different payload | `409`; no overwrite of stored response |
| Token expires during push | Refresh then resume; no duplicate writes |
| Permission revoked while offline | Operation rejected with terminal `403`; user notified |
| Link to distributor ends while offline | Cached cross-tenant data purged; new operations denied |
| Stale tour plan version | `409`, local merge prompt, no silent overwrite |
| Tenant switch with pending queue | Queue and tenant data wiped before new tenant bootstrap |
| Clock skew on device | Cursor advances using server timestamps |
| Server schema migration adds field | Old client payloads remain valid; unknown fields ignored |

Related documents: [api-design.md](api-design.md), [tenant-isolation.md](tenant-isolation.md), [implementation-roadmap.md](implementation-roadmap.md).
