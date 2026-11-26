# PharmaOS Architecture

> **Status:** Phases 1–10 are complete. Components marked **Planned** describe the follow-on backlog. The current tree contains `backend/`, `mobile/`, `admin-web/` and `infrastructure/`.

## 1. Purpose and shape

PharmaOS is a production-oriented, multi-tenant pharmaceutical platform combining:

- **Sales Force Automation (SFA):** CRM, tour plans, field visits/DCR, products, targets, reporting.
- **Distribution Connect:** distributors, company–distributor links, orders, secondary sales, inventory.

The backend is a **modular monolith** in Go, with PostgreSQL as the system of record. Mobile is offline-first; the admin web app is a thin, authenticated console. Tenant context is always derived from the authenticated session, never from client input.

```text
  Expo / RN field app                Vite / React admin console
        |  HTTPS + JWT                    |  HTTPS + JWT
        +----- nginx (edge: /api/, /admin/, probes, docs) ---+
                              |
                      Gin API (Go 1.25)
        middleware -> authn -> tenant transaction -> module handler
                              |
             GORM / pgx pool  ->  PostgreSQL 16 (RLS + SQL migrations)
                              |
                        Redis 7 (provisioned; integration planned)
```

## 2. Implemented components

| Area | Implementation | Notes |
| --- | --- | --- |
| API process | `backend/cmd/api/main.go` | Composition root: config, logger, migrations, DB, HTTP server, graceful shutdown on `SIGINT`/`SIGTERM`. |
| Configuration | `backend/internal/config/config.go` | Environment-driven; validates JWT secret, bcrypt cost, CORS, webhook and rate-limit hardening in production. |
| HTTP framework | Gin; `backend/internal/server/server.go` | `gin.New()` with an explicit middleware chain; release mode in production. |
| Database | `backend/internal/database/database.go` | GORM + `gorm.io/driver/postgres` (pgx v5); pool 40 open / 10 idle, 2h lifetime, 15m idle lifetime. |
| Migrations | `backend/internal/database/migrations.go` | Versioned SQL, PostgreSQL advisory lock, one transaction per migration, `schema_migrations` ledger. |
| Foundation schema | `backend/migrations/000001__foundation.up.sql` | `pgcrypto`, `citext`, timestamp trigger and tenant/privilege helper functions. |
| Health | `backend/internal/health/handler.go` | `/healthz` liveness; `/readyz` checks DB ping and migration version. |
| API docs | `backend/internal/platform/apidocs/` | Embedded OpenAPI 3.0.3 spec served at `/docs/openapi.yaml`, landing page at `/docs`. |
| Middleware | `backend/internal/middleware/` | Request ID, structured access log, recovery, security headers, CORS allowlist, client hints, in-process and PostgreSQL-backed rate limits. |
| Observability | `backend/internal/metrics/`, `GET /metrics` | Prometheus request/latency/security counters with in-process collection. |
| Admin console | `admin-web/` | Vite/React/TypeScript SPA with a typed envelope client, refresh-aware bearer sessions and permission-aware navigation. |
| Cross-cutting | `backend/internal/platform/` | `response.Envelope`, `apperrors.AppError`, zap logger factory. |
| Infrastructure | `infrastructure/docker-compose.yml`, `nginx/nginx.conf`, `backend/Dockerfile` | Postgres 16, Redis 7, API, optional nginx edge; multi-stage non-root image. |

## 3. Request lifecycle

Current chain (`server.New`): `Recovery → RequestID → Logging → SecurityHeaders → CORS → ClientHints → Metrics → RateLimit → handler`.

Chain for business routes:

1. **Edge:** nginx forwards `/api/`, `/healthz`, `/readyz`, `/docs` to the API, serves the admin SPA at `/admin/`, and sets `X-Forwarded-*` headers; `/metrics` should be restricted to the monitoring network.
2. **Transport middleware:** request ID (honors inbound `X-Request-ID`), access log, panic recovery, CORS, selected rate limiter, metrics collection.
3. **Authentication:** validate bearer JWT, load session, build `auth.Principal` on the request context.
4. **Authorization:** require a `resource:action` permission; resolve data scope to SQL predicates.
5. **TenantTransaction:** begins a GORM transaction, sets `app.current_tenant_id` and `app.is_privileged` locally via `set_config(..., true)`, exposes the transaction through `DBFromContext`, rolls back on status ≥ 400 or GORM error, otherwise commits.
6. **Handler/service/repository:** use the transaction-scoped handle; never accept `tenant_id` from the request body.
7. **Response:** all JSON uses `response.Envelope` (`success`, `message`, `data`, `meta`, `errors`).

`TenantTransaction` is mounted on authenticated business route groups and used by write services.

## 4. Backend module boundaries

```text
backend/
  cmd/api/                  process bootstrap only
  internal/
    config/                 env loading and validation
    database/               connection pool + migration runner
    server/                 router composition
    health/                 probe handlers
    auth/                   JWT, sessions, Principal and context helpers
    middleware/             HTTP, RBAC, tenant transaction and rate-limit middleware
    metrics/                Prometheus-format collector and HTTP handler
    platform/               errors, response, logging, API docs
    tenants/, users/, roles/, permissions/, divisions/, territories/
    organization/, employees/, crm/, masterdata/, tours/, visits/, dcr/
    distribution/, reporting/, audit/, integrations/, syncapi/
  migrations/               source of truth for schema
```

Modules follow `handler → service → repository` with constructor injection from `server.New`. Services own permission and workflow decisions; repositories own tenant and data-scope predicates. Cross-module calls go through narrow interfaces, not shared mutable state.

## 5. Cross-cutting decisions

| Concern | Decision | Rationale |
| --- | --- | --- |
| Schema changes | SQL migrations only; GORM `AutoMigrate` is prohibited. | Reviewable, reversible, consistent across environments. |
| Identity/keys | UUID primary keys; client-generated UUIDs for offline records. | Safe merge across devices and tenants. |
| Isolation | Application tenant filter + PostgreSQL RLS helper functions. | Defense in depth if an application query misses a predicate. |
| Transactions | `TenantTransaction` for write paths; GORM default transaction remains enabled. | Multi-step operations are atomic; GORM's own per-save transaction is retained. |
| Errors | `apperrors.AppError` with stable codes; `ErrTenantMismatch` maps to 404. | Consistent clients and no cross-tenant existence disclosure. |
| Logs | zap: development console, production JSON; every log carries `request_id` where available. | Correlatable operations without logging credentials or patient data. |
| Rate limiting | In-memory fallback or shared PostgreSQL counters keyed by client IP (`RATE_LIMIT_BACKEND`, `RATE_LIMIT_PER_MIN`). | Local simplicity with safe multi-replica production default. |
| Metrics | In-process Prometheus collector exposed at `/metrics`. | No extra dependency; edge restricts scrape access. |
| Migration integrity | Applied migration checksums are verified on startup and rollback. | Detects edited migrations before drift is introduced. |
| Cache/queue | Redis is in Compose and `REDIS_URL` is configured; no client is wired yet. | Reserved for refresh-token revocation, sync coordination and jobs. |
| Patient data | No patient/PII prescription records are stored by design. | Reduced compliance surface; doctors and outlets are business entities. |

## 6. Planned module map

| Domain | Modules | Phase |
| --- | --- | --- |
| Identity & access | Tenants, users, roles, permissions, sessions | 2 |
| Organization | Hierarchy (MR → TBM → ASM → RSM → NSM), divisions, territories | 3 |
| Healthcare CRM | Doctors, chemists, stockists, products, targets | 4 |
| Field execution | Tour plans, visits, DCR, product detailing | 5 |
| Field client | Expo app, local SQLite, operation queue, sync | 6 |
| Distribution | Company–distributor links, orders, secondary sales, inventory | 7 |
| Insight & ops | Reporting, dashboards, append-only audit, integrations | 8 |
| Hardening | Security review, load tests, deployment validation | 9 |

## 7. Invariants to preserve

- No endpoint may trust a tenant identifier from a body, query string, or device-local state.
- Every tenant-owned table has `tenant_id uuid NOT NULL`, a composite index including `tenant_id`, and RLS policies.
- All mutating HTTP operations are transaction-scoped and auditable.
- Offline writes carry a UUID and an idempotency key; replaying a batch is safe.
- API contract changes update `backend/internal/platform/apidocs/openapi.yaml` in the same change.
- No real patient data or prescription-level patient information is stored.

Related documents: [database-design.md](database-design.md), [api-design.md](api-design.md), [tenant-isolation.md](tenant-isolation.md), [deployment.md](deployment.md).
