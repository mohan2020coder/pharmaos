# Permissions and Data Scopes

> **Status:** Roles, permissions, claim population, route guards, hierarchy-backed scopes, distribution link scopes, and Phase 8 reporting/integration permissions are implemented. Permission codes use the `resource.action` form shown below; the older colon-style names in this document are illustrative only.

## 1. Authorization model

Authorization is the conjunction of three independent checks:

```text
authenticated session
    AND permission (resource:action)
    AND data scope (which rows of that resource)
    AND tenant/RLS predicate (which tenant owns the rows)
```

`Principal` (`backend/internal/auth/principal.go`) contains: `UserID`, `TenantID`, `SessionID`, `Email`, `Roles`, `Permissions map[string]bool`, `DataScope`, `DivisionIDs`, `TerritoryIDs`, `IsPrivileged`. `HasPermission` returns true for any privileged principal; `HasRole` performs an exact role match.

## 2. Implemented RBAC schema

| Table | Key columns | Purpose |
| --- | --- | --- |
| `roles` | `id`, `tenant_id`, `code`, `name`, `is_system` | Tenant-scoped roles plus global system roles. |
| `permissions` | `code`, `description`, `module` | Global catalog of `resource:action` codes. |
| `role_permissions` | `role_id`, `permission_id` | Grants. |
| `user_roles` | `user_id`, `role_id`, `tenant_id`, `valid_from`, `valid_to` | Time-bounded assignments. |
| `role_scopes` | `role_id`, `scope`, `division_ids`, `territory_ids` | Optional scope overrides. |

Permissions are stable strings such as `doctor.view`, `doctor.create`, `visit.view`, `visit.create`, `dcr.review`, `order.approve`, `inventory.view`, `reports.view`, `audit.view`, `link.manage`, `report.manage`, `integration.manage`, `integration.view`, `webhook.retry`, and `user.manage`. Wildcards are not evaluated at runtime; privileges are expanded into the principal from role grants.

## 3. Role catalog (typical defaults)

| Role | Tenant | Default data scope | Notes |
| --- | --- | --- | --- |
| Platform Admin | Global | TENANT (explicit selection) | `IsPrivileged=true`; break-glass only, fully audited; no normal-tenant token gets this flag. |
| Tenant Admin | Either | TENANT | Full tenant configuration; cannot bypass audit or cross-tenant link rules. |
| NSM | PHARMA_COMPANY | DIVISION (or TENANT when configured) | National head; oversees all RSMs. |
| RSM | PHARMA_COMPANY | REGION | Oversees ASMs in assigned regions. |
| ASM | PHARMA_COMPANY | TERRITORY | Oversees TBMs/MRs in assigned areas. |
| TBM | PHARMA_COMPANY | TEAM | Territory business manager; manages MRs. |
| MR | PHARMA_COMPANY | SELF | Field representative; owns visits, DCR and tour plan. |
| Distributor Manager | DISTRIBUTOR | TENANT (own tenant) | Manages stock, orders and secondary sales; company data only via links. |
| Distributor User | DISTRIBUTOR | SELF/TEAM | Order entry and inventory tasks. |
| Auditor / Read-only | Either | TENANT | `*:read` and `audit:read`; no mutations. |

Hierarchy: **MR → TBM → ASM → RSM → NSM**. Each level reports to the next; the hierarchy table (`employee_hierarchy`, Phase 3) is the source for `TEAM` expansion, not role names hard-coded in queries.

## 4. Data scope resolution

| Scope | Row predicate | Typical use |
| --- | --- | --- |
| `SELF` | `owner_user_id = principal.UserID` | MR visits, DCR, targets |
| `TEAM` | `owner_user_id IN (descendants of principal in employee_hierarchy)` | TBM reviewing MR activity |
| `TERRITORY` | `territory_id IN principal.TerritoryIDs` (plus descendants) | ASM territory rolls-up |
| `REGION` | `region_id IN principal.RegionIDs` or territories under assigned regions | RSM |
| `DIVISION` | `division_id IN principal.DivisionIDs` | NSM |
| `TENANT` | `tenant_id = principal.TenantID` only | Tenant admin, audit |

`Principal` already stores `DivisionIDs` and `TerritoryIDs`; `RegionIDs` and the materialized descendant set are Phase 3 additions. Scope predicates are always intersected with the tenant predicate, never substituted for it.

## 5. Enforcement points

| Layer | Responsibility |
| --- | --- |
| Edge/middleware | Validate JWT, build principal, attach to context; reject unauthenticated requests. |
| Handler | Require permission through `middleware.RequirePermission("visit.create")` before decoding expensive bodies. |
| Service | Apply workflow state checks and compute allowable scope IDs from the hierarchy. |
| Repository | Add `tenant_id` plus scope predicate to every SQL statement. |
| RLS | Reject rows outside the transaction-local tenant (defense in depth). |
| Admin web/mobile UI | Hide unavailable actions for usability only; never trusted as enforcement. |
| Audit log | Record allow/deny decisions for sensitive actions with principal, tenant, scope and resource ID. |

Example service shape:

```go
p := auth.MustPrincipal(ctx)
if !p.HasPermission("visit.view") {
    return nil, apperrors.ErrForbidden
}
q := tx.Where("tenant_id = ?", p.TenantID)
switch p.DataScope {
case "SELF":
    q = q.Where("owner_user_id = ?", p.UserID)
case "TEAM":
    q = q.Where("owner_user_id IN ?", teamIDs)
case "TERRITORY":
    q = q.Where("territory_id IN ?", p.TerritoryIDs)
case "DIVISION":
    q = q.Where("division_id IN ?", p.DivisionIDs)
case "TENANT":
    // tenant predicate is sufficient
default:
    return nil, apperrors.ErrForbidden
}
```

## 6. Token lifecycle implications

- Access tokens are short-lived (`ACCESS_TOKEN_TTL`, default 15m) and carry permissions/scope snapshots; refresh tokens are longer-lived (`REFRESH_TOKEN_TTL`, default 720h).
- Role, hierarchy or territory changes take effect on refresh/logout; `token_version` is tracked on users and included in claims so administrators can force re-authentication.
- Refresh token rotation is implemented with persisted sessions and family revocation on reuse; a revoked session invalidates all derived access tokens when the API checks session state. Shared Redis revocation remains follow-on work; database-backed session state is the current source of truth.
- Any change to `DataScope` or role membership must be audited.

## 7. Special cases

- **Company–distributor links:** a company-side permission plus an active link is required; a distributor-side permission alone is insufficient for company CRM data.
- **Secondary sales:** writable by distributor users; company users may read only linked records for reporting.
- **Audit log:** append-only; a database trigger rejects updates and deletes. `audit.view` is separately granted.
- **Targets:** writable by ASM/RSM/NSM within scope, readable by MR for their own territory.
- **Products/catalog:** centrally managed; field users get read-only access.

## 8. Verification

Phase 2/3 acceptance tests must cover: permission absent → 403; scope narrowing/widening attempts; role change after token issue; MR cannot see another MR's visits; TBM sees only descendants; ASM sees only assigned territories; distributor cannot read another distributor or company CRM; privileged flag cannot be self-assigned. Tests should assert both API responses and RLS behavior.

Related documents: [tenant-isolation.md](tenant-isolation.md), [api-design.md](api-design.md), [implementation-roadmap.md](implementation-roadmap.md).
