# Mobile Backend Integration Inventory

> **Status:** authoritative inventory prepared for the mobile redesign. This document records APIs that exist in the current backend. Anything not listed here is `MISSING` until a backend route is added.

## 1. Source of truth

- Generated OpenAPI document: `backend/internal/platform/apidocs/openapi.yaml` (`1.0.0-phase9`, 91 paths), served at `GET /docs/openapi.yaml`.
- Route registration: `backend/internal/server/server.go:44-176`.
- Public auth: `authHandler.RegisterPublic(api)` at `backend/internal/server/server.go:153`.
- Protected API: `/api/v1` group with `Authenticate` + `TenantTransaction` middleware at `backend/internal/server/server.go:155-156`.
- Operational routes (`/healthz`, `/readyz`, `/docs`, `/metrics`) are public and outside `/api/v1`.
- Business API base path is `/api/v1`.
- The mobile client must not invent endpoints from roadmap language. Use this inventory and the OpenAPI document.

## 2. Transport conventions

### Success envelope

`backend/internal/platform/response/response.go:9-38`

```json
{
  "success": true,
  "message": "OK",
  "data": {},
  "meta": { "page": 1, "page_size": 20, "total": 42, "total_pages": 3 }
}
```

- `data` is omitted when empty.
- `meta` is present on paginated list endpoints.
- `message` is a human-readable server message, not a stable machine key.

### Error envelope

```json
{
  "success": false,
  "message": "Validation failed",
  "errors": {
    "field": "reason"
  }
}
```

- `errors` may be an object, map, or `null`.
- Application error codes from `backend/internal/platform/errors/errors.go` are **not** serialized as a top-level `code` field. Do not rely on a wire-level error code unless a specific endpoint documents one in `errors`.
- Common statuses/messages:
  - `400 Invalid JSON payload`
  - `401 Authentication required`
  - `401 Invalid or expired token`
  - `401 Invalid tenant claim`
  - `401 Invalid session claim`
  - `403 Permission denied` with `errors.required_permission`
  - `404 Resource not found`
  - `409` for conflict/stale-write cases
  - `422 Validation failed`
  - `429` rate limited
  - `500 Internal server error`

### Authentication

- Protected routes require `Authorization: Bearer <access_token>` (`backend/internal/middleware/authorization.go:14-52`).
- `SUPER_ADMIN` and tokens with `is_privileged: true` bypass permission checks (`backend/internal/middleware/authorization.go:45`, `backend/internal/auth/principal.go:22-27`).
- Business requests are tenant-scoped by the token's `tenant_id`, not by a client-selected tenant header.
- Role permission checks are enforced server-side with `RequirePermission` / `RequireAnyPermission`.
- Use `X-Request-ID` with a UUID for every request. Client-generated request IDs are accepted by middleware but not echoed in the success envelope.
- Writes that explicitly require idempotency use a UUID in the `Idempotency-Key` header, not only in the JSON body.

### Pagination, sorting, search

`backend/internal/platform/query/query.go:12-51`

- Query: `page` (default `1`), `page_size` (default `20`, max `200`), `sort` (`created_at|updated_at|name|email|code|status`; default `created_at`), `order` (`asc|desc`; default `desc`), `search`.
- Paginated responses use `meta.page`, `meta.page_size`, `meta.total`, `meta.total_pages`.
- Some endpoints have extra filters listed below.

## 3. Authentication and session APIs

| Method | Path | Auth | Request / notes | Response data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/v1/auth/login` | public | `tenant_code` (2–40), `email`, `password` (min 6), optional `device_id` | `{tokens, user}` |
| `POST` | `/api/v1/auth/refresh` | public | `{refresh_token}` | `{tokens, user:{id,tenant_id,roles}}` |
| `POST` | `/api/v1/auth/forgot-password` | public | `{email}`; always returns accepted | `{accepted,message}` |
| `POST` | `/api/v1/auth/reset-password` | public | `{token,new_password}` (min 10) | `{password_reset:true}` |
| `POST` | `/api/v1/auth/logout` | bearer | Revokes current session | `{logged_out:true}` |
| `GET` | `/api/v1/auth/me` | bearer | Current principal | `{id,tenant_id,email,roles,permissions,data_scope,is_privileged,division_ids,territory_ids}` |
| `GET` | `/api/v1/auth/sessions` | bearer | Current user's sessions | session list |
| `POST` | `/api/v1/auth/change-password` | bearer | `{current_password,new_password}` (min 10) | `{password_changed:true}` |

Sources: `backend/internal/auth/handler.go:18-32`, `backend/internal/auth/service.go:20-33`, `backend/internal/auth/handler.go:34-155`.

### Token pair

`backend/internal/auth/service.go:27-33`

```ts
type TokenPair = {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  refresh_expires_in: number;
};
```

### Login user

`backend/internal/auth/handler.go:49-52`

```ts
type LoginUser = {
  id: string;
  tenant_id: string;
  email: string;
  roles: string[];
  permissions: string[];
  data_scope: string;
};
```

Important: `/auth/refresh` returns only `id`, `tenant_id`, and `roles` in `data.user`. After refresh, keep the richer cached login/`me` profile or call `/auth/me`.

### Refresh behavior

- Refresh rotation and family revocation are implemented in `backend/internal/auth/service.go`.
- A reuse of a rotated refresh token revokes the session family; the client must clear local session state on any refresh failure.
- `401` during a protected request triggers one refresh retry in the current mobile client (`mobile/src/api/client.ts:38-86`). If refresh fails, clear the session and route to login.

## 4. Operational APIs

| Method | Path | Auth | Response data |
| --- | --- | --- | --- |
| `GET` | `/healthz` | public | `{status:"alive", uptime_seconds}` |
| `GET` | `/readyz` | public | `{status:"ready", database:"ok", migration_version}`; `503` when not ready |
| `GET` | `/docs` | public | HTML docs UI |
| `GET` | `/docs/openapi.yaml` | public | OpenAPI YAML |
| `GET` | `/metrics` | public | Prometheus text; restrict at edge |

Use `/readyz` for startup/splash gating if the mobile app needs backend readiness; do not treat `/healthz` as database readiness.

## 5. Tenant, user, role and permission APIs

### Tenants

`backend/internal/tenants/handler.go:21-28`

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/tenants` | `tenant.view` | List tenants visible to principal |
| `GET` | `/api/v1/tenants/:id` | `tenant.view` | Get tenant |
| `POST` | `/api/v1/tenants` | `tenant.manage` | Create tenant |
| `PATCH` | `/api/v1/tenants/:id` | `tenant.manage` | Update tenant |
| `PUT` | `/api/v1/tenants/:id/settings/:key` | `tenant.manage` | Set tenant setting |

### Users

`backend/internal/users/handler.go:22-32`

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/users` | `user.view` | List users |
| `GET` | `/api/v1/users/:id` | `user.view` | Get user |
| `POST` | `/api/v1/users` | `user.manage` | Create user |
| `PATCH` | `/api/v1/users/:id` | `user.manage` | Update user |
| `POST` | `/api/v1/users/:id/deactivate` | `user.manage` | Deactivate user |
| `GET` | `/api/v1/users/:id/roles` | `user.view` | List user role grants |
| `POST` | `/api/v1/users/:id/roles` | `user.manage` | Assign role |
| `DELETE` | `/api/v1/users/:id/roles/:role_id` | `user.manage` | Remove role |

### Roles and permissions

`backend/internal/roles/handler.go:21-29`, `backend/internal/permissions/handler.go:16-18`

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/roles` | `role.view` | List roles |
| `GET` | `/api/v1/roles/:id` | `role.view` | Get role |
| `POST` | `/api/v1/roles` | `role.manage` | Create role |
| `PATCH` | `/api/v1/roles/:id` | `role.manage` | Update role |
| `DELETE` | `/api/v1/roles/:id` | `role.manage` | Delete role |
| `PUT` | `/api/v1/roles/:id/permissions` | `role.manage` | Replace role permissions |
| `GET` | `/api/v1/permissions` | `role.view` | Permission catalogue |

Mobile should use `data.permissions` from login/`/auth/me` for UI gating only. Server permission checks remain authoritative.

## 6. Organization and employee APIs

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/divisions` | `employee.view` | List divisions |
| `GET` | `/api/v1/divisions/:id` | `employee.view` | Get division |
| `POST` | `/api/v1/divisions` | `employee.manage` | Create division |
| `PATCH` | `/api/v1/divisions/:id` | `employee.manage` | Update division |
| `DELETE` | `/api/v1/divisions/:id` | `employee.manage` | Delete division |
| `GET` | `/api/v1/regions` | `employee.view` | List regions; filter `division_id` |
| `POST` | `/api/v1/regions` | `employee.manage` | Create region |
| `GET` | `/api/v1/territories` | `employee.view` | List territories |
| `POST` | `/api/v1/territories` | `employee.manage` | Create territory |
| `GET` | `/api/v1/organization/hierarchy` | `employee.view` | Organization hierarchy |
| `GET` | `/api/v1/organization/branches` | `employee.view` | List branches |
| `POST` | `/api/v1/organization/branches` | `employee.manage` | Create branch |
| `GET` | `/api/v1/employees/me` | authenticated device | Resolve the employee linked to the current user |
| `GET` | `/api/v1/employees` | `employee.view` | List employees |
| `GET` | `/api/v1/employees/:id` | `employee.view` | Get employee |
| `GET` | `/api/v1/employees/:id/history` | `employee.view` | Assignment history |
| `POST` | `/api/v1/employees` | `employee.manage` | Create employee |
| `PATCH` | `/api/v1/employees/:id` | `employee.manage` | Update employee |
| `POST` | `/api/v1/employees/:id/transfer` | `employee.manage` | Transfer employee |
| `GET` | `/api/v1/positions` | `employee.view` | List positions |

`/api/v1/employees/me` returns `404` with code `NO_EMPLOYEE_PROFILE` when the authenticated user has no linked employee; mobile treats that as a null employee and shows a friendly message.

## 7. Master data and CRM APIs

### Doctors, chemists, stockists

| Resource prefix | `GET` list | `POST` create | `GET :id` | `PATCH :id` | `DELETE :id` |
| --- | --- | --- | --- | --- | --- |
| `/api/v1/doctors` | yes (`doctor.view`) | yes (`doctor.create`) | yes | yes (`doctor.update`) | yes (`doctor.update`) |
| `/api/v1/chemists` | yes (`chemist.view`) | yes (`chemist.manage`) | yes | yes | yes |
| `/api/v1/stockists` | yes (`stockist.view`) | yes (`stockist.manage`) | yes | yes | yes |

Doctor territory routes:

- `GET /api/v1/doctors/:id/territories` — `doctor.view`
- `POST /api/v1/doctors/:id/territories` — `doctor.update`

List filters include `is_active`, standard pagination/sort/search, and scope-based territory/division visibility.

### Products and categories

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/products` | `product.view` | List products |
| `POST` | `/api/v1/products` | `product.manage` | Create product |
| `GET` | `/api/v1/products/:id` | `product.view` | Get product |
| `PATCH` | `/api/v1/products/:id` | `product.manage` | Update product |
| `DELETE` | `/api/v1/products/:id` | `product.manage` | Delete product |
| `GET` | `/api/v1/product-categories` | `product.view` | List categories |
| `POST` | `/api/v1/product-categories` | `product.manage` | Create category |

### Targets

`/api/v1/targets` supports `GET` list, `POST` create, `GET /:id`, `PATCH /:id`, `DELETE /:id` with `target.view` / `target.manage`.

## 8. Field execution APIs

### Visits

`backend/internal/visits/handler.go:21-28`

| Method | Path | Permission | Headers / notes |
| --- | --- | --- | --- |
| `GET` | `/api/v1/visits` | `visit.view` | Filters: `employee_id`, `territory_id`, `entity_type`, `status`, `from`, `to` |
| `POST` | `/api/v1/visits` | `visit.create` | Requires `Idempotency-Key: <uuid>`; body may include client `id` |
| `GET` | `/api/v1/visits/:id` | `visit.view` | Get visit |
| `POST` | `/api/v1/visits/:id/submit` | `visit.create` | Submit visit |
| `POST` | `/api/v1/visits/:id/attachments` | `visit.create` | Records attachment metadata only |

Visit create fields (`backend/internal/visits/service.go:20-33`):

```ts
type CreateVisitRequest = {
  id?: string;
  employee_id: string;
  tour_plan_line_id?: string;
  territory_id?: string;
  entity_type: 'DOCTOR' | 'CHEMIST' | 'STOCKIST';
  entity_id: string;
  client_reference_id?: string;
  visited_at?: string;
  check_in_lat?: number;
  check_in_lng?: number;
  check_in_accuracy?: number;
  remarks?: string;
};
```

Attachment metadata fields (`backend/internal/visits/service.go:35-41`): `id?`, `file_name`, `content_type`, `storage_key`, `checksum`.

### Tour plans

`backend/internal/tours/handler.go:21-31`

| Method | Path | Permission |
| --- | --- | --- |
| `GET` | `/api/v1/tour-plans` | `tourplan.view` |
| `POST` | `/api/v1/tour-plans` | `tourplan.create` |
| `GET` | `/api/v1/tour-plans/:id` | `tourplan.view` |
| `POST` | `/api/v1/tour-plans/:id/submit` | `tourplan.create` |
| `POST` | `/api/v1/tour-plans/:id/approve` | `tourplan.approve` |
| `POST` | `/api/v1/tour-plans/:id/reject` | `tourplan.approve` |
| `GET` | `/api/v1/tour-plans/:id/lines` | `tourplan.view` |
| `POST` | `/api/v1/tour-plans/:id/lines` | `tourplan.create` |

### Daily call reports

`backend/internal/dcr/handler.go:21-30`

| Method | Path | Permission |
| --- | --- | --- |
| `GET` | `/api/v1/dcr` | `dcr.view` |
| `POST` | `/api/v1/dcr` | `dcr.create` |
| `GET` | `/api/v1/dcr/:id` | `dcr.view` |
| `GET` | `/api/v1/dcr/:id/products` | `dcr.view` |
| `POST` | `/api/v1/dcr/:id/products` | `dcr.create` |
| `POST` | `/api/v1/dcr/:id/submit` | `dcr.create` |
| `POST` | `/api/v1/dcr/:id/approve` | `dcr.review` |
| `POST` | `/api/v1/dcr/:id/reject` | `dcr.review` |

## 9. Distribution Connect APIs

`backend/internal/distribution/handler.go:21-48`

| Method | Path | Permission |
| --- | --- | --- |
| `GET` | `/api/v1/distribution-links` | `distributor.view` |
| `POST` | `/api/v1/distribution-links` | `link.manage` |
| `PATCH` | `/api/v1/distribution-links/:id/status` | `link.manage` |
| `GET` | `/api/v1/distributors` | `distributor.view` |
| `POST` | `/api/v1/distributors` | `distributor.manage` |
| `GET` | `/api/v1/distributors/:id` | `distributor.view` |
| `GET` | `/api/v1/orders` | `order.view` |
| `POST` | `/api/v1/orders` | `order.manage` |
| `GET` | `/api/v1/orders/:id` | `order.view` |
| `POST` | `/api/v1/orders/:id/submit` | `order.manage` |
| `POST` | `/api/v1/orders/:id/approve` | `order.approve` |
| `POST` | `/api/v1/orders/:id/cancel` | `order.manage` |
| `GET` | `/api/v1/inventory` | `inventory.view` |
| `GET` | `/api/v1/inventory/movements` | `inventory.view` |
| `POST` | `/api/v1/inventory/movements` | `inventory.manage` |
| `GET` | `/api/v1/secondary-sales` | `secondary_sales.view` |
| `POST` | `/api/v1/secondary-sales` | `secondary_sales.import` |

Order creation and other transactional writes use `Idempotency-Key: <uuid>` where required by the endpoint service. Treat inventory and approval state as server-authoritative. Cross-tenant distributor visibility depends on active company-distributor links; a suspended/ended link immediately removes visibility.

## 10. Reporting and audit APIs

`backend/internal/reporting/handler.go:20-30`

| Method | Path | Permission | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/reports/field-activity` | `reports.view` | Scope-aware dashboard; filters `from`, `to` |
| `GET` | `/api/v1/reports/distribution` | `reports.view` | Distribution dashboard; filters `from`, `to` |
| `GET` | `/api/v1/reports/audit` | `audit.view` | Immutable audit trail; filters `from`, `to`, `module`, `action`; paginated |
| `GET` | `/api/v1/reports/definitions` | `reports.view` | Saved report definitions |
| `POST` | `/api/v1/reports/definitions` | `report.manage` | Create definition |
| `GET` | `/api/v1/reports/snapshots` | `reports.view` | Report snapshots |
| `GET` | `/api/v1/reports/snapshots/:id` | `reports.view` | Get snapshot |
| `POST` | `/api/v1/reports/snapshots` | `report.manage` | Create snapshot |

## 11. Integration APIs

`backend/internal/integrations/handler.go:20-33`

| Method | Path | Permission |
| --- | --- | --- |
| `GET` | `/api/v1/integrations/connections` | `integration.view` |
| `POST` | `/api/v1/integrations/connections` | `integration.manage` |
| `PATCH` | `/api/v1/integrations/connections/:id/status` | `integration.manage` |
| `DELETE` | `/api/v1/integrations/connections/:id` | `integration.manage` |
| `POST` | `/api/v1/integrations/events` | `integration.manage` |
| `GET` | `/api/v1/integrations/deliveries` | `integration.view` |
| `POST` | `/api/v1/integrations/deliveries/:id/retry` | `webhook.retry` |

Connection secrets are referenced by environment variable name and are never returned. These endpoints are administrative; normal field mobile screens should not expose them.

## 12. Mobile sync APIs

`backend/internal/syncapi/handler.go:17-56`, `backend/internal/syncapi/model.go:10-57`

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/sync/changes` | Pull scope-aware changes |
| `POST` | `/api/v1/sync/operations` | Push durable outbox operations |
| `GET` | `/api/v1/sync/status` | Server time, cursor and queue hints |

### Pull changes

Query: `since` (opaque cursor), `entities` (comma-separated filter), `limit` (default `100`).

```ts
type Change = {
  entity: string;
  entity_id: string;
  op: string;
  updated_at: string;
  payload: Record<string, unknown>;
};

type ChangesResponse = {
  next_cursor: string;
  has_more: boolean;
  changes: Change[];
};
```

Supported `entities` values are the keys of `entitySpecs` in `backend/internal/syncapi/model.go:79-213`: `doctor`, `chemist`, `stockist`, `product_category`, `product`, `target`, `territory`, `employee`, `tour_plan`, `tour_plan_line`, `visit`, `dcr`, `dcr_product`, `distributor`, `distribution_link`, `order`, `order_item`, `inventory`, `inventory_movement`, `secondary_sale`.

### Push operations

```ts
type SyncOperation = {
  id: string;
  entity: string;
  entity_id: string;
  operation: string;
  payload: unknown;
  idempotency_key: string;
};

type OperationResult = {
  client_operation_id: string;
  status: 'APPLIED' | 'REPLAYED' | 'CONFLICT' | 'REJECTED';
  entity: string;
  entity_id?: string;
  error_code?: string;
  error_message?: string;
  retryable: boolean;
};

type PushResponse = { operations: OperationResult[] };
```

Supported operations: `visit.create`, `visit.submit`, `visit.attachment`, `dcr.create`, `dcr.product.add`, `dcr.submit`, `dcr.approve`, `dcr.reject`, `tour.create`, `tour.line.add`, `tour.submit`, `tour.approve`, `tour.reject`, `order.create`, `order.submit`, `order.approve`, `secondary_sale.create`, `inventory.movement`.

### Status

```ts
type SyncStatus = {
  server_time: string;
  latest_cursor: string;
  pending_operations: number;
  supported_entities: string[];
  entity_counts: Record<string, number>;
};
```

The existing offline design and conflict policy remain in `docs/mobile-offline-sync.md`.

## 13. Mobile screen-to-API mapping

| Planned mobile area | Existing APIs | Status |
| --- | --- | --- |
| Login / session bootstrap | `/auth/login`, `/auth/refresh`, `/auth/me`, `/auth/logout` | Available |
| Forgot/reset password | `/auth/forgot-password`, `/auth/reset-password` | Available |
| Home dashboard | `/reports/field-activity`, `/reports/distribution`, `/sync/status` | Available; use cached/summary views for field users |
| Profile / permissions | `/auth/me`, `/users/:id`, `/auth/change-password` | Available |
| Visits | `/visits`, `/sync/changes`, `/sync/operations` | Available |
| Tour plans | `/tour-plans`, `/sync/changes`, `/sync/operations` | Available |
| DCR | `/dcr`, `/sync/changes`, `/sync/operations` | Available |
| Doctors/chemists/stockists/products | corresponding master-data endpoints plus sync read models | Available |
| Territories/divisions/employees | organization APIs plus sync read models | Available |
| Orders/distributors/inventory/secondary sales | distribution APIs plus sync read models | Available |
| Attachments | `/visits/:id/attachments` metadata only | Binary upload is `MISSING` |
| Notifications | none | `MISSING` |
| Tenant switch endpoint | none | `MISSING`; login with new tenant code is the supported flow |
| Global search | none | `MISSING`; use list endpoints/search filters locally |
| Comments/chat/feed | none | `MISSING` |
| Profile picture upload | none | `MISSING` |
| Barcode/QR lookup endpoint | none | `MISSING`; device scanning can still feed existing product lookup |
| Client-created users/roles/tenants for field users | endpoints exist but permission-gated to admins | Available only for authorized roles |
| Push notification device token registration | none | `MISSING` |
| File/object storage upload | none | `MISSING` |
| Offline conflict resolution API | no dedicated endpoint; use `/sync/operations` statuses | Available via sync contract |

## 14. Implementation rules for the redesign

1. Treat `/auth/login` as the only way to select a tenant. Do not send a tenant header.
2. Store tokens and user profile in Expo SecureStore; purge tenant SQLite data when the authenticated tenant changes.
3. Use `data.permissions`, `data.roles`, `data.data_scope`, `data.division_ids`, and `data.territory_ids` from `/auth/me` for navigation and UI affordances.
4. On refresh failure or logout, clear local session state and redirect to login.
5. Use the standard envelope parser for all endpoints; do not read `data` without checking `success`.
6. Use `meta` for list pagination. Never assume a fixed page size.
7. Use `Idempotency-Key: <uuid>` for direct visit/distribution writes and a stable idempotency key in queued sync operations.
8. Prefer sync read models for offline field screens; use direct REST reads only for online-only/admin flows.
9. Mark unsupported product requirements as `MISSING` in the UI/design rather than wiring them to fabricated endpoints.
10. Re-check `openapi.yaml` before adding a new endpoint; the backend spec version is part of release verification.
