import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import { createOrder, listDistributionLinks, listProducts } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  ChoiceList,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  TextField
} from '../../../src/components/ui';
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatMoney, todayInput } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { useSyncStore } from '../../../src/stores/sync';
import { createDraftOrder } from '../../../src/storage/orders';
import { colors, spacing } from '../../../src/theme/tokens';
import type { CreateOrderItemInput } from '../../../src/types';

type DraftItem = CreateOrderItemInput & { key: string };

export default function NewOrderScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const [distributorId, setDistributorId] = useState('');
  const [orderNumber, setOrderNumber] = useState('');
  const [orderDate, setOrderDate] = useState(todayInput());
  const [requiredBy, setRequiredBy] = useState('');
  const [remarks, setRemarks] = useState('');
  const [productSearch, setProductSearch] = useState('');
  const [productId, setProductId] = useState<string>();
  const [quantity, setQuantity] = useState('1');
  const [unitPrice, setUnitPrice] = useState('0');
  const [items, setItems] = useState<DraftItem[]>([]);
  const storeRun = useSyncStore((state) => state.run);

  const links = useQuery({
    queryKey: queryKeys.distributionLinks,
    queryFn: async () => (await listDistributionLinks()).data,
    enabled: can(user, appPermissions.distributorView),
    staleTime: 30000
  });
  const catalog = useQuery({
    queryKey: queryKeys.products(productSearch),
    queryFn: async () => (await listProducts(productSearch)).data.map((item) => ({
      id: item.id,
      label: item.name,
      detail: `${item.code} · ${formatMoney(item.mrp)}`
    })),
    enabled: can(user, appPermissions.productView),
    staleTime: 30000
  });

  const save = useMutation({
    mutationFn: createOrder,
    onSuccess: (order) => {
      void queryClient.invalidateQueries({ queryKey: ['orders'] });
      router.push(`/orders/${order.id}`);
    },
    onError: (error) => {
      Alert.alert('Unable to create order', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleAddItem() {
    if (!productId) {
      Alert.alert('Product required', 'Select a product or enter its ID.');
      return;
    }
    setItems((current) => [
      ...current,
      {
        key: `${productId}-${current.length}`,
        product_id: productId,
        quantity: Number(quantity) || 0,
        unit_price: Number(unitPrice) || 0
      }
    ]);
    setProductId(undefined);
    setQuantity('1');
    setUnitPrice('0');
  }

  function handleSubmit() {
    if (!distributorId) {
      Alert.alert('Distributor required', 'Select or enter a distributor tenant ID.');
      return;
    }
    if (items.length === 0) {
      Alert.alert('Items required', 'Add at least one order item.');
      return;
    }
    save.mutate({
      distributor_tenant_id: distributorId,
      order_number: orderNumber.trim() || undefined,
      order_date: orderDate,
      required_by: requiredBy || undefined,
      remarks: remarks.trim() || undefined,
      items: items.map(({ product_id, quantity: itemQuantity, unit_price }) => ({
        product_id,
        quantity: itemQuantity,
        unit_price
      }))
    });
  }

  async function handleQueueOffline() {
    if (!distributorId) {
      Alert.alert('Distributor required', 'Select or enter a distributor tenant ID.');
      return;
    }
    if (items.length === 0) {
      Alert.alert('Items required', 'Add at least one order item.');
      return;
    }
    try {
      await createDraftOrder({
        distributorTenantId: distributorId,
        orderNumber: orderNumber.trim() || undefined,
        orderDate,
        remarks: remarks.trim() || undefined,
        items: items.map(({ product_id, quantity: itemQuantity, unit_price }) => ({
          product_id,
          quantity: itemQuantity,
          unit_price
        }))
      });
      void storeRun('manual');
      void queryClient.invalidateQueries({ queryKey: ['outbox-counts'] });
      void queryClient.invalidateQueries({ queryKey: ['outbox-problems'] });
      Alert.alert('Queued for sync', 'The order is saved locally and will sync when a connection is available.', [
        { text: 'View sync center', onPress: () => router.replace('/sync') },
        { text: 'OK', style: 'cancel' }
      ]);
    } catch (error) {
      Alert.alert('Unable to queue order', error instanceof Error ? error.message : 'Please try again.');
    }
  }

  return (
    <Screen scroll>
      <AppText variant="title" tone="primaryDark">
        New order
      </AppText>
      <AppText variant="body" tone="muted" style={styles.intro}>
        Orders are created as drafts and require an idempotency key.
      </AppText>

      {!can(user, appPermissions.orderManage) ? (
        <ErrorState message="You do not have permission to manage orders." />
      ) : null}

      <Card style={styles.section}>
        <AppText variant="heading">Distributor</AppText>
        <View style={styles.spaced}>
          <TextField
            label="Distributor tenant ID"
            value={distributorId}
            onChangeText={setDistributorId}
            placeholder="UUID"
          />
        </View>
        {links.isPending ? <LoadingState message="Loading distribution links…" /> : null}
        {links.data && links.data.length > 0 ? (
          <View style={styles.spaced}>
            <AppText variant="caption" tone="muted">
              Or choose an active link
            </AppText>
            <View style={styles.spaced}>
              <ChoiceList
                choices={links.data.map((link) => ({
                  id: link.distributor_tenant_id,
                  label: link.distributor_tenant_id,
                  detail: `${link.status} · ${link.payment_terms_days} day terms`
                }))}
                selectedId={distributorId}
                onSelect={setDistributorId}
              />
            </View>
          </View>
        ) : null}
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Order details</AppText>
        <View style={styles.spaced}>
          <TextField label="Order number" value={orderNumber} onChangeText={setOrderNumber} />
        </View>
        <View style={styles.spaced}>
          <TextField label="Order date (YYYY-MM-DD)" value={orderDate} onChangeText={setOrderDate} />
        </View>
        <View style={styles.spaced}>
          <TextField label="Required by (YYYY-MM-DD)" value={requiredBy} onChangeText={setRequiredBy} />
        </View>
        <View style={styles.spaced}>
          <TextField label="Remarks" value={remarks} onChangeText={setRemarks} multiline />
        </View>
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Items</AppText>
        {items.length === 0 ? (
          <AppText variant="body" tone="muted" style={styles.spaced}>
            No items added yet.
          </AppText>
        ) : null}
        {items.map((item, index) => (
          <View key={item.key} style={styles.itemRow}>
            <View style={styles.itemCopy}>
              <AppText variant="section">{labelFor(item.product_id, 'PRODUCT')}</AppText>
              <AppText variant="caption" tone="muted">
                Qty {item.quantity} · {formatMoney(item.unit_price)}
              </AppText>
            </View>
            <SecondaryButton
              label="Remove"
              onPress={() => setItems((current) => current.filter((_, itemIndex) => itemIndex !== index))}
            />
          </View>
        ))}

        <View style={styles.addItem}>
          <TextField label="Search product" value={productSearch} onChangeText={setProductSearch} />
          {catalog.isPending ? <LoadingState message="Loading products…" /> : null}
          {catalog.data ? (
            <View style={styles.spaced}>
              <ChoiceList choices={catalog.data} selectedId={productId} onSelect={setProductId} />
            </View>
          ) : null}
          {!can(user, appPermissions.productView) ? (
            <View style={styles.spaced}>
              <TextField label="Product ID" value={productId} onChangeText={setProductId} />
            </View>
          ) : null}
          <View style={styles.row}>
            <View style={styles.rowItem}>
              <TextField label="Quantity" value={quantity} onChangeText={setQuantity} keyboardType="numeric" />
            </View>
            <View style={styles.rowItem}>
              <TextField label="Unit price" value={unitPrice} onChangeText={setUnitPrice} keyboardType="numeric" />
            </View>
          </View>
          <View style={styles.spaced}>
            <SecondaryButton label="Add item" onPress={handleAddItem} />
          </View>
        </View>
      </Card>

      <View style={styles.actions}>
        <Button label={save.isPending ? 'Creating…' : 'Create order'} onPress={handleSubmit} loading={save.isPending} />
        <SecondaryButton label="Queue offline" onPress={() => void handleQueueOffline()} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  intro: {
    marginTop: spacing.sm
  },
  section: {
    marginTop: spacing.lg
  },
  spaced: {
    marginTop: spacing.md
  },
  itemRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: spacing.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.borderSoft,
    paddingVertical: spacing.md
  },
  itemCopy: {
    flex: 1
  },
  addItem: {
    marginTop: spacing.md,
    paddingTop: spacing.md
  },
  row: {
    flexDirection: 'row',
    gap: spacing.md,
    marginTop: spacing.md
  },
  rowItem: {
    flex: 1
  },
  actions: {
    gap: spacing.md,
    marginTop: spacing.lg
  }
});
