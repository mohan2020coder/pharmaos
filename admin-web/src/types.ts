export type JsonRecord = Record<string, unknown>
export type StringMap = Record<string, string>

export interface PageMeta {
  page: number
  page_size: number
  total: number
  total_pages: number
}

export interface ApiResult<T> {
  data: T
  meta?: PageMeta
}

export interface TokenPair {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  refresh_expires_in: number
}

export interface AuthMe {
  id: string
  tenant_id: string
  email: string
  roles: string[]
  permissions: string[]
  data_scope: string
  is_privileged: boolean
  division_ids: string[]
  territory_ids: string[]
}

export interface LoginUser {
  id: string
  tenant_id: string
  email: string
  roles: string[]
  permissions: string[]
  data_scope: string
}

export interface LoginResponse {
  tokens: TokenPair
  user: LoginUser
}

export interface Tenant {
  id: string
  code: string
  name: string
  tenant_type: string
  status: string
  is_platform: boolean
  country_code?: string
  timezone: string
  currency_code: string
  settings: JsonRecord
  created_at: string
  updated_at: string
}

export interface User {
  id: string
  tenant_id: string
  email: string
  first_name: string
  last_name: string
  phone?: string
  status: string
  email_verified: boolean
  must_change_password: boolean
  locked_until?: string
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface Role {
  id: string
  tenant_id?: string
  code: string
  name: string
  description: string
  is_system: boolean
  is_privileged: boolean
  data_scope: string
  created_at: string
  updated_at: string
}

export interface RoleWithPermissions extends Role {
  permissions: string[]
}

export interface Permission {
  id: string
  code: string
  module: string
  description: string
  created_at: string
}

export interface RoleAssignment {
  user_id: string
  role_id: string
  scope_type: string
  scope_id?: string
  valid_from: string
  valid_to?: string
}

export interface Division {
  id: string
  tenant_id: string
  code: string
  name: string
  description: string
  status: string
  manager_user_id?: string
  settings: JsonRecord
  created_at: string
  updated_at: string
}

export interface Region {
  id: string
  tenant_id: string
  division_id: string
  code: string
  name: string
  status: string
  created_at: string
  updated_at: string
}

export interface Territory {
  id: string
  tenant_id: string
  division_id: string
  region_id: string
  branch_id?: string
  code: string
  name: string
  city: string
  status: string
  created_at: string
  updated_at: string
}

export interface Branch {
  id: string
  tenant_id: string
  division_id?: string
  code: string
  name: string
  address: string
  city: string
  state: string
  postal_code: string
  country_code?: string
  status: string
  created_at: string
  updated_at: string
}

export interface Employee {
  id: string
  tenant_id: string
  employee_code: string
  user_id?: string
  first_name: string
  last_name: string
  email?: string
  phone?: string
  status: string
  joining_date?: string
  division_id?: string
  region_id?: string
  territory_id?: string
  manager_employee_id?: string
  position_id?: string
  created_at: string
  updated_at: string
}

export interface Position {
  id: string
  tenant_id: string
  code: string
  title: string
  level: number
  is_management: boolean
  parent_position_id?: string
  created_at: string
  updated_at: string
}

export interface Doctor {
  id: string
  tenant_id: string
  external_id?: string
  name: string
  specialty?: string
  qualification?: string
  registration_no?: string
  email?: string
  phone?: string
  address?: string
  territory_id?: string
  division_id?: string
  is_active: boolean
  settings: JsonRecord
  created_at: string
  updated_at: string
}

export interface Chemist {
  id: string
  tenant_id: string
  external_id?: string
  name: string
  owner_name?: string
  license_no?: string
  email?: string
  phone?: string
  address?: string
  territory_id?: string
  division_id?: string
  is_active: boolean
  settings: JsonRecord
  created_at: string
  updated_at: string
}

export interface Stockist extends Chemist {
  gst_number?: string
  credit_limit: number
  payment_terms_days: number
}

export interface ProductCategory {
  id: string
  tenant_id: string
  code: string
  name: string
  parent_category_id?: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface Product {
  id: string
  tenant_id: string
  category_id?: string
  code: string
  name: string
  brand?: string
  manufacturer?: string
  composition?: string
  pack_size?: string
  unit?: string
  mrp: number
  is_active: boolean
  settings: JsonRecord
  created_at: string
  updated_at: string
}

export interface Target {
  id: string
  tenant_id: string
  entity_type: string
  entity_id: string
  employee_id: string
  territory_id?: string
  period_start: string
  period_end: string
  target_visits: number
  target_sales: number
  status: string
  created_at: string
  updated_at: string
}

export interface DistributionLink {
  id: string
  tenant_id: string
  distributor_tenant_id: string
  status: string
  effective_from: string
  effective_to?: string
  credit_limit: number
  payment_terms_days: number
  created_by?: string
  created_at: string
  updated_at: string
}

export interface Distributor {
  id: string
  tenant_id: string
  code: string
  name: string
  license_no?: string
  email?: string
  phone?: string
  address?: string
  status: string
  settings?: JsonRecord
  created_at: string
  updated_at: string
}

export interface OrderItem {
  id: string
  tenant_id: string
  order_id: string
  product_id: string
  quantity: number
  unit_price: number
  line_total: number
  created_at: string
  updated_at: string
}

export interface Order {
  id: string
  tenant_id: string
  distributor_tenant_id: string
  link_id: string
  order_number: string
  status: string
  currency_code: string
  total_amount: number
  idempotency_key: string
  requested_by?: string
  approved_by?: string
  order_date: string
  required_by?: string
  submitted_at?: string
  approved_at?: string
  cancelled_at?: string
  remarks?: string
  created_at: string
  updated_at: string
}

export interface OrderDetail extends Order {
  items: OrderItem[]
}

export interface Inventory {
  id: string
  tenant_id: string
  product_id: string
  warehouse_code: string
  quantity_on_hand: number
  quantity_reserved: number
  version: number
  created_at: string
  updated_at: string
}

export interface InventoryMovement {
  id: string
  tenant_id: string
  product_id: string
  warehouse_code: string
  movement_type: string
  quantity_delta: number
  reference_type?: string
  reference_id?: string
  idempotency_key: string
  created_by?: string
  occurred_at: string
  created_at: string
}

export interface SecondarySale {
  id: string
  tenant_id: string
  order_id?: string
  company_tenant_id?: string
  product_id: string
  quantity: number
  unit_price: number
  total_amount: number
  sale_date: string
  invoice_no?: string
  customer_code?: string
  idempotency_key: string
  created_by?: string
  created_at: string
  updated_at: string
}

export interface FieldActivityReport {
  period: { from: string; to: string }
  scope: string
  visits: { total: number; submitted: number; draft: number; unique_customers: number }
  daily_call_reports: { total: number; submitted: number; approved: number }
}

export interface DistributionReport {
  period: { from: string; to: string }
  orders: { total: number; submitted: number; approved: number; cancelled: number; value: number }
  inventory: { sku_count: number; on_hand: number; reserved: number }
  secondary_sales: { records: number; units: number; value: number }
  active_links: number
}

export interface AuditEntry {
  id: number
  tenant_id?: string
  user_id?: string
  action: string
  module: string
  entity_type: string
  entity_id?: string
  created_at: string
}

export interface ReportDefinition {
  id: string
  tenant_id: string
  code: string
  name: string
  description?: string
  report_type: string
  config: JsonRecord
  created_by?: string
  created_at: string
  updated_at: string
}

export interface ReportSnapshot {
  id: string
  tenant_id: string
  definition_id?: string
  period_from?: string
  period_to?: string
  payload: JsonRecord
  created_by?: string
  created_at: string
}

export interface IntegrationConnection {
  id: string
  tenant_id: string
  name: string
  kind: string
  endpoint_url?: string
  secret_ref?: string
  headers: StringMap
  status: string
  created_by?: string
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: string
  tenant_id: string
  connection_id: string
  event_type: string
  event_id: string
  payload: JsonRecord
  status: string
  attempts: number
  next_attempt_at: string
  last_error?: string
  response_status?: number
  delivered_at?: string
  created_at: string
  updated_at: string
}

export interface SyncChange {
  entity: string
  entity_id: string
  op: string
  updated_at: string
  payload: JsonRecord
}

export interface SyncStatus {
  server_time: string
  latest_cursor: string
  pending_operations: number
  supported_entities: string[]
  entity_counts: Record<string, number>
}
