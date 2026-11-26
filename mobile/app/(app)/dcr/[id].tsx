import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import {
  addDcrProduct,
  decideDcr,
  getDcr,
  listDcrProducts,
  listProducts,
  submitDcr
} from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  ChoiceList,
  DetailRow,
  EmptyState,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Tag,
  TextField
} from '../../../src/components/ui';
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatDate } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { colors, spacing } from '../../../src/theme/tokens';

export default function DcrDetailScreen() {
  const params = useLocalSearchParams<{ id: string }>();
  const id = String(params.id ?? '');
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const report = useQuery({
    queryKey: ['dcr', id],
    queryFn: () => getDcr(id),
    enabled: Boolean(id)
  });
  const products = useQuery({
    queryKey: queryKeys.dcrProducts(id),
    queryFn: () => listDcrProducts(id),
    enabled: Boolean(id)
  });
  const [search, setSearch] = useState('');
  const [productId, setProductId] = useState<string>();
  const [quantity, setQuantity] = useState('1');
  const [sampleCount, setSampleCount] = useState('0');
  const [outcome, setOutcome] = useState('');
  const catalog = useQuery({
    queryKey: queryKeys.products(search),
    queryFn: async () => (await listProducts(search)).data.map((item) => ({
      id: item.id,
      label: item.name,
      detail: `${item.code} · ${item.pack_size || item.unit || 'pack'}`
    })),
    staleTime: 30000
  });

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['dcr'] });
    void report.refetch();
    void products.refetch();
  }

  const addProduct = useMutation({
    mutationFn: () =>
      addDcrProduct(id, {
        product_id: productId as string,
        quantity: Number(quantity) || 0,
        sample_count: Number(sampleCount) || 0,
        outcome: outcome.trim() || undefined
      }),
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to add product', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  const submit = useMutation({
    mutationFn: submitDcr,
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to submit report', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  const decide = useMutation({
    mutationFn: (approve: boolean) => decideDcr(id, approve),
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to update report', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleAddProduct() {
    if (!productId) {
      Alert.alert('Product required', 'Select a product to add to this report.');
      return;
    }
    addProduct.mutate();
  }

  return (
    <Screen scroll>
      {report.isPending ? <LoadingState message="Loading report…" /> : null}
      {report.isError ? (
        <ErrorState
          message={report.error instanceof Error ? report.error.message : 'Unable to load report'}
          onRetry={() => void report.refetch()}
        />
      ) : null}
      {report.data ? (
        <>
          <View style={styles.header}>
            <View>
              <AppText variant="caption" tone="primary">
                Daily call report
              </AppText>
              <AppText variant="title" tone="primaryDark">
                {formatDate(report.data.report_date)}
              </AppText>
            </View>
            <Tag label={report.data.status} tone={report.data.status === 'APPROVED' ? 'success' : 'neutral'} />
          </View>

          <Card style={styles.section}>
            <AppText variant="heading">Summary</AppText>
            <DetailRow label="Visits" value={<AppText variant="body">{report.data.total_visits}</AppText>} />
            <DetailRow
              label="Employee"
              value={<AppText variant="body">{labelFor(report.data.employee_id, 'EMPLOYEE')}</AppText>}
            />
            {report.data.remarks ? (
              <DetailRow label="Remarks" value={<AppText variant="body">{report.data.remarks}</AppText>} />
            ) : null}
          </Card>

          <Card style={styles.section}>
            <AppText variant="heading">Products</AppText>
            {products.isPending ? <LoadingState message="Loading products…" /> : null}
            {products.isError ? (
              <ErrorState
                message={products.error instanceof Error ? products.error.message : 'Unable to load products'}
                onRetry={() => void products.refetch()}
              />
            ) : null}
            {products.data && products.data.length === 0 ? (
              <EmptyState title="No products" message="Add the first product outcome." />
            ) : null}
            {products.data?.map((item) => (
              <View key={item.id} style={styles.productRow}>
                <AppText variant="body">{labelFor(item.product_id, 'PRODUCT')}</AppText>
                <AppText variant="caption" tone="muted">
                  Qty {item.quantity} · Samples {item.sample_count}
                </AppText>
              </View>
            ))}
          </Card>

          {report.data.status === 'DRAFT' && can(user, appPermissions.dcrCreate) ? (
            <Card style={styles.section}>
              <AppText variant="heading">Add product</AppText>
              <View style={styles.spaced}>
                <TextField label="Search catalog" value={search} onChangeText={setSearch} />
              </View>
              {catalog.isPending ? <LoadingState message="Loading catalog…" /> : null}
              {catalog.data ? (
                <View style={styles.spaced}>
                  <ChoiceList choices={catalog.data} selectedId={productId} onSelect={setProductId} />
                </View>
              ) : null}
              <View style={styles.row}>
                <View style={styles.rowItem}>
                  <TextField label="Quantity" value={quantity} onChangeText={setQuantity} keyboardType="numeric" />
                </View>
                <View style={styles.rowItem}>
                  <TextField label="Samples" value={sampleCount} onChangeText={setSampleCount} keyboardType="numeric" />
                </View>
              </View>
              <View style={styles.spaced}>
                <TextField label="Outcome" value={outcome} onChangeText={setOutcome} />
              </View>
              <View style={styles.spaced}>
                <Button
                  label={addProduct.isPending ? 'Adding…' : 'Add product'}
                  onPress={handleAddProduct}
                  loading={addProduct.isPending}
                />
              </View>
            </Card>
          ) : null}

          <View style={styles.actions}>
            {report.data.status === 'DRAFT' && can(user, appPermissions.dcrCreate) ? (
              <Button
                label={submit.isPending ? 'Submitting…' : 'Submit report'}
                onPress={() => submit.mutate(id)}
                loading={submit.isPending}
              />
            ) : null}
            {report.data.status === 'SUBMITTED' && can(user, appPermissions.dcrReview) ? (
              <>
                <Button
                  label={decide.isPending ? 'Updating…' : 'Approve report'}
                  onPress={() => decide.mutate(true)}
                  loading={decide.isPending}
                />
                <SecondaryButton
                  label="Reject report"
                  onPress={() => decide.mutate(false)}
                  disabled={decide.isPending}
                />
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
  spaced: {
    marginTop: spacing.md
  },
  row: {
    flexDirection: 'row',
    gap: spacing.md,
    marginTop: spacing.md
  },
  rowItem: {
    flex: 1
  },
  productRow: {
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
