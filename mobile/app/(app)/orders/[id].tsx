import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useLocalSearchParams } from 'expo-router';
import { Alert, StyleSheet, View } from 'react-native';

import { approveOrder, cancelOrder, getOrder, submitOrder } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  DetailRow,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Tag
} from '../../../src/components/ui';
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatDate, formatMoney } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { colors, spacing } from '../../../src/theme/tokens';

export default function OrderDetailScreen() {
  const params = useLocalSearchParams<{ id: string }>();
  const id = String(params.id ?? '');
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const order = useQuery({
    queryKey: queryKeys.orderDetail(id),
    queryFn: () => getOrder(id),
    enabled: Boolean(id)
  });

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['orders'] });
    void order.refetch();
  }

  const action = useMutation({
    mutationFn: (operation: 'submit' | 'approve' | 'cancel') => {
      if (operation === 'submit') return submitOrder(id);
      if (operation === 'approve') return approveOrder(id);
      return cancelOrder(id);
    },
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to update order', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  return (
    <Screen scroll>
      {order.isPending ? <LoadingState message="Loading order…" /> : null}
      {order.isError ? (
        <ErrorState
          message={order.error instanceof Error ? order.error.message : 'Unable to load order'}
          onRetry={() => void order.refetch()}
        />
      ) : null}
      {order.data ? (
        <>
          <View style={styles.header}>
            <View>
              <AppText variant="caption" tone="primary">
                Distribution order
              </AppText>
              <AppText variant="title" tone="primaryDark">
                {order.data.order_number}
              </AppText>
            </View>
            <Tag
              label={order.data.status}
              tone={order.data.status === 'APPROVED' || order.data.status === 'FULFILLED' ? 'success' : 'neutral'}
            />
          </View>

          <Card style={styles.section}>
            <AppText variant="heading">Summary</AppText>
            <DetailRow
              label="Total"
              value={
                <AppText variant="section" tone="primaryDark">
                  {formatMoney(order.data.total_amount, order.data.currency_code)}
                </AppText>
              }
            />
            <DetailRow label="Order date" value={<AppText variant="body">{formatDate(order.data.order_date)}</AppText>} />
            <DetailRow
              label="Required by"
              value={<AppText variant="body">{formatDate(order.data.required_by)}</AppText>}
            />
            <DetailRow
              label="Distributor"
              value={<AppText variant="body">{labelFor(order.data.distributor_tenant_id, 'DISTRIBUTOR')}</AppText>}
            />
            {order.data.remarks ? (
              <DetailRow label="Remarks" value={<AppText variant="body">{order.data.remarks}</AppText>} />
            ) : null}
          </Card>

          <Card style={styles.section}>
            <AppText variant="heading">Items</AppText>
            {order.data.items.map((item) => (
              <View key={item.id} style={styles.itemRow}>
                <AppText variant="body">{labelFor(item.product_id, 'PRODUCT')}</AppText>
                <AppText variant="caption" tone="muted">
                  {item.quantity} × {formatMoney(item.unit_price, order.data.currency_code)} ={' '}
                  {formatMoney(item.line_total, order.data.currency_code)}
                </AppText>
              </View>
            ))}
          </Card>

          <View style={styles.actions}>
            {order.data.status === 'DRAFT' && can(user, appPermissions.orderManage) ? (
              <Button
                label={action.isPending ? 'Updating…' : 'Submit order'}
                onPress={() => action.mutate('submit')}
                loading={action.isPending}
              />
            ) : null}
            {order.data.status === 'SUBMITTED' ? (
              <>
                {can(user, appPermissions.orderApprove) ? (
                  <Button
                    label={action.isPending ? 'Updating…' : 'Approve order'}
                    onPress={() => action.mutate('approve')}
                    loading={action.isPending}
                  />
                ) : null}
                {can(user, appPermissions.orderManage) ? (
                  <SecondaryButton
                    label="Cancel order"
                    onPress={() => action.mutate('cancel')}
                    disabled={action.isPending}
                  />
                ) : null}
              </>
            ) : null}
          </View>
        </>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: spacing.md,
    marginBottom: spacing.lg
  },
  section: {
    marginBottom: spacing.lg
  },
  itemRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    gap: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.borderSoft,
    paddingVertical: spacing.md
  },
  actions: {
    gap: spacing.md,
    marginBottom: spacing.lg
  }
});
