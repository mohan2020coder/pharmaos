# Demo data and test links

## Start the stack

Terminal 1 — database and API:

```bash
make up
cd backend
go run ./cmd/api
```

The API applies pending migrations on startup and listens on `http://localhost:8080`.

Terminal 2 — seed deterministic demo data:

```bash
make seed
```

`make seed` is idempotent and uses fixed UUIDs, so it can be rerun safely.

## Demo accounts

Password for all demo accounts: `DemoPass123!`

| Tenant code | Email | Role | Use |
| --- | --- | --- | --- |
| `DEMO` | `admin@demo.test` | `SUPER_ADMIN` | Full admin access and all permissions |
| `DEMO` | `manager@demo.test` | `COMPANY_ADMIN` | Manager-level web/mobile navigation |
| `DEMO` | `mr@demo.test` | `MR` | Field mobile flows, visits, tours, DCR |
| `DEMO` | `viewer@demo.test` | `REPORT_VIEWER` | Read-only report testing |
| `DEMODIST` | `distributor@demo.test` | `DISTRIBUTOR_ADMIN` | Distributor-side orders and secondary sales |

Override the password with `PHARMAOS_SEED_PASSWORD` before running `make seed` if needed.

Login requests use `tenant_code`, `email` and `password`, for example `{ "tenant_code": "DEMO", "email": "mr@demo.test", "password": "DemoPass123!" }`.

## Web admin

Standalone Vite admin:

```bash
cd admin-web
set VITE_API_BASE_URL=http://localhost:8080
npm run dev
```

Open `http://localhost:5173/login`.

Compose admin:

```bash
docker compose -f infrastructure/docker-compose.yml up -d --build api admin-web
```

Open `http://localhost:8081/login`.

Edge profile:

```bash
docker compose -f infrastructure/docker-compose.yml --profile edge up -d --build
```

Open `http://localhost/admin/login`.

Use tenant `DEMO` and any account from the table above.

## Mobile

Start the Expo app:

```bash
cd mobile
npm start
```

| Target | API base URL | App URL |
| --- | --- | --- |
| Expo web | `http://localhost:8080` | `http://localhost:19006` |
| Android emulator | `http://10.0.2.2:8080` | Expo Go / dev server QR |
| iOS simulator | `http://localhost:8080` | Expo Go / dev server QR |
| Physical device | `http://<host-lan-ip>:8080` | Expo Go / dev server QR |

For iOS simulator or a physical device, set `EXPO_PUBLIC_API_BASE_URL` to the reachable API URL before `npm start`. Expo web needs the API CORS origin `http://localhost:19006`; both the local `go run ./cmd/api` default and the Compose API service allow `5173`, `8081` and `19006` on `localhost` and `127.0.0.1`.

Mobile test flow:

1. Sign in with tenant `DEMO`, email `mr@demo.test`, password `DemoPass123!`.
2. Open **Sync center** from the home dashboard and tap **Sync now**.
3. Verify visits, tour plans, DCRs, doctors, chemists, stockists, products, orders and inventory are pulled from the API.
4. Use **Record visit** or **New order** then **Queue offline** to create a durable local operation, and sync it back when online.
5. Sign out and sign in with `admin@demo.test` to test permission-aware navigation.

Auth recovery screens are at `http://localhost:19006/forgot-password` and `http://localhost:19006/reset-password?token=<token>` on Expo web. Password delivery depends on the configured reset sender; the reset screen also accepts a pasted token.

## API links

| Link | Purpose |
| --- | --- |
| `http://localhost:8080/healthz` | Liveness |
| `http://localhost:8080/readyz` | Readiness including migration state |
| `http://localhost:8080/docs` | API docs UI |
| `http://localhost:8080/docs/openapi.yaml` | OpenAPI document |
| `http://localhost:8080/metrics` | Prometheus metrics |

## What the seed creates

- 2 tenants: `DEMO` company and `DEMODIST` distributor.
- 5 users with deterministic IDs and demo roles.
- Organization: one division, branch, region, territory, two positions and five employees (`ADM-001`, `MGR-001`, `MR-001`, `VIEW-001`, `DIST-001`), so every demo account resolves through `GET /api/v1/employees/me`.
- Master data: 3 doctors, 2 chemists, 2 stockists, 2 product categories and 3 products.
- Field execution: 1 submitted tour plan with 2 lines, 4 visits and 2 submitted DCRs with product details.
- Distribution: 1 active company-distributor link, 1 distributor profile, 3 inventory rows, 2 orders with line items and 2 secondary sales.
