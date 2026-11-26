# PharmaOS

Enterprise multi-tenant pharmaceutical Sales Force Automation (SFA) and Distribution Connect platform.

## Status

Implementation is milestone-driven. See [PROJECT_STATUS.md](PROJECT_STATUS.md) for the current phase and checklist.

For deterministic demo users and data plus web/mobile test links, see [docs/demo-data.md](docs/demo-data.md).

## Run locally (Phase 1)

```bash
cp .env.example .env
make up
cd backend && go run ./cmd/api
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

API documentation: `http://localhost:8080/docs` and `http://localhost:8080/docs/openapi.yaml`. Prometheus metrics: `http://localhost:8080/metrics` (restrict at the edge in production).

## Verification

```bash
make fmt
make lint
make test
make test-db   # additionally runs migration tests against PostgreSQL on :55433
make verify-release  # fmt + lint + tests + mobile/admin typechecks + admin audit + dependency/secret checks
```

## Mobile app (Phases 6–7)

```bash
cd mobile
cp .env.example .env
npm install
npm run typecheck
npm start
```

Set `EXPO_PUBLIC_API_BASE_URL` to the reachable API base URL for your device or emulator. The app supports offline visit capture, offline order capture, and sync of distribution read models through `/api/v1/sync/*`.

## Admin web (Phase 10)

```bash
cd admin-web
cp .env.example .env
npm install
npm run dev
```

Set `VITE_API_BASE_URL` to `http://localhost:8080` when running Vite against the local API. Leave it empty for same-origin deployment behind nginx.

With Docker Compose:

```bash
docker compose -f infrastructure/docker-compose.yml up -d
open http://localhost:8081

docker compose -f infrastructure/docker-compose.yml --profile edge up -d
open http://localhost/admin
```

The console provides login/refresh-aware sessions and screens for tenants, users, roles/permissions, organization hierarchy, master data, distribution, reports, integrations and sync health. UI permission checks only hide actions; the API remains the enforcement boundary.

## Repository layout

| Path | Purpose |
| --- | --- |
| `backend/` | Go modular monolith (Gin, GORM, PostgreSQL) |
| `backend/migrations/` | Versioned SQL migrations |
| `mobile/` | Expo / React Native field application |
| `admin-web/` | Vite / React administration console |
| `infrastructure/` | Docker Compose and reverse proxy configuration |
| `docs/` | Architecture, database, API, security and deployment documentation |

## Design principles

- Multi-tenant by default; tenant context is derived from the authenticated session, never from the request body.
- Modular monolith with explicit constructor-based dependency injection.
- SQL migrations are the source of truth for the schema; GORM AutoMigrate is never used.
- Database transactions for multi-step operations; PostgreSQL RLS as defense in depth.
- Offline-first mobile with idempotency keys and a durable operation queue.
- Append-only audit trail for sensitive operations.

No real patient data or prescription-level patient information is stored by design.
