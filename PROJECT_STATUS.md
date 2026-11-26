# Project status

Last updated: Mobile redesign Phase 5 (demo data) verified end to end.

## Completed

### Phase 1 — Foundation
- [x] Repository initialized as a monorepo.
- [x] Go 1.25 backend module, Gin server, graceful shutdown.
- [x] Environment-driven configuration with production secret validation.
- [x] Structured logging (zap), request IDs, access logs, recovery, CORS, rate limiting, security headers.
- [x] PostgreSQL connection pool via GORM/pgx, DB ping.
- [x] Versioned SQL migration runner with advisory lock and transactional up/down scripts.
- [x] Foundation migration (utilities, RLS helper functions, non-superuser application role).
- [x] Liveness/readiness endpoints including migration-state check.
- [x] OpenAPI 3 document served at `/docs/openapi.yaml`.
- [x] Docker Compose for PostgreSQL, Redis, API, nginx; backend Dockerfile.
- [x] Unit tests for configuration, migration loader, HTTP liveness/docs.

### Phase 2 — Identity and tenancy
- [x] Tenants, tenant settings, legal entities, users, roles, permissions, user_roles, sessions, password resets, idempotency keys, audit logs tables.
- [x] Seeded configurable roles/permissions (13 roles, 36 permissions) with data scopes.
- [x] JWT access/refresh tokens, refresh rotation with family revocation on reuse, logout, sessions, password change/reset interfaces.
- [x] bcrypt password hashing, login lockout after repeated failures.
- [x] Tenant context derived from authenticated session; request-scoped transaction with `SET LOCAL ROLE pharmaos_app` and tenant GUCs.
- [x] PostgreSQL RLS enforced on users, roles, tenant settings, legal entities, sessions and audit logs.
- [x] Tenant/user/role/permission management APIs with permission middleware.
- [x] Bootstrap CLI (`go run ./cmd/bootstrap`) driven by environment variables.
- [x] Integration tests: cross-tenant user access blocked, tenant list filtered, cross-tenant user creation forbidden, RLS probe, auth login/refresh rotation.
- [x] OpenAPI document covers all Phase 2 routes.

### Phase 3 — Organization and employees
- [x] Organization migration: divisions, branches, regions, territories, positions, employees, effective-dated assignments, manager history, transfers, RLS.
- [x] Business division CRUD API under `/api/v1/divisions`.
- [x] Region and territory APIs under `/api/v1/regions` and `/api/v1/territories`; territory hierarchy validated in the database.
- [x] Branch API and organization hierarchy endpoint under `/api/v1/organization`.
- [x] Employee API under `/api/v1/employees` with position management, assignment history, and transactional transfers.
- [x] Phase 3 routes wired into the authenticated, tenant-scoped API group.
- [x] OpenAPI document updated for Phase 3 routes.

### Phase 4 — Healthcare CRM and catalog
- [x] Master-data migration: doctors, chemists, stockists, doctor-territory assignments, product categories, products, targets, indexes, and RLS.
- [x] Doctor CRUD and effective-dated territory assignment APIs under `/api/v1/doctors`.
- [x] Chemist and stockist CRUD APIs under `/api/v1/chemists` and `/api/v1/stockists`.
- [x] Product category and product APIs under `/api/v1/product-categories` and `/api/v1/products`.
- [x] Target CRUD API under `/api/v1/targets` with entity/employee existence checks and period validation.
- [x] Phase 4 routes wired into the authenticated, tenant-scoped API group.
- [x] OpenAPI document updated for Phase 4 routes.

### Phase 5 — Field execution
- [x] Field-execution migration: tour plans/lines, visits, DCR/product details, visit attachments, indexes, and RLS.
- [x] Data-scope helper supporting `SELF`, `TEAM`, `DIVISION`, `TERRITORY`, and `TENANT` access.
- [x] Tour plan API under `/api/v1/tour-plans` with lines, submit, approve, reject, and audit events.
- [x] Visit API under `/api/v1/visits` with idempotent creation, GPS check-in, submit, attachments, and audit events.
- [x] Daily call report API under `/api/v1/dcr` with product details, submit, approve, reject, and audit events.
- [x] Phase 5 routes wired into the authenticated, tenant-scoped API group.
- [x] OpenAPI document updated for Phase 5 routes.

### Phase 6 — Mobile offline application
- [x] Expo app scaffold under `mobile/` with login, home/sync status, and offline visit capture screens.
- [x] Secure token and session storage via `expo-secure-store`.
- [x] SQLite read models, sync state, and durable outbox with retry/backoff and dead-letter handling.
- [x] Scope-aware pull endpoint under `/api/v1/sync/changes` with opaque keyset cursors.
- [x] Idempotent operation push endpoint under `/api/v1/sync/operations` with per-operation results.
- [x] Sync status endpoint under `/api/v1/sync/status` with server time and queue hints.
- [x] Client-generated UUIDs accepted by visit, tour, and DCR create endpoints.
- [x] OpenAPI document and offline-sync design updated for Phase 6 routes.

### Phase 7 — Distribution Connect
- [x] Distribution migration: company-distributor links, distributor profiles, orders/items, inventory, inventory movements, secondary sales, link-aware RLS, triggers, indexes, role permissions, and grants.
- [x] Link-aware services and APIs for `/api/v1/distribution-links`, `/api/v1/distributors`, `/api/v1/orders`, `/api/v1/inventory`, `/api/v1/inventory/movements`, and `/api/v1/secondary-sales`.
- [x] Idempotent order creation with client-generated UUIDs and replay-safe `Idempotency-Key` handling.
- [x] Server-authoritative stock reservations on order approval using row locks, version checks, and per-order-line movement idempotency.
- [x] Immediate loss of cross-tenant visibility when a link is suspended or ended.
- [x] Mobile sync read models for distributors, links, orders, order items, inventory, inventory movements, and secondary sales.
- [x] Offline order capture in `mobile/` with `order.create` queued through the durable sync outbox.
- [x] Sync operations for `order.create`, `order.submit`, `order.approve`, `secondary_sale.create`, and `inventory.movement`.
- [x] OpenAPI document and design docs updated for Phase 7 routes and sync contracts.

### Phase 8 — Insight, audit and integrations
- [x] Migration `000007__insights_integrations` adds report definitions, report snapshots, integration connections, webhook deliveries, RLS, indexes, permissions, role grants, and an immutable audit trigger.
- [x] Audit logs are append-only; `UPDATE` and `DELETE` are rejected by database trigger.
- [x] Scope-aware field activity and distribution dashboards under `/api/v1/reports/*`, including link-aware distribution metrics across active company-distributor links.
- [x] Immutable audit trail query with tenant, date, module, action, and entity filters under `/api/v1/reports/audit`.
- [x] Saved report definitions and report snapshots APIs under `/api/v1/reports/definitions` and `/api/v1/reports/snapshots`.
- [x] Integration connection API under `/api/v1/integrations/connections`; secrets are referenced through environment variable names and never returned.
- [x] Idempotent webhook event publishing under `/api/v1/integrations/events`, delivery history under `/api/v1/integrations/deliveries`, and manual retry for failed/dead deliveries.
- [x] Webhook dispatcher worker with advisory-lock coordination, exponential backoff, dead-letter status, HMAC signatures, and graceful startup/shutdown wiring.
- [x] OpenAPI document and design docs updated for Phase 8 routes and policies.

### Phase 9 — Hardening and release
- [x] Migration `000008__hardening` adds a shared PostgreSQL rate-limit counter table and a `pharmaos_rate_limit` security-definer function.
- [x] Distributed rate limiter with in-process fallback, explicit trusted-proxy configuration, and production configuration validation.
- [x] Prometheus-formatted `/metrics` endpoint with request counts, latency summaries, and security-event counters.
- [x] Production validation enforces strong JWT secrets, explicit CORS allowlists, HTTPS-only webhooks, and PostgreSQL-backed rate limiting.
- [x] Migration checksum verification prevents edited applied migrations from being skipped or rolled back.
- [x] Regression tests for permission denial, rate-limit function behavior, metrics exposure, config hardening, and trusted-proxy validation.
- [x] Operational scripts for load smoke tests, backup/restore drills, dependency verification, and inline secret scanning.
- [x] Deployment and API documentation finalized for observability, scaling, and release validation.

### Phase 10 — Admin web console
- [x] Vite / React / TypeScript admin app under `admin-web/` with typed API client, bearer tokens, refresh-on-401 and permission-aware navigation.
- [x] Login, dashboard, tenant, user, role/permission, organization, master-data, distribution, reports, integrations and sync screens.
- [x] Forms and actions for core administrative writes, including idempotent distribution writes and webhook retry/status controls.
- [x] Admin web Docker image with nginx SPA fallback and same-origin `/api/` proxy; edge nginx exposes `/admin/` alongside the API.
- [x] Makefile targets `admin-web-typecheck` and `admin-web-build`; release verification includes the admin typecheck.

### Mobile redesign — Phase 1
- [x] Backend API inventory captured in `docs/mobile-backend-integration.md` from live routes, handlers, DTOs and OpenAPI; unavailable capabilities are marked `MISSING`.
- [x] Expo Router foundation added with guarded routes, session bootstrap, query client, gesture provider and safe-area handling.
- [x] Shared design system added with tokens, typography, screen/card/field/button/tag/loading primitives and accessible states.
- [x] Login rewritten with React Hook Form + Zod validation, stable device ID, in-app errors, loading state and real `/api/v1/auth/login` integration.
- [x] Session state moved to a Zustand store; refresh failure clears the session; tenant data is purged on tenant switch and logout.
- [x] HTTP client migrated to Axios with envelope unwrapping, request IDs, bearer injection, single-flight refresh, timeout handling and structured errors.
- [x] Home shell reads live `/api/v1/sync/status` through TanStack Query and exposes sign-out.
- [x] SDK 52 dependencies aligned with Expo; web export added for bundle verification.

### Mobile redesign — Phase 2
- [x] Role-aware tab shell added with permission-gated Visits, Tours, DCR, Catalog, Orders and Profile destinations.
- [x] Axios client extended with envelope-aware paginated reads for live list queries.
- [x] Auth bootstrap enriches the session from `/api/v1/auth/me` with privileged status, division IDs and territory IDs.
- [x] Dashboard reads live field activity and distribution reports, sync status and role-specific quick actions.
- [x] Visit workflow supports live customer selection, current-location check-in, draft creation, detail view and submission.
- [x] Tour plan workflow supports plan creation, line items, submission, approval and rejection.
- [x] Daily call report workflow supports report creation, product outcomes, submission, approval and rejection.
- [x] Catalog screen exposes permission-filtered doctors, chemists, stockists and products with live search.
- [x] Order workflow supports distributor links or manual distributor IDs, multiple line items, detail view, submission, approval and cancellation.
- [x] Inventory and secondary-sales snapshots and profile security screens added; password change and active session count use real auth endpoints.

### Mobile redesign — Phase 3
- [x] Sync provider added with NetInfo connectivity events, AppState foreground triggers, a 60-second interval, single-flight runs and authentication gating.
- [x] Outbox schema hardened with error codes, resolution hints and update timestamps; conflicts are stored separately and terminal failures are no longer retried automatically.
- [x] Sync engine maps backend `CONFLICT` results to manual review and records retryable/non-retryable error codes.
- [x] Sync center screen added with pending/problem/cached metrics, last-run results, retry and use-server conflict resolution actions.
- [x] Home dashboard exposes offline sync state, manual sync and a link to the sync center.
- [x] Visit and order forms can queue durable offline operations; offline orders now include every line item.

### Mobile redesign — Phase 4 (auth recovery)
- [x] Forgot-password flow calls `/api/v1/auth/forgot-password` with validated email input and shows the accepted confirmation.
- [x] Reset-password flow calls `/api/v1/auth/reset-password` with token, 10-character password and confirmation validation.
- [x] Reset screens accept an optional query-string token and return users to sign-in after success.
- [x] Login screen exposes a forgot-password link; mobile integration inventory marks auth recovery as available.

### Mobile redesign — Phase 5 (demo data)
- [x] Idempotent `backend/cmd/seed` CLI creates deterministic demo tenants, users, organization, master data, field records and distribution records.
- [x] `make seed` target added and linked from the README.
- [x] `docs/demo-data.md` provides demo credentials, web/mobile links, API links and a suggested end-to-end test flow.
- [x] JSONB scan support added via `backend/internal/platform/jsontypes` for master data, divisions and tenants so seeded rows list through the API.
- [x] Compose API CORS origins include Expo web on `5173`, `8081` and `19006`.

## Pending

- [ ] Remaining production rollout, device/soak testing and post-release scaling work.

## Verification log

- `go mod tidy`, `gofmt -w`, `go vet ./...`, `go test ./...`: passing (with `TEST_DATABASE_URL` pointing to PostgreSQL on port 55433).
- `go build ./...`: passing after Phase 3.
- Migrations through `000003__organization.up.sql` applied against live PostgreSQL test database: passing.
- `go build ./...` and `go vet ./...`: passing after Phase 4.
- Migrations through `000004__master_data.up.sql` applied against live PostgreSQL test database: passing.
- `gofmt -w`, `go build ./...`, `go vet ./...`, and `go test ./...`: passing after Phase 5.
- Migrations through `000005__field_execution.up.sql` applied against live PostgreSQL test database: passing.
- `gofmt -w`, `go build ./...`, `go vet ./...`, and package-level `go test ./internal/...`: passing after Phase 6.
- OpenAPI YAML parse: passing after Phase 6.
- `npm install` and `npm run typecheck`: passing for `mobile/` after Phase 6.
- Phase 7: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./internal/...`, integration test compilation, OpenAPI YAML parse (version `1.0.0-phase7`), and `npm run typecheck`: passing.
- Phase 8: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./internal/...`, integration test compilation, OpenAPI YAML parse (version `1.0.0-phase8`, 90 paths), and `npm run typecheck`: passing.
- Phase 9: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./...`, integration test compilation, OpenAPI YAML parse (version `1.0.0-phase9`, 91 paths), mobile typecheck, `go mod verify`, secret scan and `git diff --check`: passing.
- Docker Compose `postgres_test` on port 55433: all migrations through `000008__hardening` apply and `TEST_DATABASE_URL=... go test ./...` passes, including DB-backed integration, migration and hardening assertions.
- Test-found fixes applied: idempotent role grants in `000006__distribution`, optional distribution link `EffectiveFrom`, reusable response bodies in integration helpers.
- Phase 10: `npm install`, `npm audit --audit-level=moderate` (0 vulnerabilities), `npm run typecheck` and `npm run build` passing for `admin-web/`; Docker image builds from the pinned lockfile and nginx serves the SPA with an `/api/` proxy.
- Compose smoke test: `GET /` on `admin-web:8081` returned 200 and `GET /healthz` through the admin nginx `/api` proxy returned 200.
- Manual HTTP smoke test: registration/bootstrap login issued access + refresh tokens; cross-tenant API access returns 404/403.
- Mobile redesign Phase 1: `npx expo install --check` reports all Expo-managed dependencies aligned with SDK 52.
- Mobile redesign Phase 1: `npm run typecheck` passes for `mobile/`.
- Mobile redesign Phase 1: `npm run export:web` bundles Expo Router routes successfully (`1005` modules).
- Mobile redesign Phase 1: `npm audit` reports transitive advisories in the pinned Expo SDK 52 toolchain; forced fixes require an Expo major upgrade and were intentionally not applied.
- Mobile redesign Phase 2: `npm run typecheck` passes for `mobile/`.
- Mobile redesign Phase 2: `npm run export:web` bundles the role-aware Expo Router app successfully (`1044` modules).
- Mobile redesign Phase 2: `npx expo install --check` reports all Expo-managed dependencies aligned with SDK 52.
- Mobile redesign Phase 3: `npm run typecheck` passes for `mobile/`.
- Mobile redesign Phase 3: `npm run export:web` bundles the sync-hardened Expo Router app successfully (`1060` modules).
- Mobile redesign Phase 3: `npx expo install --check` reports all Expo-managed dependencies, including `@react-native-community/netinfo`, aligned with SDK 52.
- Mobile redesign Phase 3: `git diff --check` and `scripts/scan-secrets.sh .` pass.
- Mobile redesign Phase 4: `npm run typecheck` passes for `mobile/`.
- Mobile redesign Phase 4: `npm run export:web` bundles the auth-recovery routes successfully (`1062` modules).
- Mobile redesign Phase 5: `gofmt` and `go vet ./...` pass with `backend/cmd/seed` included.
- Demo seed verification: rebuilt Compose API returns seeded counts `doctors=3`, `products=3`, `chemists=2`, `stockists=2`, `product-categories=2`, `visits=4`, `tour-plans=1`, `dcr=2`, `orders=2`, `inventory=3`, `secondary-sales=2`, `tenants=3`, `divisions=1`, `regions=1`, `territories=1`.
- Demo seed fix verification: `go build ./...`, `go vet ./...`, `go test ./...`, mobile `npm run typecheck`, `npx expo install --check`, `git diff --check`, and `scripts/scan-secrets.sh .` pass after the JSONB scan fix.
- Mobile runtime fix: pinned `query-string@7.1.3` for Expo Router 4.0.22 compatibility; cleared Metro caches, then `npm run export:web`, `npm run typecheck`, `npx expo install --check`, `git diff --check`, and secret scan pass.
- Mobile API wrapper fix: raw envelope responses now return `response.data` instead of the full Axios response, so `getPage()` lists render as arrays; `npm run typecheck`, `npm run export:web`, `git diff --check`, and secret scan pass.
- Android keyboard fix: `softwareKeyboardLayoutMode` set to `pan` in `mobile/app.json` and `android:windowSoftInputMode="adjustPan"` in the generated manifest to stop focused inputs flashing a black screen.
- Runtime API fix: `scope.Apply` now builds employee subqueries from a clean session with an explicit `employees` table, eliminating invalid `IN ()` SQL; rebuilt Compose API verified `GET /api/v1/sync/status` 200 and `GET /api/v1/sync/changes?limit=200` 200 with 43 changes for the MR demo user.
- Employee profile fix: added `GET /api/v1/employees/me` (no `employee.view` permission required); verified MR login resolves `Arjun Mehta`, DCR detail/products return 200, and missing profiles return `NO_EMPLOYEE_PROFILE`.
- Demo seed fix: added linked employee profiles for `admin@demo.test` (`ADM-001`), `viewer@demo.test` (`VIEW-001`) and `distributor@demo.test` (`DIST-001`); re-seeded live DB and verified `/employees/me` returns 200 for all five demo accounts.
- Runtime hardening: location capture uses an 8s timeout plus last-known fallback; detail navigation uses `router.push` so back buttons remain available; visit/DCR/tour/order screens display customer, employee, product and distributor names instead of raw IDs.
- Profile display fix: replaced raw `User ID` and `Tenant` UUID rows with employee name/code, email and readable access scope; mobile `npm run typecheck` and `npm run export:web` pass.
- Final verification: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./...`, mobile `npm run typecheck`, mobile `npm run export:web`, `git diff --check`, and `scripts/scan-secrets.sh .` pass.
