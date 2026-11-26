# API Design

> **Status:** Health, documentation, observability and all business endpoints through Phase 9 are implemented in the checked-in OpenAPI contract.

## 1. Conventions

| Item | Rule |
| --- | --- |
| Base path | `/api/v1` (declared as the OpenAPI server URL). nginx preserves the `/api/...` prefix. |
| Encoding | `application/json`; UTF-8; UUIDs as strings; timestamps as RFC 3339 UTC. |
| Auth | `Authorization: Bearer <access_token>` on all business routes; no tenant header. |
| Correlation | `X-Request-ID` is accepted from clients and echoed on every response. |
| Idempotency | `Idempotency-Key: <uuid>` required on retryable HTTP mutations (orders, inventory, secondary sales); `X-Idempotency-Key` is accepted for older clients. Sync operations carry `idempotency_key` in the operation body. |
| Pagination | `page` (default 1), `page_size` (default 20); `meta` uses `PageMeta`. Keyset cursors are used for `/sync/changes`. |
| Tenant input | Any `tenant_id` in a body/query is ignored or rejected with 400; identity comes from the JWT. |
| Versioning | URI major version; additive fields are non-breaking; breaking changes require `/api/v2`. |

## 2. Implemented endpoints

| Method | Path | Purpose | Response |
| --- | --- | --- | --- |
| GET | `/healthz` | Liveness and uptime | `{"success":true,"message":"OK","data":{"status":"alive","uptime_seconds":...}}` |
| GET | `/readyz` | DB ping + migration version gate | 200 when ready, 503 with `database`, `migrations` or `applied`/`expected` details |
| GET | `/docs` | Lightweight API landing page | HTML |
| GET | `/docs/openapi.yaml` | Embedded OpenAPI 3.0.3 spec | `application/yaml` |
| GET | `/metrics` | Prometheus-formatted request and security metrics | `text/plain` |
| GET | `/reports/field-activity` | Scope-aware visit/DCR dashboard | `FieldActivityReport` |
| GET | `/reports/distribution` | Link-aware order/inventory/sales dashboard | `DistributionSummaryReport` |
| GET | `/reports/audit` | Immutable audit trail query | Paged `AuditEntry` list |
| GET/POST | `/reports/definitions` | Saved report definitions | Paged list / created definition |
| GET/POST | `/reports/snapshots` | Report snapshots | Paged list / created snapshot |
| GET | `/reports/snapshots/{id}` | Single report snapshot | `ReportSnapshot` |
| GET/POST | `/integrations/connections` | Integration connection management | Paged list / created connection |
| PATCH | `/integrations/connections/{id}/status` | Enable or disable a connection | Updated flag |
| DELETE | `/integrations/connections/{id}` | Delete a connection | Deleted flag |
| POST | `/integrations/events` | Queue idempotent webhook deliveries | Created delivery list |
| GET | `/integrations/deliveries` | Webhook delivery history | Paged delivery list |
| POST | `/integrations/deliveries/{id}/retry` | Requeue failed/dead delivery | Retried flag |

`backend/internal/platform/apidocs/openapi.yaml` describes implemented routes through Phase 9 (`/metrics`, `/auth/login`, `/users`, `/doctors`, `/products`, `/targets`, `/tour-plans`, `/visits`, `/dcr`, `/distribution-links`, `/distributors`, `/orders`, `/inventory`, `/secondary-sales`, `/reports/*`, `/integrations/*`, `/sync/changes`, `/sync/operations`, `/sync/status`); it is the versioned HTTP contract.

## 3. Response and error envelope

```json
{
  "success": true,
  "message": "OK",
  "data": { "id": "8f2e..." },
  "meta": { "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
}
```

```json
{
  "success": false,
  "message": "You do not have access to this resource",
  "errors": { "code": "FORBIDDEN" }
}
```

`response.OK`, `response.Created`, `response.JSON` and `response.Fail` are implemented in `backend/internal/platform/response/response.go`. Handlers translate `apperrors.AppError` through `apperrors.Status`/`apperrors.Public` into this envelope.

| Code | HTTP | Meaning |
| --- | --- | --- |
| `BAD_REQUEST` | 400 | Malformed request or forbidden tenant override |
| `UNAUTHORIZED` | 401 | Missing, expired or invalid access token |
| `FORBIDDEN` | 403 | Missing permission or out-of-scope operation |
| `NOT_FOUND` | 404 | Resource unknown |
| `TENANT_MISMATCH` | 404 | Resource belongs to another tenant (deliberately indistinguishable from `NOT_FOUND`) |
| `CONFLICT` | 409 | Duplicate key, stale version or idempotency payload mismatch |
| `VALIDATION_FAILED` | 422 | Field-level validation errors |
| `TOO_MANY_REQUESTS` | 429 | Rate limit; includes `Retry-After` |
| internal | 500 | Generic message; details logged with request ID only |

## 4. Authentication endpoints (planned, Phase 2)

```http
POST /api/v1/auth/login
Content-Type: application/json

{ "email": "mr@example.com", "password": "..." }
```

```json
{
  "success": true,
  "message": "OK",
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "<opaque token>",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

| Route | Purpose |
| --- | --- |
| `POST /auth/login` | Verify credentials, create session, issue access + refresh tokens. |
| `POST /auth/refresh` | Rotate refresh token, issue new access token; reload roles/scope. |
| `POST /auth/logout` | Revoke session and refresh token (Redis-backed, planned). |
| `GET /auth/me` | Return user, tenant type, roles, permissions and data scope. |

Configuration already defines `ACCESS_TOKEN_TTL=15m`, `REFRESH_TOKEN_TTL=720h`, `JWT_SECRET` and `PASSWORD_HASH_COST`. Production startup fails without `JWT_SECRET`.

## 5. Resource surface by module

| Module | Representative routes | Permission | Phase |
| --- | --- | --- | --- |
| Tenants/users | `GET/POST /tenants`, `/users`, `/roles` | `tenants:read`, `users:write`, `roles:write` | 2 |
| Organization | `/divisions`, `/territories`, `/employees` | `organization:read/write` | 3 |
| CRM | `GET/POST /doctors`, `/chemists`, `/stockists`, `/products`, `/targets` | `doctors:read`, `products:write`, `targets:write` | 4 |
| Tours | `POST /tour-plans`, `GET /tour-plans/{id}`, `POST /tour-plans/{id}/submit` | `tourplan.create`, `tourplan.approve` | 5 |
| Visits | `POST /visits`, `GET /visits`, `POST /visits/{id}/submit` | `visit.create`, `visit.submit` | 5 |
| DCR | `POST /dcr`, `GET /dcr/{id}`, `POST /dcr/{id}/products`, `POST /dcr/{id}/submit` | `dcr.create`, `dcr.review` | 5 |
| Distribution | `GET/POST /distribution-links`, `PATCH /distribution-links/{id}/status`, `GET/POST /distributors`, `GET/POST /orders`, `GET /inventory`, `GET/POST /inventory/movements`, `GET/POST /secondary-sales` | `link.manage`, `distributor.view/manage`, `order.view/manage/approve`, `inventory.view/manage`, `secondary_sales.view/import` | 7 |
| Orders | `POST /orders` (idempotent), `GET /orders/{id}` | `orders:create`, `orders:read` | 7 |
| Secondary sales | `POST /secondary-sales`, `GET /secondary-sales` | `secondary_sales:write` | 7 |
| Inventory | `GET /inventory`, `POST /inventory/movements` | `inventory:read/write` | 7 |
| Reporting | `GET /reports/{name}`, `GET /dashboards/{name}` | `reports:read` | 8 |
| Audit | `GET /audit-log`, `GET /audit-log/{id}` | `audit:read` (append-only) | 8 |
| Integrations | `/integrations/connections`, `/integrations/webhooks` | `integrations:write` | 8 |
| Sync | `GET /sync/changes`, `POST /sync/operations`, `GET /sync/status` | authenticated device | 6 |
| Idempotency diagnostics | `GET /idempotency-keys/{key}` (not yet implemented) | `platform:read` | 8 |

## 6. Idempotency contract

For every retryable `POST`, the client sends a UUID key that is stable for the logical operation across retries and app restarts:

```http
POST /api/v1/orders
Authorization: Bearer <jwt>
X-Idempotency-Key: 3f8a2c10-6b11-4d0c-9f7b-1e9e5a4d0b22

{ "distributor_id": "...", "lines": [ ... ] }
```

Server behavior:

1. Look up `(tenant_id, key, endpoint)` inside the tenant transaction.
2. Replay the stored status/body if the request hash matches.
3. Return 409 if the same key is reused with a different payload.
4. On success, persist the response atomically with the business write; on failure ≥ 400 the transaction rolls back and no successful record is stored.
5. Expire keys only after the maximum client retry window.

CORS permits both `X-Idempotency-Key` and `Idempotency-Key` in `backend/internal/middleware/common.go`. Direct visit, order, secondary-sale, and inventory-movement creation use the `Idempotency-Key` header. Sync operations carry `idempotency_key` in each operation; the backend replays sync results by client operation ID and uses the operation key as the domain idempotency key.

## 7. Filtering, sorting and scope

- Filters are explicit query parameters: `status`, `search`, `territory_id`, `from`, `to`, `updated_since`.
- Sorting uses `sort=-created_at,id`; allowlisted fields only.
- The service layer intersects request filters with the principal's data scope; narrowing is allowed, widening returns 403 or an empty page as appropriate.
- Cross-tenant distribution resources are visible only through an active `dist_company_distributor_links` row. See [tenant-isolation.md](tenant-isolation.md).

## 8. Sync response shape

```json
{
  "success": true,
  "message": "OK",
  "data": {
    "next_cursor": "eyJ1cGRhdGVkX2F0Ijoi...",
    "has_more": false,
    "changes": [
      { "entity": "doctor", "entity_id": "...", "version": 12, "op": "UPSERT", "payload": {} }
    ],
    "operations": [
      { "client_operation_id": "...", "status": "APPLIED", "entity_id": "..." },
      { "client_operation_id": "...", "status": "CONFLICT", "error": "STALE_VERSION" }
    ]
  }
}
```

## 9. Verification

`backend/internal/server/server_test.go` uses `httptest` to assert probe/docs routes; Phase 9 adds metrics, permission-denial, config, trusted-proxy and rate-limit regression coverage. DB-backed integration tests pin tenant isolation, idempotent replay and webhook/rate-limit behavior when `TEST_DATABASE_URL` is available. The OpenAPI file must be updated in the same change as any new route.

Related documents: [architecture.md](architecture.md), [permissions.md](permissions.md), [mobile-offline-sync.md](mobile-offline-sync.md).
