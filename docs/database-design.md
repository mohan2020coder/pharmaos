# Database Design

> **Status:** Migrations `000001__foundation` through `000008__hardening` are present in the repository. Tables and policies are implemented through Phase 9; later sections remain **target design** for the follow-on backlog.

## 1. Migration source of truth

Schema changes live in `backend/migrations/` as paired files:

```text
000001__foundation.up.sql
000001__foundation.down.sql
NNNNNN__short_snake_case_name.up.sql
NNNNNN__short_snake_case_name.down.sql
```

`LoadMigrations` accepts `^(\d+)__([a-z0-9_]+)\.(up|down)\.sql$`, sorts by numeric version, and rejects a version without an `up` file. `MigrateUp` runs at process startup from `cmd/api/main.go`; `MigrateDown` is exposed via `api -migrate-down N`.

| Property | Current behavior |
| --- | --- |
| Ledger | `schema_migrations(version BIGINT PK, name TEXT, checksum TEXT, applied_at TIMESTAMPTZ)`; checksum is `md5(up_sql)`. |
| Concurrency | `pg_advisory_lock(76381922451)` held for the full up/down run. |
| Atomicity | Each migration's SQL and ledger insert/delete run in one transaction; failure rolls back. |
| Targets | `MigrateUp(ctx, dsn, dir, target)` with `target=0` means latest. `-migrate-down N` rolls back N versions. |
| Checksum use | `md5(up_sql)` is stored and verified before a migration is skipped or rolled back; edited applied migrations fail fast. |
| Readiness | `/readyz` compares the largest local migration version to the largest applied version and returns 503 if migrations are pending. |

`make migrate` currently runs `-migrate-down 0`, i.e. **rolls back every migration**. Treat it as a local reset helper, never as a production deploy step.

**Rules for contributors:** never edit an applied migration; add the next version; always provide a reversible `down` script; test with `make test-db`; keep production changes additive/backfill-first.

## 2. Implemented foundation (`000001`)

- Extensions: `pgcrypto` (`gen_random_uuid()`), `citext` (case-insensitive emails).
- `pharmaos_set_updated_at()` — trigger function setting `NEW.updated_at = now()`.
- `pharmaos_current_tenant_id()` — reads `current_setting('app.current_tenant_id', true)` as UUID, returns `NULL` when unset.
- `pharmaos_is_privileged()` — reads `current_setting('app.is_privileged', true)`, defaults to `false`.
- `pharmaos_touch_updated_at(table_name text)` — creates `trg_<table>_updated_at`; pass trusted identifiers only because it uses `format('%1$s')` without identifier quoting.

`TenantTransaction` sets both settings with `set_config(..., true)` so they are local to the transaction and cannot leak between pooled connections.

## 3. Naming and column conventions

| Item | Convention |
| --- | --- |
| Tables / columns | `snake_case`, singular domain prefix where useful (`crm_doctors`, `dist_orders`). |
| Primary key | `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`; offline records use a client-generated UUID. |
| Tenant key | `tenant_id uuid NOT NULL` on every tenant-owned table, including join and audit tables. |
| Actor fields | `created_by uuid`, `updated_by uuid`; immutable events also use `occurred_at`. |
| Timestamps | `created_at timestamptz NOT NULL DEFAULT now()`, `updated_at timestamptz NOT NULL DEFAULT now()`, optional `deleted_at timestamptz` for soft deletes. |
| External identities | `external_id citext` where case-insensitivity is required (e.g. email). |
| Money | `numeric(18,2)` plus `currency char(3)`; never floating point. |
| Indexes | Prefix tenant-specific indexes with `tenant_id`, e.g. `(tenant_id, status, created_at DESC)`. |
| Constraints | Database-level `UNIQUE` and `CHECK`; application validation is not sufficient. |

## 4. Table inventory

| Domain | Tables | Tenant owner | Phase |
| --- | --- | --- | --- |
| Identity | `tenants`, `users`, `user_roles`, `roles`, `permissions`, `role_permissions`, `sessions`, `refresh_tokens` | Tenant rows except global roles/permissions catalog | 2 |
| Organization | `divisions`, `territories`, `employees`, `employee_hierarchy`, `user_territories` | PHARMA_COMPANY | 3 |
| CRM | `doctors`, `chemists`, `stockists`, `doctor_territories`, `products`, `targets` | PHARMA_COMPANY | 4 |
| Field execution | `tour_plans`, `tour_plan_lines`, `visits`, `daily_call_reports`, `dcr_product_details`, `visit_attachments` | PHARMA_COMPANY | 5 (implemented in `000005`) |
| Distribution | `dist_distributors`, `dist_company_distributor_links`, `dist_orders`, `dist_order_items`, `dist_secondary_sales`, `dist_inventory`, `dist_inventory_movements` | Link-scoped; owner tenant plus active link | 7 |
| Operations | `audit_logs`, `idempotency_keys`, `sync_operations`, `outbox`, `int_integration_connections`, `int_webhook_deliveries` | Tenant + system | 6–8 |
| Reporting | `rpt_report_definitions`, `rpt_report_snapshots`, `dashboard_widgets` | Tenant | 8 (widgets planned) |

### 4.1 Tenants and company–distributor links

Tenant type is `PHARMA_COMPANY` or `DISTRIBUTOR`. A company may work with many distributors and a distributor may serve many companies; the relationship is explicitly many-to-many:

```sql
-- Planned (Phase 2/7)
CREATE TABLE company_distributor_links (
  id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  company_tenant_id     uuid NOT NULL REFERENCES tenants(id),
  distributor_tenant_id uuid NOT NULL REFERENCES tenants(id),
  status                text NOT NULL DEFAULT 'ACTIVE'
                        CHECK (status IN ('ACTIVE','SUSPENDED','ENDED')),
  effective_from        date NOT NULL DEFAULT CURRENT_DATE,
  effective_to          date,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT uq_company_distributor UNIQUE (company_tenant_id, distributor_tenant_id),
  CONSTRAINT ck_distinct_tenants CHECK (company_tenant_id <> distributor_tenant_id)
);
CREATE INDEX idx_cdl_distributor ON company_distributor_links
  (distributor_tenant_id, status);
```

Rows in this table are the only bridge that permits cross-tenant reads of shared distribution data. Both sides are evaluated in RLS policies; a distributor cannot read another distributor's links.

### 4.2 Example tenant-owned table with RLS

```sql
-- Implemented in migration 000004; helper functions are defined in migration 000001
CREATE TABLE crm_doctors (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL DEFAULT pharmaos_current_tenant_id(),
  external_id citext,
  name        text NOT NULL,
  specialty   text,
  territory_id uuid,
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  deleted_at  timestamptz,
  CONSTRAINT uq_doctors_external UNIQUE (tenant_id, external_id)
);
CREATE INDEX idx_doctors_tenant_active_name
  ON crm_doctors (tenant_id, is_active, name);
SELECT pharmaos_touch_updated_at('crm_doctors');

ALTER TABLE crm_doctors ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm_doctors FORCE ROW LEVEL SECURITY;
CREATE POLICY crm_doctors_tenant ON crm_doctors
  FOR ALL
  USING (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged())
  WITH CHECK (tenant_id = pharmaos_current_tenant_id() OR pharmaos_is_privileged());
```

The `DEFAULT` on `tenant_id` is a convenience for server-side inserts; the application must still supply/verify the token-derived tenant. `FORCE RLS` ensures even the table owner is subject to policies. Privileged bypass is reserved for audited platform operations and must never be set from a normal user token.

## 5. Indexing, growth and retention

- Always include `tenant_id` as the leading column on tenant-scoped secondary indexes so RLS predicates remain selective.
- Use `(tenant_id, updated_at, id)` keyset pagination cursors for sync and large lists; avoid deep `OFFSET`.
- Append-only tables (`audit_log`, `sync_operations`, `webhook_deliveries`) should be partitioned by month with tenant-local indexes; retain per contractual policy and export before pruning.
- `inventory_movements` and `secondary_sales` are event-sourced deltas; current stock is a materialized projection rebuilt from movements.
- Use foreign keys within a tenant where possible (composite `(tenant_id, id)` keys are preferred for hard isolation).
- Run `ANALYZE` after bulk backfills; track slow queries per tenant to catch missing composite indexes.

## 6. Transaction and idempotency records

`TenantTransaction` rolls back whenever the HTTP status is ≥ 400. Business writes that can arrive from mobile must additionally persist:

```text
idempotency_keys(
  tenant_id, key uuid, endpoint text, request_hash text,
  response_json jsonb, status_code int, created_at, expires_at,
  PRIMARY KEY (tenant_id, key)
)
```

The server stores the first result and replays it for the same `(tenant_id, key, endpoint)`. A different payload under the same key is a conflict. See [api-design.md](api-design.md) and [mobile-offline-sync.md](mobile-offline-sync.md).

Migration `000008__hardening` adds the non-tenant operational table `ops_rate_limit_counters(counter_key, window_start, hits, updated_at)` and `pharmaos_rate_limit(key, limit, window_seconds)`. The function is `SECURITY DEFINER`, grants only `EXECUTE` to `pharmaos_app`, atomically increments fixed-window counters, and occasionally prunes windows older than one hour. Rate-limit counters are operational metadata and intentionally carry no tenant identifier.

## 7. Testing and operations

- Unit: `LoadMigrations` validates file discovery; `make test-db` runs migrations against the `postgres_test` service on port 55433 (tmpfs, disposable).
- Add Phase 2+ integration tests that assert RLS denial with `SET LOCAL app.current_tenant_id`, cross-tenant UUID access, link-based access, and down/up round trips.
- Backups: managed PostgreSQL with PITR; verify tenant-scoped restore before production cutover.
- Tenant offboarding: export, verify, then delete in dependency order with an audited, transactional runbook (planned).

Related documents: [architecture.md](architecture.md), [tenant-isolation.md](tenant-isolation.md), [deployment.md](deployment.md).
