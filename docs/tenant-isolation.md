# Tenant Isolation

> **Status:** Tenant tables, role/permission grants, request middleware, service predicates, RLS policies, Phase 7 distribution link tables, Phase 8 reporting/integration tables, and Phase 9 hardening controls are implemented. Business routes for Phases 2–9 are mounted under `/api/v1` or the observability namespace. Cross-tenant access is only allowed through active, effective-dated `dist_company_distributor_links` records; webhook workers and PostgreSQL-backed rate limiting use privileged database functions/local GUCs rather than tenant headers.

## 1. Tenant model

| Tenant type | Purpose | Typical data |
| --- | --- | --- |
| `PHARMA_COMPANY` | Manufacturer/importer running the field force | Users, hierarchy, doctors, tour plans, visits/DCR, products, targets, orders placed on distributors |
| `DISTRIBUTOR` | Distribution partner | Own users, inventory, received orders, secondary sales |

Pharma companies and distributors are separate tenant rows. Their commercial relationship is an explicit many-to-many `dist_company_distributor_links` record with status and effective dates. Nothing else implies cross-tenant visibility.

## 2. Authority chain

1. **JWT is the only source of tenant identity.** Claims carry `tenant_id`, `user_id`, `session_id`, roles, permissions, data scope and scope IDs. `auth.Principal` already models these fields in `backend/internal/auth/principal.go`.
2. **Authentication middleware (implemented)** verifies signature, expiry and session state, then stores the principal on the request context with `auth.WithPrincipal`.
3. **TenantTransaction (implemented)** starts a GORM transaction and executes:
   ```sql
   SELECT set_config('app.current_tenant_id', '<uuid>', true);
   SELECT set_config('app.is_privileged', '<bool>', true);
   ```
   `true` makes the settings transaction-local, so pooled connections never inherit another request's tenant. A status ≥ 400 or GORM error rolls back; otherwise the transaction commits.
4. **Service/repository layer (implemented)** applies tenant and data-scope predicates to every query; distribution services add explicit active-link checks before cross-tenant reads or writes.
5. **PostgreSQL RLS (implemented)** re-checks the same tenant using `pharmaos_current_tenant_id()` and link-aware helpers. A query that forgets a predicate returns no rows rather than another tenant's data.

`app.current_tenant_id` is not set for privileged sessions; `app.is_privileged` is. Normal user tokens must always set `is_privileged=false`.

## 3. RLS policy pattern

Helpers already exist in `000001__foundation.up.sql`:

| Function | Behavior |
| --- | --- |
| `pharmaos_current_tenant_id()` | Casts `current_setting('app.current_tenant_id', true)` to UUID; `NULL` when unset. |
| `pharmaos_is_privileged()` | Boolean session flag; defaults to `false`. |
| `pharmaos_set_updated_at()` / `pharmaos_touch_updated_at()` | Trigger support. |
| `pharmaos_active_distributor_link(company, distributor)` | True only for an `ACTIVE`, effective-dated link. |
| `pharmaos_can_view_distribution_row(owner, distributor)` / `pharmaos_can_write_distribution_row(owner, distributor)` | Link-aware order visibility and owner-only write checks. |
| `pharmaos_can_view_distributor_data(distributor)` | Allows distributor owners and actively linked companies to read inventory/movements. |

Every tenant-owned table must enable RLS (preferably `FORCE RLS`) and define policies for `SELECT`, `INSERT`, `UPDATE`, `DELETE`:

```sql
-- Planned pattern; apply per table and per operation
ALTER TABLE crm_doctors ENABLE ROW LEVEL SECURITY;
ALTER TABLE crm_doctors FORCE ROW LEVEL SECURITY;

CREATE POLICY crm_doctors_tenant ON crm_doctors
  FOR ALL
  USING (tenant_id = pharmaos_current_tenant_id()
         OR pharmaos_is_privileged())
  WITH CHECK (tenant_id = pharmaos_current_tenant_id()
              OR pharmaos_is_privileged());
```

Rules:

- `USING` controls visible/affected rows; `WITH CHECK` prevents inserting or moving a row into another tenant.
- Use separate policies where different operations need different conditions (for example read-only audit tables).
- Do not expose a privileged setting path to client code. Privileged operations require an audited break-glass workflow and dual approval.
- Background jobs must set tenant context explicitly per job, never run with a global bypass.

## 4. Cross-tenant distribution access

Allowed visibility is driven only by active links:

```text
PHARMA_COMPANY A ─┐
                  ├── company_distributor_links ── DISTRIBUTOR D
PHARMA_COMPANY B ─┘
```

| Actor | May see | Condition |
| --- | --- | --- |
| Company A user | Orders it placed on D, D's shared inventory summary, secondary sales tied to those orders | Active link `(A, D)` and company-side permission |
| Distributor D user | Orders from A and B, own inventory, own secondary sales | Active link `(A, D)` or `(B, D)` and distributor-side permission |
| Company A user | Company B users, doctors, targets, tour plans | Never |
| Distributor D user | Another distributor's data | Never |

The link table has policies whose checks reference both tenant columns, plus a unique constraint on `(tenant_id, distributor_tenant_id)`, where `tenant_id` is the company tenant. Link creation/revocation requires `link.manage` and emits audit events. When a link is suspended or ended, cross-tenant queries stop returning data immediately through service predicates and RLS helpers; cached client data is removed on the next sync pull.

## 5. Application-layer enforcement checklist

For every new tenant-owned resource:

- [x] Table has `tenant_id uuid NOT NULL` and a composite index starting with `tenant_id`.
- [x] Repository query includes `tenant_id = auth.MustPrincipal(ctx).TenantID` even though RLS exists.
- [x] Lookups by UUID include tenant in the `WHERE` clause; a foreign UUID returns `apperrors.ErrTenantMismatch` (HTTP 404).
- [x] Create payloads never accept `tenant_id`; the value is injected server-side.
- [x] RLS `USING`/`WITH CHECK` policies exist for all operations and are covered by tests.
- [x] `TenantTransaction` wraps the handler so the session settings and writes share one transaction.
- [x] Cross-tenant reads go through an explicit link-aware service and emit an audit event.
- [x] Data scope (SELF/TEAM/TERRITORY/REGION/DIVISION/TENANT) is applied after tenant filtering. See [permissions.md](permissions.md).

## 6. Mobile and offline isolation

- Local SQLite data is scoped per tenant; on tenant switch or logout the app clears tenant data and the operation queue (implemented in Phase 6).
- Every outbox row stores `tenant_id` and `user_id`; the sync service rejects operations whose tenant differs from the current token.
- Refresh tokens are bound to a session and tenant; rotating credentials cannot widen tenant access.
- Cached distribution data carries link status; stale cross-tenant records are removed when a link ends.

## 7. Required isolation tests (Phase 2 exit criteria)

| Scenario | Expected result |
| --- | --- |
| User A requests resource UUID owned by tenant B | 404 `TENANT_MISMATCH`; no row leaked through errors or timing of list endpoints |
| Raw SQL without `app.current_tenant_id` while RLS enabled | Zero rows / policy violation, never cross-tenant rows |
| Create with body `tenant_id` = B while session is A | Rejected (400) or stored under A; never B |
| Company A reads distributor D with no active link | 403/404 |
| Company A reads distributor D with active link | Allowed fields only, audited |
| Link suspended between token issue and request | Access denied immediately |
| Offline operation queued under A replayed after switching to B | Rejected; queue entry dropped and alerted |
| Privileged flag set on normal token | Rejected; privilege only from platform-controlled path |
| Concurrent requests on one pooled connection | No tenant-setting bleed (transaction-local `set_config`) |

## 8. Operational guidance

- Run isolation tests in CI against a real PostgreSQL instance (`make test-db` pattern), not mocks.
- Review every new policy with `EXPLAIN` to ensure the tenant predicate uses indexes.
- Monitor audit events for denied cross-tenant attempts and failed privileged actions.
- Include tenant-scoped restore verification in disaster recovery drills.

Related documents: [database-design.md](database-design.md), [permissions.md](permissions.md), [api-design.md](api-design.md).
