import { useState } from 'react';
import { Alert, Button, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';

import type { RootStackParamList } from '../navigation';
import { createDraftOrder } from '../storage/orders';

type Props = NativeStackScreenProps<RootStackParamList, 'OrderForm'>;

export default function OrderFormScreen({ navigation }: Props) {
  const [distributorTenantId, setDistributorTenantId] = useState('');
  const [productId, setProductId] = useState('');
  const [quantity, setQuantity] = useState('1');
  const [unitPrice, setUnitPrice] = useState('');
  const [orderNumber, setOrderNumber] = useState('');
  const [remarks, setRemarks] = useState('');
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    try {
      await createDraftOrder({
        distributorTenantId: distributorTenantId.trim(),
        productId: productId.trim(),
        quantity: Number(quantity),
        unitPrice: unitPrice.trim() ? Number(unitPrice) : undefined,
        orderNumber: orderNumber.trim() || undefined,
        remarks: remarks.trim() || undefined,
      });
      Alert.alert('Order saved', 'Queued for sync. Open Sync now when you are online.', [
        { text: 'OK', onPress: () => navigation.goBack() },
      ]);
    } catch (error) {
      Alert.alert('Save failed', error instanceof Error ? error.message : 'Unable to save order');
    } finally {
      setBusy(false);
    }
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>New order</Text>
      <Text style={styles.label}>Distributor tenant UUID</Text>
      <TextInput style={styles.input} value={distributorTenantId} onChangeText={setDistributorTenantId} autoCapitalize="none" />
      <Text style={styles.label}>Product UUID</Text>
      <TextInput style={styles.input} value={productId} onChangeText={setProductId} autoCapitalize="none" />
      <Text style={styles.label}>Quantity</Text>
      <TextInput style={styles.input} value={quantity} onChangeText={setQuantity} keyboardType="numeric" />
      <Text style={styles.label}>Unit price (optional)</Text>
      <TextInput style={styles.input} value={unitPrice} onChangeText={setUnitPrice} keyboardType="decimal-pad" />
      <Text style={styles.label}>Order number (optional)</Text>
      <TextInput style={styles.input} value={orderNumber} onChangeText={setOrderNumber} autoCapitalize="none" />
      <Text style={styles.label}>Remarks</Text>
      <TextInput style={styles.input} value={remarks} onChangeText={setRemarks} multiline />
      <Button title={busy ? 'Saving…' : 'Save order'} onPress={save} disabled={busy} />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { padding: 24, gap: 8, backgroundColor: '#f7f9fc' },
  title: { fontSize: 26, fontWeight: '700', marginBottom: 12, color: '#16324f' },
  label: { fontSize: 12, color: '#64748b', marginTop: 8 },
  input: { borderWidth: 1, borderColor: '#cbd5e1', borderRadius: 8, backgroundColor: '#fff', padding: 12 },
});
