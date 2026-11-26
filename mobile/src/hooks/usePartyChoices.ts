import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

import {
  listChemists,
  listDistributors,
  listDoctors,
  listEmployees,
  listProducts,
  listStockists
} from '../api/domain';
import type { Choice } from '../components/ui';
import { can } from '../lib/permissions';
import { queryKeys } from '../lib/queryKeys';
import { useAuthStore } from '../stores/auth';
import type { Employee, EntityType } from '../types';
import { useCurrentEmployee } from './useCurrentEmployee';

export function usePartyChoices(entityType: EntityType, search: string) {
  return useQuery<Choice[]>({
    queryKey: queryKeys.partyChoices(entityType, search),
    queryFn: async () => {
      if (entityType === 'DOCTOR') {
        const { data } = await listDoctors(search);
        return data.map((item) => ({ id: item.id, label: item.name, detail: item.specialty || item.territory_id }));
      }
      if (entityType === 'CHEMIST') {
        const { data } = await listChemists(search);
        return data.map((item) => ({ id: item.id, label: item.name, detail: item.owner_name || item.territory_id }));
      }
      const { data } = await listStockists(search);
      return data.map((item) => ({ id: item.id, label: item.name, detail: item.gst_number || item.territory_id }));
    },
    staleTime: 30000
  });
}

export type DisplayNameKind = EntityType | 'PRODUCT' | 'EMPLOYEE' | 'DISTRIBUTOR';

export function useEntityNames() {
  const user = useAuthStore((state) => state.user);
  const currentEmployee = useCurrentEmployee();
  const doctors = useQuery({
    queryKey: queryKeys.doctors(''),
    queryFn: async () => (await listDoctors('')).data,
    staleTime: 300000
  });
  const chemists = useQuery({
    queryKey: queryKeys.chemists(''),
    queryFn: async () => (await listChemists('')).data,
    staleTime: 300000
  });
  const stockists = useQuery({
    queryKey: queryKeys.stockists(''),
    queryFn: async () => (await listStockists('')).data,
    staleTime: 300000
  });
  const products = useQuery({
    queryKey: queryKeys.products(''),
    queryFn: async () => (await listProducts('')).data,
    staleTime: 300000
  });
  const distributors = useQuery({
    queryKey: queryKeys.distributors,
    queryFn: async () => (await listDistributors()).data,
    staleTime: 300000
  });
  const employees = useQuery<Employee[]>({
    queryKey: [...queryKeys.employees, 'list'],
    queryFn: async () => (await listEmployees()).data,
    enabled: can(user, 'employee.view'),
    staleTime: 300000
  });

  const names = useMemo(() => {
    const map = new Map<string, string>();
    doctors.data?.forEach((item) => map.set(item.id, item.name));
    chemists.data?.forEach((item) => map.set(item.id, item.name));
    stockists.data?.forEach((item) => map.set(item.id, item.name));
    products.data?.forEach((item) => map.set(item.id, item.name));
    distributors.data?.forEach((item) => map.set(item.id, item.name));
    employees.data?.forEach((item) => map.set(item.id, `${item.first_name} ${item.last_name}`.trim()));
    if (currentEmployee.data) {
      const employee = currentEmployee.data;
      map.set(employee.id, `${employee.first_name} ${employee.last_name}`.trim());
    }
    return map;
  }, [
    doctors.data,
    chemists.data,
    stockists.data,
    products.data,
    distributors.data,
    employees.data,
    currentEmployee.data
  ]);

  function labelFor(id?: string, kind?: DisplayNameKind): string {
    if (!id) {
      return '—';
    }
    const name = names.get(id);
    if (name) {
      return name;
    }
    const prefix = kind ? `${kind.toLowerCase()} ` : '';
    return `${prefix}${id.slice(0, 8)}`;
  }

  return { labelFor };
}
