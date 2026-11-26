import type { SessionUser } from '../types';

export const appPermissions = {
  visitView: 'visit.view',
  visitCreate: 'visit.create',
  tourView: 'tourplan.view',
  tourCreate: 'tourplan.create',
  tourApprove: 'tourplan.approve',
  dcrView: 'dcr.view',
  dcrCreate: 'dcr.create',
  dcrReview: 'dcr.review',
  doctorView: 'doctor.view',
  chemistView: 'chemist.view',
  stockistView: 'stockist.view',
  productView: 'product.view',
  orderView: 'order.view',
  orderManage: 'order.manage',
  orderApprove: 'order.approve',
  distributorView: 'distributor.view',
  inventoryView: 'inventory.view',
  secondarySalesView: 'secondary_sales.view',
  reportsView: 'reports.view',
  auditView: 'audit.view'
} as const;

export function can(user: SessionUser | null, permission: string): boolean {
  if (!user) {
    return false;
  }
  if (user.is_privileged || user.roles.includes('SUPER_ADMIN')) {
    return true;
  }
  return user.permissions.includes(permission);
}

export function canAny(user: SessionUser | null, permissions: string[]): boolean {
  return permissions.some((permission) => can(user, permission));
}
