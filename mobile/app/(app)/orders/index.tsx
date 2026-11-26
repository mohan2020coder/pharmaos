import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { listInventory, listOrders, listSecondarySales } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  DetailRow,
  EmptyState,
  ErrorState,
  LoadingState,
  Screen,
  Segmented,
  Tag
} from '../../../src/components/ui';
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatDate, formatMoney } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { radius, spacing } from '../../../src/theme/tokens';
import type { OrderStatus } from '../../../src/types';

type StatusFilter = 'ALL' | OrderStatus;

export default function OrdersIndexScreen() {
  const router = useRouter();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const orders = useQuery({
    queryKey: queryKeys.orders({}),
    queryFn: () => listOrders(),
    staleTime: 30000
  });
  const inventory = useQuery({
    queryKey: queryKeys.inventory({}),
    queryFn: () => listInventory(),
    enabled: can(user, appPermissions.inventoryView),
    staleTime: 30000
  });
  const secondarySales = useQuery({
    queryKey: ['secondary-sales'],
    queryFn: () => listSecondarySales(),
    enabled: can(user, appPermissions.secondarySalesView),
    staleTime: 30000
  });
  const [filter, setFilter] = useState<StatusFilter>('ALL');
  const visible = (orders.data?.data ?? []).filter((order) => filter === 'ALL' || order.status === filter);

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View>
          <AppText variant="caption" tone="primary">
            Distribution
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Orders
          </AppText>
          <AppText variant="body" tone="muted">
            {orders.data?.meta?.total ?? 0} orders
          </AppText>
        </View>
        {can(user, appPermissions.orderManage) ? (
          <Button label="New" onPress={() => router.push('/orders/new')} />
        ) : null}
      </View>

      <Segmented
        value={filter}
        onChange={setFilter}
        options={[
          { label: 'All', value: 'ALL' },
          { label: 'Draft', value: 'DRAFT' },
          { label: 'Submitted', value: 'SUBMITTED' },
          { label: 'Approved', value: 'APPROVED' },
          { label: 'Fulfilled', value: 'FULFILLED' }
        ]}
      />

      <View style={styles.section}>
        {orders.isPending ? <LoadingState message="Loading orders…" /> : null}
        {orders.isError ? (
          <ErrorState
            message={orders.error instanceof Error ? orders.error.message : 'Unable to load orders'}
            onRetry={() => void orders.refetch()}
          />
        ) : null}
        {!orders.isPending && !orders.isError && visible.length === 0 ? (
          <EmptyState title="No orders" message="No orders match the selected filter." />
        ) : null}
        {visible.map((order) => (
          <Pressable
            accessibilityRole="button"
            key={order.id}
            onPress={() => router.push(`/orders/${order.id}`)}
            style={({ pressed }) => [styles.card, pressed ? styles.cardPressed : null]}
          >
            <Card>
              <View style={styles.cardHeader}>
                <AppText variant="section" tone="primaryDark">
                  {order.order_number}
                </AppText>
                <Tag
                  label={order.status}
                  tone={order.status === 'APPROVED' || order.status === 'FULFILLED' ? 'success' : 'neutral'}
                />
              </View>
              <DetailRow label="Total" value={<AppText variant="body">{formatMoney(order.total_amount, order.currency_code)}</AppText>} />
              <DetailRow label="Order date" value={<AppText variant="body">{formatDate(order.order_date)}</AppText>} />
            </Card>
          </Pressable>
        ))}
      </View>

      {can(user, appPermissions.inventoryView) ? (
        <Card style={styles.section}>
          <AppText variant="heading">Inventory snapshot</AppText>
          {inventory.isPending ? <LoadingState message="Loading inventory…" /> : null}
          {inventory.isError ? (
            <ErrorState
              message={inventory.error instanceof Error ? inventory.error.message : 'Unable to load inventory'}
              onRetry={() => void inventory.refetch()}
            />
          ) : null}
          {inventory.data && inventory.data.data.length === 0 ? (
            <EmptyState title="No inventory" message="No inventory records are available." />
          ) : null}
          {inventory.data?.data.slice(0, 8).map((item) => (
            <DetailRow
              key={item.id}
              label={`${item.warehouse_code} · ${labelFor(item.product_id, 'PRODUCT')}`}
              value={<AppText variant="body">{item.quantity_on_hand - item.quantity_reserved}</AppText>}
            />
          ))}
        </Card>
      ) : null}

      {can(user, appPermissions.secondarySalesView) ? (
        <Card style={styles.section}>
          <AppText variant="heading">Secondary sales</AppText>
          {secondarySales.isPending ? <LoadingState message="Loading secondary sales…" /> : null}
          {secondarySales.isError ? (
            <ErrorState
              message={
                secondarySales.error instanceof Error
                  ? secondarySales.error.message
                  : 'Unable to load secondary sales'
              }
              onRetry={() => void secondarySales.refetch()}
            />
          ) : null}
          <DetailRow
            label="Records"
            value={<AppText variant="body">{secondarySales.data?.data.length ?? 0}</AppText>}
          />
          <DetailRow
            label="Value"
            value={
              <AppText variant="body">
                {formatMoney(
                  secondarySales.data?.data.reduce((sum, sale) => sum + sale.total_amount, 0) ?? 0
                )}
              </AppText>
            }
          />
        </Card>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: spacing.lg,
    marginBottom: spacing.lg
  },
  section: {
    gap: spacing.md,
    marginTop: spacing.lg
  },
  card: {
    borderRadius: radius.lg
  },
  cardPressed: {
    opacity: 0.75
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: spacing.md,
    marginBottom: spacing.sm
  }
});
