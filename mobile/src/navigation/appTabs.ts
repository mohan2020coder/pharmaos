import { can, canAny } from '../lib/permissions';
import type { SessionUser } from '../types';

export type AppTab = {
  name: string;
  title: string;
  short: string;
  permissions?: string[];
  anyPermission?: boolean;
};

export const appTabs: AppTab[] = [
  { name: 'home', title: 'Home', short: 'HM' },
  { name: 'visits', title: 'Visits', short: 'VS', permissions: ['visit.view'] },
  { name: 'tours', title: 'Tours', short: 'TR', permissions: ['tourplan.view'] },
  { name: 'dcr', title: 'DCR', short: 'DC', permissions: ['dcr.view'] },
  {
    name: 'customers',
    title: 'Catalog',
    short: 'CT',
    permissions: ['doctor.view', 'chemist.view', 'stockist.view', 'product.view'],
    anyPermission: true
  },
  { name: 'orders', title: 'Orders', short: 'OR', permissions: ['order.view'] },
  { name: 'profile', title: 'Profile', short: 'PF' }
];

export function visibleTabs(user: SessionUser | null): AppTab[] {
  return appTabs.filter((tab) => {
    if (!tab.permissions || tab.permissions.length === 0) {
      return true;
    }
    return tab.anyPermission
      ? canAny(user, tab.permissions)
      : tab.permissions.some((permission) => can(user, permission));
  });
}
