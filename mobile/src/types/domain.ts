export type PaginationMeta = {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
};

export type Employee = {
  id: string;
  tenant_id: string;
  employee_code?: string;
  user_id?: string;
  first_name: string;
  last_name: string;
  email?: string;
  status: string;
  division_id?: string;
  region_id?: string;
  territory_id?: string;
  manager_employee_id?: string;
  position_id?: string;
};

export type VisitStatus = 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'REJECTED';
export type EntityType = 'DOCTOR' | 'CHEMIST' | 'STOCKIST';

export type Visit = {
  id: string;
  tenant_id: string;
  employee_id: string;
  tour_plan_line_id?: string;
  territory_id?: string;
  entity_type: EntityType;
  entity_id: string;
  client_reference_id?: string;
  visited_at: string;
  check_in_lat?: number;
  check_in_lng?: number;
  check_in_accuracy?: number;
  check_out_lat?: number;
  check_out_lng?: number;
  status: VisitStatus;
  remarks?: string;
  submitted_at?: string;
  created_at: string;
  updated_at: string;
};

export type CreateVisitInput = {
  id?: string;
  employee_id: string;
  tour_plan_line_id?: string;
  territory_id?: string;
  entity_type: EntityType;
  entity_id: string;
  client_reference_id?: string;
  visited_at?: string;
  check_in_lat?: number;
  check_in_lng?: number;
  check_in_accuracy?: number;
  remarks?: string;
};

export type TourPlanStatus = 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'REJECTED';

export type TourPlan = {
  id: string;
  tenant_id: string;
  employee_id: string;
  title: string;
  period_start: string;
  period_end: string;
  status: TourPlanStatus;
  submitted_at?: string;
  approved_by?: string;
  approved_at?: string;
  comments?: string;
  created_at: string;
  updated_at: string;
};

export type TourPlanLineStatus = 'PLANNED' | 'COMPLETED' | 'SKIPPED';

export type TourPlanLine = {
  id: string;
  tenant_id: string;
  tour_plan_id: string;
  planned_date: string;
  territory_id?: string;
  entity_type: EntityType;
  entity_id: string;
  purpose?: string;
  sequence: number;
  status: TourPlanLineStatus;
  created_at: string;
  updated_at: string;
};

export type CreateTourPlanInput = {
  id?: string;
  employee_id: string;
  title: string;
  period_start: string;
  period_end: string;
};

export type CreateTourPlanLineInput = {
  id?: string;
  planned_date: string;
  territory_id?: string;
  entity_type: EntityType;
  entity_id: string;
  purpose?: string;
  sequence: number;
};

export type DcrStatus = 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'REJECTED';

export type DailyCallReport = {
  id: string;
  tenant_id: string;
  employee_id: string;
  report_date: string;
  total_visits: number;
  status: DcrStatus;
  submitted_at?: string;
  approved_by?: string;
  approved_at?: string;
  remarks?: string;
  created_at: string;
  updated_at: string;
};

export type ProductDetail = {
  id: string;
  tenant_id: string;
  dcr_id: string;
  product_id: string;
  quantity: number;
  sample_count: number;
  outcome?: string;
  remarks?: string;
  created_at: string;
  updated_at: string;
};

export type PartyBase = {
  id: string;
  tenant_id: string;
  name: string;
  external_id?: string;
  email?: string;
  phone?: string;
  address?: string;
  territory_id?: string;
  division_id?: string;
  is_active: boolean;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type Doctor = PartyBase & {
  specialty?: string;
  qualification?: string;
  registration_no?: string;
};

export type Chemist = PartyBase & {
  owner_name?: string;
  license_no?: string;
};

export type Stockist = PartyBase & {
  license_no?: string;
  gst_number?: string;
  credit_limit: number;
  payment_terms_days: number;
};

export type Product = {
  id: string;
  tenant_id: string;
  category_id?: string;
  code: string;
  name: string;
  brand?: string;
  manufacturer?: string;
  composition?: string;
  pack_size?: string;
  unit?: string;
  mrp: number;
  is_active: boolean;
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type DistributionLink = {
  id: string;
  tenant_id: string;
  distributor_tenant_id: string;
  status: 'ACTIVE' | 'SUSPENDED' | 'ENDED';
  effective_from: string;
  effective_to?: string;
  credit_limit: number;
  payment_terms_days: number;
  created_by?: string;
  created_at: string;
  updated_at: string;
};

export type Distributor = {
  id: string;
  tenant_id: string;
  code: string;
  name: string;
  license_no?: string;
  email?: string;
  phone?: string;
  address?: string;
  status: 'ACTIVE' | 'INACTIVE';
  settings?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type OrderStatus = 'DRAFT' | 'SUBMITTED' | 'APPROVED' | 'FULFILLED' | 'CANCELLED';

export type OrderItem = {
  id: string;
  tenant_id: string;
  order_id: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  line_total: number;
  created_at: string;
  updated_at: string;
};

export type Order = {
  id: string;
  tenant_id: string;
  distributor_tenant_id: string;
  link_id: string;
  order_number: string;
  status: OrderStatus;
  currency_code: string;
  total_amount: number;
  idempotency_key: string;
  requested_by?: string;
  approved_by?: string;
  order_date: string;
  required_by?: string;
  submitted_at?: string;
  approved_at?: string;
  cancelled_at?: string;
  remarks?: string;
  created_at: string;
  updated_at: string;
};

export type OrderDetail = Order & { items: OrderItem[] };

export type CreateOrderItemInput = {
  id?: string;
  product_id: string;
  quantity: number;
  unit_price?: number;
};

export type CreateOrderInput = {
  id?: string;
  distributor_tenant_id: string;
  order_number?: string;
  currency_code?: string;
  order_date?: string;
  required_by?: string;
  remarks?: string;
  items: CreateOrderItemInput[];
};

export type Inventory = {
  id: string;
  tenant_id: string;
  product_id: string;
  warehouse_code: string;
  quantity_on_hand: number;
  quantity_reserved: number;
  version: number;
  created_at: string;
  updated_at: string;
};

export type InventoryMovement = {
  id: string;
  tenant_id: string;
  product_id: string;
  warehouse_code: string;
  movement_type: 'RECEIPT' | 'SALE' | 'ADJUSTMENT';
  quantity_delta: number;
  reference_type?: string;
  reference_id?: string;
  idempotency_key: string;
  created_by?: string;
  occurred_at: string;
  created_at: string;
};

export type SecondarySale = {
  id: string;
  tenant_id: string;
  order_id?: string;
  company_tenant_id?: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  total_amount: number;
  sale_date: string;
  invoice_no?: string;
  customer_code?: string;
  idempotency_key: string;
  created_by?: string;
  created_at: string;
  updated_at: string;
};

export type FieldActivityReport = {
  period: { from: string; to: string };
  scope: string;
  visits: {
    total: number;
    submitted: number;
    draft: number;
    unique_customers: number;
  };
  daily_call_reports: {
    total: number;
    submitted: number;
    approved: number;
  };
};

export type DistributionSummaryReport = {
  period: { from: string; to: string };
  orders: {
    total: number;
    submitted: number;
    approved: number;
    cancelled: number;
    value: number;
  };
  inventory: {
    sku_count: number;
    on_hand: number;
    reserved: number;
  };
  secondary_sales: {
    records: number;
    units: number;
    value: number;
  };
  active_links: number;
};
