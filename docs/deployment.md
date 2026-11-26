# Deployment

> **Status:** All phases through Phase 10 are implemented, including shared PostgreSQL rate limiting, migration checksum verification, Prometheus metrics, production config validation, operational scripts and the admin web console. Redis-backed coordination remains follow-on work.

## 1. Current topology

| Service | Image | Exposure | Role |
| --- | --- | --- | --- |
| `postgres` | `postgres:16-alpine` | `55432:5432`, named volume `postgres_data` | System of record; health checked with `pg_isready`. |
| `postgres_test` | `postgres:16-alpine`, profile `test` | `55433:5432`, tmpfs | Disposable database for migration tests. |
| `redis` | `redis:7-alpine` | `6379:6379`, named volume `redis_data` | Provisioned with AOF; not yet used by application code. |
| `api` | built from `backend/Dockerfile` | `8080:8080` | Go/Gin API; waits for healthy Postgres and Redis; runs migrations on startup. |
| `admin-web` | built from `admin-web/Dockerfile` | `8081:80` | Vite/React SPA served by nginx with SPA fallback and `/api/` proxy. |
| `nginx` | `nginx:1.27-alpine`, profile `edge` | `80:80` | Reverse proxy for `/api/`, `/admin/`, `/healthz`, `/readyz`, `/docs`, `/metrics`; 25 MB body limit. |

The Dockerfile is a two-stage build: `golang:1.25-alpine` builds a static `CGO_ENABLED=0` binary; the runtime is `alpine:3.21` with CA certificates and timezone data, runs as UID 10001, and copies `migrations/` and `docs/`. The API serves graceful shutdown with `SHUTDOWN_TIMEOUT` (default 15s) and HTTP timeouts of 10s read-header, 60s idle and 2m write.

## 2. Local startup

```bash
cp .env.example .env
make up                         # postgres + redis
cd backend && go run ./cmd/api  # applies migrations, starts :8080
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

Admin web with Compose:

```bash
docker compose -f infrastructure/docker-compose.yml up -d
curl http://localhost:8081
```

Optional edge proxy (API plus admin console at `/admin/`):

```bash
docker compose -f infrastructure/docker-compose.yml --profile edge up -d
curl http://localhost/admin
```

`make down` stops the stack. `make dev` combines `make up` with `go run ./cmd/api`. The admin build accepts `VITE_API_BASE_URL` as a build argument; an empty value keeps API calls same-origin behind nginx.

## 3. Configuration reference

Loaded by `backend/internal/config/config.go`; see `.env.example` for defaults.

| Variable | Purpose | Production guidance |
| --- | --- | --- |
| `APP_ENV` | `development` / `production`; controls log format, Gin mode and secret check | Set to `production`. |
| `HTTP_ADDR` | Listen address | `:8080` behind proxy. |
| `DATABASE_URL` | PostgreSQL DSN | Managed PostgreSQL with TLS; never the Compose credential. |
| `TEST_DATABASE_URL` | DSN for migration tests | Disposable database only. |
| `REDIS_URL` | Redis DSN | Managed Redis with TLS/auth once integrated. |
| `JWT_SECRET` | Token signing secret | Required in production; inject from a secrets manager, ≥ 32 random bytes. |
| `ACCESS_TOKEN_TTL` / `REFRESH_TOKEN_TTL` | Default `15m` / `720h` | Keep access tokens short; rotate refresh tokens. |
| `PASSWORD_HASH_COST` | Default 12; validated 10–15 | Benchmark against hardware; 12+ recommended. |
| `CORS_ORIGINS` | Comma-separated exact origins | Admin web and mobile origins only; no wildcard with credentials. |
| `RATE_LIMIT_PER_MIN` | Default 300; requests per client IP per minute | Size to expected client burst; returns `429` with `Retry-After: 60`. |
| `RATE_LIMIT_BACKEND` | `memory` or `postgres`; production defaults to `postgres` | Keep `postgres` for multi-replica deployments; shared counters use migration `000008`. |
| `TRUSTED_PROXIES` | Comma-separated IPs/CIDRs trusted to set `X-Forwarded-For` | Restrict to the load balancer/ingress CIDR; leave empty without a proxy. |
| `MIGRATIONS_DIR` | Default `migrations`; `/app/migrations` in image | Ship migrations with every image. |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` or `warn`. |
| `SHUTDOWN_TIMEOUT` | Default `15s` | Keep below orchestrator termination grace period. |
| `WEBHOOK_WORKER_INTERVAL` / `WEBHOOK_MAX_ATTEMPTS` / `WEBHOOK_TIMEOUT` | Dispatcher tuning | Keep one active worker or rely on advisory-lock coordination. |
| `ALLOW_INSECURE_WEBHOOKS` | Allows private/loopback webhook URLs in non-production | Must remain false in production. |

## 4. Release sequence

1. Build and scan the immutable API image; verify it is non-root and contains migrations.
2. Apply migrations with a one-off job using the **same image** (`/app/api` startup path applies them). Do not use `make migrate`: it currently rolls back all migrations.
3. Wait for `/readyz` to report `migration_version` ≥ expected from the image.
4. Roll the API Deployment with a rolling strategy; readiness gates traffic, graceful shutdown drains in-flight requests.
5. Run smoke checks (`/healthz`, `/readyz`, `/metrics`, authenticated login).
6. Run `make deps-audit` and `make scan-secrets`; run `make load-smoke` against staging and record status counts.
7. Monitor `/metrics` for 401/403/429 spikes, `5xx`, p95 latency and request ID-correlated logs for one deploy window.
8. Perform `make backup-drill` against a disposable database and confirm the migration count before rollout.

Migrations must be additive and backward compatible so old and new API replicas can coexist. Destructive changes require two releases: expand (add/backfill), then contract after all clients are upgraded. `-migrate-down N` is for local recovery only; production rollback is code rollback, not schema rollback.

## 5. Production checklist

- [ ] TLS terminated at a trusted edge; HSTS configured there; internal DB/Redis traffic encrypted.
- [ ] Secrets in a managed store; no secrets in Compose files, images or repository.
- [ ] Managed PostgreSQL with automated backups, PITR and tested tenant-scoped restore.
- [ ] RLS enabled and tested on every tenant-owned table before exposing routes.
- [ ] `RATE_LIMIT_BACKEND=postgres` and migration `000008` applied; in-memory mode only for local use.
- [ ] `TRUSTED_PROXIES` limited to ingress CIDRs before relying on client IP rate limits.
- [ ] `/docs` and `/docs/openapi.yaml` restricted or intentionally exposed; `/metrics` restricted to the monitoring network.
- [ ] Centralized JSON logs with request IDs; no tokens, secrets or patient data in logs.
- [ ] Resource requests/limits, liveness on `/healthz`, readiness on `/readyz`, scraping on `/metrics`.
- [ ] CORS allowlist matches deployed origins and excludes wildcard.
- [ ] `APP_ENV=production`, `JWT_SECRET` ≥ 32 bytes, `ALLOW_INSECURE_WEBHOOKS=false`, development fallback never used.
- [ ] Migration checksum verification enabled; no local edits to applied migrations.
- [ ] Mobile and admin web builds distributed through their platform stores/hosting with pinned API base URLs.

## 6. Scaling and availability

- Start with 2+ API replicas; rate-limit counters are shared in PostgreSQL when `RATE_LIMIT_BACKEND=postgres`.
- The GORM pool is 40 open / 10 idle **per replica**; size Postgres `max_connections` accordingly or introduce a pooler (PgBouncer in transaction mode) later.
- Redis will hold refresh-token revocation, sync coordination and background-job state; enable persistence and failover for production.
- Webhook delivery runs in a database-coordinated dispatcher worker; Redis coordination and deeper worker scaling tests remain follow-on work.
- Add read replicas only after query analysis; tenant predicates must remain index-friendly on replicas.

## 7. Environments

| Environment | Database | Secrets | Migrations | Notes |
| --- | --- | --- | --- | --- |
| Local | Compose `postgres` on 55432 | `.env` copy of `.env.example` | Applied on `go run` | Developer machine; admin web on 8081. |
| Test | Compose `postgres_test` on 55433, tmpfs | Test-only JWT secret | `make test-db` | Disposable; skipped without `TEST_DATABASE_URL`. |
| Staging | Managed PG, sanitized data | Secrets manager | Same image and release sequence as prod | Full RLS/isolation suite runs here. |
| Production | Managed PG with PITR | Secrets manager | Additive only, one-off job | No privileged break-glass without audit and approval. |

## 8. Observability and alerts

`/healthz` reports process uptime only; `/readyz` fails when the DB is unreachable or the applied migration version lags the image. Access logs include method, route, status, latency, request ID and client IP. `/metrics` exposes `pharmaos_http_requests_total`, `pharmaos_http_request_duration_seconds` and `pharmaos_security_events_total` in Prometheus text format. Alert on readiness failures, 401/403 spikes, 429 rate, `5xx`, p95 latency, sync queue depth and migration drift; restrict scraping at the edge or over the internal network.

Related documents: [development-setup.md](development-setup.md), [database-design.md](database-design.md), [implementation-roadmap.md](implementation-roadmap.md).
