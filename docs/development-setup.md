# Development Setup

> **Status:** Backend, Expo mobile and admin web setup reflect the repository through Phase 10.

## 1. Prerequisites

| Tool | Version | Used for |
| --- | --- | --- |
| Go | 1.25+ (module declares `go 1.25.0`) | API, tests, migrations. |
| Docker + Docker Compose | Current stable | PostgreSQL 16, Redis 7, optional nginx; PostgreSQL test service. |
| `curl` | Any | Probe and docs smoke checks. |
| `psql` | Optional | Manual SQL inspection. |
| Node.js + npm | 20+ | Expo mobile app and Vite admin console. |

## 2. First run

```bash
# From the repository root
cp .env.example .env
make up
cd backend && cp ../.env .env 2>/dev/null || true   # optional local convenience
cd backend && go run ./cmd/api
```

In another terminal:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/docs
curl http://localhost:8080/docs/openapi.yaml
```

`go run ./cmd/api` loads environment variables from the process environment (`.env` is a template, not auto-loaded), applies all pending migrations, starts the HTTP server and shuts down cleanly on Ctrl+C. `make dev` performs `make up` then the same `go run`.

## 3. Verification

```bash
make fmt                 # gofmt -w ./cmd ./internal ./tests (from backend/)
make lint                # go vet ./...
make test                # go test ./...
make test-db             # starts postgres_test on :55433 and runs migration/integration tests
make test-all            # fmt + lint + test
make mobile-typecheck    # TypeScript check for mobile/
make admin-web-typecheck # TypeScript check for admin-web/
make admin-web-build     # full admin-web production build
make admin-web-audit     # fail on moderate+ npm advisories in admin-web
make deps-audit          # go mod verify + dependency inventory
make scan-secrets        # fail on inline secret assignments
make load-smoke          # configurable HTTP smoke/load loop
make verify-release      # fmt + lint + test + mobile/admin typechecks + admin audit + deps + secrets
```

`TestMigrateUpAgainstTestDatabase` skips automatically when `TEST_DATABASE_URL` is unset. The test database uses tmpfs, so it is discarded when the container stops. `make backup-drill` requires `DATABASE_URL`, `RESTORE_DATABASE_URL` and `pg_dump`/`pg_restore`/`psql`; it is destructive only to the restore database.

Admin web development:

```bash
cd admin-web
cp .env.example .env
npm install
npm run dev
```

Set `VITE_API_BASE_URL=http://localhost:8080` for a standalone Vite server. Leave it empty when the SPA is served by nginx so `/api/v1/*` calls stay same-origin. With Compose, `http://localhost:8081` serves the admin container directly and the `edge` profile exposes it at `http://localhost/admin`.

## 4. Repository layout

```text
pharmaos/
  backend/
    cmd/api/main.go                  bootstrap, flags, graceful shutdown
    internal/
      config/                        env configuration + tests
      database/                      GORM pool, migration runner + tests
      server/                        router composition + httptest coverage
      health/                        liveness/readiness
      auth/                          Principal/context helpers
      middleware/                    HTTP middleware, RBAC, transactions, rate limits
      metrics/                       Prometheus collector + /metrics handler
      platform/                      apidocs, errors, response, logging
      auth/, users/, tenants/        identity and tenancy modules
      crm/, masterdata/, tours/      CRM, catalog, and field-execution modules
      visits/, dcr/, distribution/   field and distribution modules
      reporting/, audit/             dashboards and immutable audit trail
      integrations/                  webhook connections, queue and worker
      syncapi/                       mobile sync API
    migrations/                      versioned up/down SQL (source of truth)
  infrastructure/                    docker-compose.yml, edge nginx config
  scripts/                           load, backup-drill and secret-scan helpers
  docs/                              this documentation set
  mobile/                            Expo / React Native + SQLite field app
  admin-web/                         Vite / React / TypeScript admin console
  .env.example, Makefile
```

## 5. Backend conventions

- **Composition root:** add dependencies to `server.New`; use constructor injection rather than globals.
- **Handlers:** register under a versioned group (`/api/v1`) once authentication exists; use `response.OK/Created/Fail`.
- **Errors:** define stable `apperrors.AppError` values; never leak internal errors to clients.
- **Database work:** obtain the transaction via `middleware.DBFromContext(c)`; never bypass it for tenant writes.
- **Tenant data:** take `tenant_id` from `auth.MustPrincipal(ctx)`; reject client-supplied tenant values.
- **Schema:** add a numbered up/down SQL pair; never use GORM `AutoMigrate`; update RLS policies and indexes in the same migration.
- **Tests:** `testify/require`, table-driven where useful; integration tests need `TEST_DATABASE_URL`.
- **Docs:** update `backend/internal/platform/apidocs/openapi.yaml` with every new endpoint; keep these `docs/` files aligned with the actual phase state.

## 6. Adding a feature (checklist)

1. Add or extend SQL migrations with UUID keys, `tenant_id`, indexes, constraints and RLS policies.
2. Implement `handler → service → repository` in the relevant `internal/modules/<domain>` package.
3. Wire dependencies and routes in `server.New`; add auth permission and data-scope enforcement.
4. Define permission codes and assign them to roles; add audit events for sensitive writes.
5. Document the endpoint in OpenAPI and the relevant doc in this directory.
6. Add unit tests plus tenant-isolation integration tests; run `make test-all` and `make test-db`.
7. For mobile-facing writes, add idempotency handling and an outbox/sync contract entry.

## 7. Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `JWT_SECRET must be set in production` | `APP_ENV=production` without a secret; set a ≥ 32-byte value from a secret store. |
| `/readyz` returns 503 `Migrations pending` | Image expects a newer migration version; run the API/migration job and re-check. |
| `TestMigrateUpAgainstTestDatabase` skipped | `TEST_DATABASE_URL` unset; use `make test-db`. |
| Port 55432/55433/6379/8080/8081 already in use | Stop conflicting local services or remap ports in `infrastructure/docker-compose.yml`. |
| CORS preflight returns no allow headers | Origin is not in `CORS_ORIGINS`; exact scheme/host/port required. |
| HTTP 429 with `Retry-After: 60` | Per-IP rate limit exceeded; lower client burst or raise `RATE_LIMIT_PER_MIN`. |
| `migration N was modified after it was applied` | An applied `up` migration changed; restore the original file or create a new migration version. |
| `migrate` unexpectedly removes tables | `make migrate` runs `-migrate-down 0`; use a fresh `go run ./cmd/api` to apply instead. |
| Missing route under `/api/v1` | Check the OpenAPI contract; modules through Phase 9 are mounted. |
| `Rate limiter unavailable` | `RATE_LIMIT_BACKEND=postgres` requires migration `000008` and a reachable database. |

Related documents: [architecture.md](architecture.md), [deployment.md](deployment.md), [implementation-roadmap.md](implementation-roadmap.md).
