# Implementation Roadmap

> **Status:** Phases 1–10 are **complete**. This document records delivered scope and the remaining post-release backlog; see `PROJECT_STATUS.md` for verification status.

## 1. Phase summary

| Phase | Theme | Status | Primary deliverables |
| --- | --- | --- | --- |
| 1 | Foundation | **Complete** | Go module, Gin server, config, logging/middleware, GORM pool, migration runner + foundation migration, probes, OpenAPI, Docker Compose, unit tests. |
| 2 | Tenants, auth and access control | **Complete** | Tenant tables, JWT login/refresh/logout, users, roles, permissions, RBAC + data-scope guards, isolation tests. |
| 3 | Organization hierarchy | **Complete** | Divisions, territories, employees, MR → TBM → ASM → RSM → NSM hierarchy, scope expansion from hierarchy. |
| 4 | Healthcare CRM and catalog | **Complete** | Doctors, chemists, stockists, products, targets, CRM APIs with tenant filtering and RLS. |
| 5 | Field execution | **Complete** | Tour plans, visits, DCR, product detailing, GPS check-in, photo attachments, workflow states. |
| 6 | Mobile offline app | **Complete** | Expo/React Native app, SQLite read models, outbox, idempotency keys, sync endpoints, conflict handling. |
| 7 | Distribution Connect | **Complete** | Distributors, many-to-many company–distributor links, orders, secondary sales, inventory movements, mobile sync integration. |
| 8 | Insight and operations | **Complete** | Reporting, dashboards, append-only audit log, integrations/webhooks, background workers. |
| 9 | Hardening and release | **Complete** | Shared PostgreSQL rate limiting, production config validation, Prometheus metrics, migration checksum verification, regression tests, operational scripts and final deployment docs. |
| 10 | Admin web console | **Complete** | Vite/React/TypeScript SPA, typed API client, authenticated session handling, permission-aware navigation, administrative screens and nginx delivery. |

## 2. Phase details

### Phase 1 — Foundation (complete)

Delivered: Go 1.25 modular monolith with graceful shutdown; env-driven config with production secret validation; zap logging, request IDs, access logs, recovery, CORS, rate limit and security headers; PostgreSQL pool; advisory-locked transactional SQL migration runner; `000001__foundation` with `pgcrypto`, `citext`, updated-at and tenant/privilege helpers; `/healthz` and `/readyz` including migration-state check; embedded OpenAPI 3 document; Docker Compose and non-root backend image; configuration, migration-loader and HTTP tests. See [architecture.md](architecture.md) and [development-setup.md](development-setup.md).

### Phase 2 — Tenants, authentication and access control

- `tenants` with `PHARMA_COMPANY` / `DISTRIBUTOR`, `users`, roles/permissions tables, sessions and refresh tokens.
- JWT access/refresh flow; `Principal` populated from claims; session revocation via Redis; `token_version` for forced logout.
- Mount auth middleware + `TenantTransaction` on `/api/v1`; route-level permission guards; RLS policies on all Phase 2 tables.
- **Exit:** cross-tenant access tests pass (see [tenant-isolation.md](tenant-isolation.md)); permission matrix tests pass; `/readyz` still gates on migrations.

### Phase 3 — Organization hierarchy

- Divisions, territories, employees and `employee_hierarchy`; assign users to territories/divisions.
- Resolve `TEAM`, `TERRITORY`, `REGION`, `DIVISION` scopes from hierarchy; extend principal with region IDs/team descendants.
- **Exit:** MR/TBM/ASM/RSM/NSM reporting chain enforced in queries; users see only in-scope records.

### Phase 4 — Healthcare CRM and catalog

- Doctors, chemists, stockists, products and targets with UUID keys, `tenant_id`, indexes, constraints and RLS.
- CRUD/list/search APIs with permission + scope; no patient-level records.
- **Exit:** tenant and scope isolation tests for all new tables; OpenAPI and [database-design.md](database-design.md) updated.

### Phase 5 — Field execution (complete)

Delivered: tour plans and lines, visits with GPS check-in, DCR and product detailing, visit attachment metadata, workflow states, scoped listing, idempotent visit creation, and audit events on state transitions. Field records are scoped through `SELF`, `TEAM`, `DIVISION`, `TERRITORY`, and `TENANT` rules.

- **Exit:** visit/DCR lifecycle exposed through the authenticated API and documented in OpenAPI; offline submission contract is covered by Phase 6 sync work.

### Phase 6 — Mobile offline application (complete)

Delivered: Expo/React Native app under `mobile/`, secure token storage via `expo-secure-store`, tenant-scoped SQLite read models, durable outbox with retry/backoff and dead-letter state, sync cursors, and a login/Home/visit capture flow. The backend exposes `GET /sync/changes`, `POST /sync/operations`, and `GET /sync/status` with per-operation idempotent replay and conflict results.

- **Exit:** offline visit capture and replay-safe push are wired end to end; encrypted SQLite containers, device tests and full soak testing remain post-release work.

### Phase 7 — Distribution Connect (complete)

Delivered: `dist_company_distributor_links` with status/effective dates, distributor profiles, idempotent orders with server-priced lines, order workflow, secondary sales, inventory projections, and serialized movement processing. Link-aware RLS policies and service predicates grant company/distributor visibility only while an `ACTIVE` effective-dated link exists. Order approval reserves stock in a transaction with row locks and version checks; each order line has a deterministic movement idempotency key. Mobile sync pulls distribution read models and pushes order, secondary-sale, and inventory-movement operations.

- **Exit:** link-aware API and mobile sync are implemented; integration coverage asserts cross-tenant visibility, link suspension revocation, order replay, and reserved-stock deltas. Docker was unavailable during the implementation pass, so the DB-backed test must be rerun when PostgreSQL test services are available.

### Phase 8 — Insight, audit and integrations (complete)

Delivered: `000007__insights_integrations` with RLS-protected report definitions, report snapshots, integration connections, webhook deliveries, role grants, and a database trigger that rejects updates/deletes on `audit_logs`. Field activity and distribution dashboards respect tenant and data scope; distribution metrics use active company-distributor links. The immutable audit trail is queryable with date/module/action/entity filters. Integration APIs create idempotent webhook deliveries, expose delivery history, and retry failed/dead deliveries. The dispatcher worker coordinates through a PostgreSQL advisory lock, applies exponential backoff, dead-letters exhausted deliveries, and signs payloads with an HMAC secret referenced by environment variable name.

- **Exit:** reports respect tenant + data scope; audit entries are immutable; webhook retries are idempotent.
- **Deferred beyond Phase 9:** scheduled exports, shared Redis-backed job coordination, device tests, and load-tested worker scaling.

### Phase 9 — Hardening and release (complete)

Delivered: `000008__hardening` adds a shared PostgreSQL rate-limit counter and a security-definer function used by `middleware.DistributedRateLimit`; in-process limiting remains the local/fallback backend. Production config validation requires a 32-byte JWT secret, explicit CORS allowlist, HTTPS webhooks, and PostgreSQL-backed rate limiting. `GET /metrics` exposes Prometheus counters for routes, status classes, latency, and security outcomes. Migration checksums are now verified before subsequent applies and rollbacks. Regression tests cover RBAC denial, config/trusted-proxy validation, metrics exposure and the rate-limit function. Operational scripts cover load smoke tests, backup/restore drills, dependency verification and inline secret scanning.

- **Exit:** release validation and deployment documentation are complete; live production sign-off remains an operational approval step.

### Phase 10 — Admin web console (complete)

Delivered: `admin-web/` Vite/React/TypeScript SPA using the same `response.Envelope` contract as mobile, with bearer-token storage, refresh-on-401, logout and permission-aware navigation. Screens cover dashboard, tenants, users, roles/permissions, organization hierarchy, doctors/chemists/stockists/products/targets, distribution links/orders/inventory/secondary sales, reports/audit/definitions/snapshots, integration connections/deliveries and sync health. Distribution writes send idempotency keys. The production build ships in an nginx image with SPA fallback and a same-origin `/api/` proxy; the edge profile exposes it at `/admin/`.

- **Exit:** `npm run typecheck` and `npm run build` pass; UI checks are presentation-only because the API remains the authorization boundary.

## 3. Dependency chain

```text
1 ──> 2 ──> 3 ──> 4 ──> 5 ──> 6 ──> 7 ──> 8 ──> 9 ──> 10
       |            |            |          |             |
       +-- auth  +-- scope  +-- sync-ready  +-- distribution +-- stable API contracts for admin UI
```

Phase 2 unblocks all business modules; Phase 3 is required for correct roll-up scopes; Phase 6 depends on stable visit/order contracts from Phases 5 and 7; Phase 8 depends on the complete domain data; Phase 9 validates the whole system; Phase 10 consumes the stable API surface.

## 4. Requirement traceability

| Product requirement | Phase(s) |
| --- | --- |
| `PHARMA_COMPANY` and `DISTRIBUTOR` tenants | 2, 7 |
| `company_distributor_links` many-to-many | 7 |
| MR → TBM → ASM → RSM → NSM hierarchy | 3 |
| CRM, tour plans, visits/DCR, products, targets | 4, 5 |
| Orders, secondary sales, inventory | 7 |
| Reporting, audit, integrations | 8 |
| JWT access/refresh + RBAC | 2 |
| `SELF`/`TEAM`/`TERRITORY`/`REGION`/`DIVISION`/`TENANT` scopes | 2, 3 |
| Versioned SQL migrations, UUID keys, `tenant_id`, RLS helpers | 1, 2, 4–8 |
| Offline-first SQLite, UUIDs, idempotency keys, operation queue, sync | 5, 6, 7 |
| Administrative console for tenants, access, masters, distribution and reports | 10 |
| Security review, performance tests, deployment validation | 9 |

## 5. Definition of done (each phase)

- Schema changes are versioned SQL with working `down` scripts and RLS where tenant data exists.
- Routes enforce authentication, permission and data scope; tenant comes only from the session.
- OpenAPI and affected documents in `docs/` are updated with the code.
- Unit tests and tenant-isolation integration tests pass (`make test-all`, `make test-db`).
- Sensitive operations are audited; no patient-level data is stored.
- Deployment notes and configuration changes are reflected in [deployment.md](deployment.md).
