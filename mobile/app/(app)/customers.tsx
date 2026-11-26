import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { StyleSheet, View } from 'react-native';

import { listChemists, listDoctors, listProducts, listStockists } from '../../src/api/domain';
import { PermissionGate } from '../../src/components/auth/PermissionGate';
import {
  AppText,
  Card,
  DetailRow,
  EmptyState,
  ErrorState,
  LoadingState,
  Screen,
  Segmented,
  TextField
} from '../../src/components/ui';
import { formatMoney } from '../../src/lib/format';
import { appPermissions, can } from '../../src/lib/permissions';
import { queryKeys } from '../../src/lib/queryKeys';
import { useAuthStore } from '../../src/stores/auth';
import { spacing } from '../../src/theme/tokens';
import type { EntityType } from '../../src/types';

type CatalogTab = EntityType | 'PRODUCT';

export default function CustomersScreen() {
  const user = useAuthStore((state) => state.user);
  const [tab, setTab] = useState<CatalogTab>('DOCTOR');
  const [search, setSearch] = useState('');

  const doctors = useQuery({
    queryKey: queryKeys.doctors(search),
    queryFn: async () => (await listDoctors(search)).data,
    enabled: can(user, appPermissions.doctorView),
    staleTime: 30000
  });
  const chemists = useQuery({
    queryKey: queryKeys.chemists(search),
    queryFn: async () => (await listChemists(search)).data,
    enabled: can(user, appPermissions.chemistView),
    staleTime: 30000
  });
  const stockists = useQuery({
    queryKey: queryKeys.stockists(search),
    queryFn: async () => (await listStockists(search)).data,
    enabled: can(user, appPermissions.stockistView),
    staleTime: 30000
  });
  const products = useQuery({
    queryKey: queryKeys.products(search),
    queryFn: async () => (await listProducts(search)).data,
    enabled: can(user, appPermissions.productView),
    staleTime: 30000
  });

  const options = [
    can(user, appPermissions.doctorView) ? { label: 'Doctors', value: 'DOCTOR' as const } : null,
    can(user, appPermissions.chemistView) ? { label: 'Chemists', value: 'CHEMIST' as const } : null,
    can(user, appPermissions.stockistView) ? { label: 'Stockists', value: 'STOCKIST' as const } : null,
    can(user, appPermissions.productView) ? { label: 'Products', value: 'PRODUCT' as const } : null
  ].filter((option): option is { label: string; value: CatalogTab } => option !== null);

  const activeQuery = tab === 'DOCTOR' ? doctors : tab === 'CHEMIST' ? chemists : tab === 'STOCKIST' ? stockists : products;

  return (
    <PermissionGate
      anyPermission={[
        appPermissions.doctorView,
        appPermissions.chemistView,
        appPermissions.stockistView,
        appPermissions.productView
      ]}
    >
      <Screen scroll>
        <AppText variant="title" tone="primaryDark">
          Catalog
        </AppText>
        <AppText variant="body" tone="muted" style={styles.intro}>
          Live master data available to your role.
        </AppText>

        <View style={styles.section}>
          <Segmented
            value={options.some((option) => option.value === tab) ? tab : (options[0]?.value ?? 'DOCTOR')}
            onChange={(value) => setTab(value as CatalogTab)}
            options={options}
          />
        </View>

        <View style={styles.section}>
          <TextField label="Search" value={search} onChangeText={setSearch} placeholder="Name, code or registration" />
        </View>

        <View style={styles.section}>
          {activeQuery.isPending ? <LoadingState message="Loading records…" /> : null}
          {activeQuery.isError ? (
            <ErrorState
              message={
                activeQuery.error instanceof Error ? activeQuery.error.message : 'Unable to load catalog records'
              }
              onRetry={() => void activeQuery.refetch()}
            />
          ) : null}
          {activeQuery.data && activeQuery.data.length === 0 ? (
            <EmptyState title="No records" message="No catalog records matched this search." />
          ) : null}
          {activeQuery.data?.map((item) => (
            <Card key={item.id} style={styles.card}>
              <AppText variant="section" tone="primaryDark">
                {'name' in item ? item.name : 'Unnamed'}
              </AppText>
              {'code' in item ? (
                <DetailRow label="Code" value={<AppText variant="body">{item.code}</AppText>} />
              ) : null}
              {'specialty' in item && item.specialty ? (
                <DetailRow label="Specialty" value={<AppText variant="body">{item.specialty}</AppText>} />
              ) : null}
              {'owner_name' in item && item.owner_name ? (
                <DetailRow label="Owner" value={<AppText variant="body">{item.owner_name}</AppText>} />
              ) : null}
              {'gst_number' in item && item.gst_number ? (
                <DetailRow label="GST" value={<AppText variant="body">{item.gst_number}</AppText>} />
              ) : null}
              {'mrp' in item ? (
                <DetailRow label="MRP" value={<AppText variant="body">{formatMoney(item.mrp)}</AppText>} />
              ) : null}
              {'email' in item && item.email ? (
                <DetailRow label="Email" value={<AppText variant="body">{item.email}</AppText>} />
              ) : null}
              {'phone' in item && item.phone ? (
                <DetailRow label="Phone" value={<AppText variant="body">{item.phone}</AppText>} />
              ) : null}
            </Card>
          ))}
        </View>
      </Screen>
    </PermissionGate>
  );
}

const styles = StyleSheet.create({
  intro: {
    marginTop: spacing.sm
  },
  section: {
    marginTop: spacing.lg
  },
  card: {
    marginBottom: spacing.md
  }
});
