import { randomUUID } from 'expo-crypto';

import { get, getPage, post } from './client';
import type {
  Chemist,
  CreateOrderInput,
  CreateTourPlanInput,
  CreateTourPlanLineInput,
  CreateVisitInput,
  DailyCallReport,
  Distributor,
  DistributionLink,
  DistributionSummaryReport,
  Doctor,
  Employee,
  FieldActivityReport,
  Inventory,
  Order,
  OrderDetail,
  Product,
  ProductDetail,
  SecondarySale,
  Stockist,
  TourPlan,
  TourPlanLine,
  Visit
} from '../types';

type QueryParams = Record<string, string | number | undefined>;

function queryString(params: QueryParams): string {
  const pairs: string[] = [];
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      pairs.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
    }
  }
  return pairs.length > 0 ? `?${pairs.join('&')}` : '';
}

function idempotentHeaders(): Record<string, string> {
  return { 'Idempotency-Key': randomUUID() };
}

export function listEmployees(): Promise<{ data: Employee[]; meta?: { total: number } }> {
  return getPage<Employee[]>('/api/v1/employees?page_size=200');
}

export function getCurrentEmployee(): Promise<Employee> {
  return get<Employee>('/api/v1/employees/me');
}

export function listVisits(
  filters: QueryParams = {}
): Promise<{ data: Visit[]; meta?: { total: number } }> {
  return getPage<Visit[]>(`/api/v1/visits${queryString({ page_size: 100, ...filters })}`);
}

export function getVisit(id: string): Promise<Visit> {
  return get<Visit>(`/api/v1/visits/${id}`);
}

export function createVisit(input: CreateVisitInput): Promise<Visit> {
  return post<Visit>('/api/v1/visits', input, { headers: idempotentHeaders() });
}

export function submitVisit(id: string): Promise<Visit> {
  return post<Visit>(`/api/v1/visits/${id}/submit`);
}

export function listTourPlans(
  filters: QueryParams = {}
): Promise<{ data: TourPlan[]; meta?: { total: number } }> {
  return getPage<TourPlan[]>(`/api/v1/tour-plans${queryString({ page_size: 100, ...filters })}`);
}

export function getTourPlan(id: string): Promise<TourPlan> {
  return get<TourPlan>(`/api/v1/tour-plans/${id}`);
}

export function createTourPlan(input: CreateTourPlanInput): Promise<TourPlan> {
  return post<TourPlan>('/api/v1/tour-plans', input);
}

export function listTourLines(id: string): Promise<TourPlanLine[]> {
  return get<TourPlanLine[]>(`/api/v1/tour-plans/${id}/lines`);
}

export function createTourLine(id: string, input: CreateTourPlanLineInput): Promise<TourPlanLine> {
  return post<TourPlanLine>(`/api/v1/tour-plans/${id}/lines`, input);
}

export function submitTourPlan(id: string): Promise<TourPlan> {
  return post<TourPlan>(`/api/v1/tour-plans/${id}/submit`);
}

export function decideTourPlan(id: string, approve: boolean, comments?: string): Promise<TourPlan> {
  return post<TourPlan>(`/api/v1/tour-plans/${id}/${approve ? 'approve' : 'reject'}`, { comments });
}

export function listDcr(
  filters: QueryParams = {}
): Promise<{ data: DailyCallReport[]; meta?: { total: number } }> {
  return getPage<DailyCallReport[]>(`/api/v1/dcr${queryString({ page_size: 100, ...filters })}`);
}

export function createDcr(employeeId: string, reportDate: string): Promise<DailyCallReport> {
  return post<DailyCallReport>('/api/v1/dcr', { employee_id: employeeId, report_date: reportDate });
}

export function getDcr(id: string): Promise<DailyCallReport> {
  return get<DailyCallReport>(`/api/v1/dcr/${id}`);
}

export function listDcrProducts(id: string): Promise<ProductDetail[]> {
  return get<ProductDetail[]>(`/api/v1/dcr/${id}/products`);
}

export function addDcrProduct(
  id: string,
  input: { product_id: string; quantity: number; sample_count: number; outcome?: string; remarks?: string }
): Promise<ProductDetail> {
  return post<ProductDetail>(`/api/v1/dcr/${id}/products`, input);
}

export function submitDcr(id: string): Promise<DailyCallReport> {
  return post<DailyCallReport>(`/api/v1/dcr/${id}/submit`);
}

export function decideDcr(id: string, approve: boolean, remarks?: string): Promise<DailyCallReport> {
  return post<DailyCallReport>(`/api/v1/dcr/${id}/${approve ? 'approve' : 'reject'}`, { remarks });
}

export function listDoctors(
  search = ''
): Promise<{ data: Doctor[]; meta?: { total: number } }> {
  return getPage<Doctor[]>(`/api/v1/doctors${queryString({ page_size: 100, search })}`);
}

export function listChemists(
  search = ''
): Promise<{ data: Chemist[]; meta?: { total: number } }> {
  return getPage<Chemist[]>(`/api/v1/chemists${queryString({ page_size: 100, search })}`);
}

export function listStockists(
  search = ''
): Promise<{ data: Stockist[]; meta?: { total: number } }> {
  return getPage<Stockist[]>(`/api/v1/stockists${queryString({ page_size: 100, search })}`);
}

export function listProducts(
  search = ''
): Promise<{ data: Product[]; meta?: { total: number } }> {
  return getPage<Product[]>(`/api/v1/products${queryString({ page_size: 100, search })}`);
}

export function listDistributionLinks(): Promise<{ data: DistributionLink[]; meta?: { total: number } }> {
  return getPage<DistributionLink[]>('/api/v1/distribution-links?page_size=100');
}

export function listDistributors(): Promise<{ data: Distributor[]; meta?: { total: number } }> {
  return getPage<Distributor[]>('/api/v1/distributors?page_size=100');
}

export function listOrders(
  filters: QueryParams = {}
): Promise<{ data: Order[]; meta?: { total: number } }> {
  return getPage<Order[]>(`/api/v1/orders${queryString({ page_size: 100, ...filters })}`);
}

export function getOrder(id: string): Promise<OrderDetail> {
  return get<OrderDetail>(`/api/v1/orders/${id}`);
}

export function createOrder(input: CreateOrderInput): Promise<OrderDetail> {
  return post<OrderDetail>('/api/v1/orders', input, { headers: idempotentHeaders() });
}

export function submitOrder(id: string): Promise<Order> {
  return post<Order>(`/api/v1/orders/${id}/submit`);
}

export function approveOrder(id: string, remarks?: string): Promise<Order> {
  return post<Order>(`/api/v1/orders/${id}/approve`, { remarks });
}

export function cancelOrder(id: string, remarks?: string): Promise<Order> {
  return post<Order>(`/api/v1/orders/${id}/cancel`, { remarks });
}

export function listInventory(
  filters: QueryParams = {}
): Promise<{ data: Inventory[]; meta?: { total: number } }> {
  return getPage<Inventory[]>(`/api/v1/inventory${queryString({ page_size: 100, ...filters })}`);
}

export function listSecondarySales(
  filters: QueryParams = {}
): Promise<{ data: SecondarySale[]; meta?: { total: number } }> {
  return getPage<SecondarySale[]>(
    `/api/v1/secondary-sales${queryString({ page_size: 100, ...filters })}`
  );
}

export function getFieldActivity(from: string, to: string): Promise<FieldActivityReport> {
  return get<FieldActivityReport>(`/api/v1/reports/field-activity${queryString({ from, to })}`);
}

export function getDistributionReport(from: string, to: string): Promise<DistributionSummaryReport> {
  return get<DistributionSummaryReport>(`/api/v1/reports/distribution${queryString({ from, to })}`);
}

export function getSessions(): Promise<unknown[]> {
  return get<unknown[]>('/api/v1/auth/sessions');
}
