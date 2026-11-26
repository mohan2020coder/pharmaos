import { useEffect, useState } from 'react';
import { Alert, Button, ScrollView, StyleSheet, Text, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';

import { logout } from '../api/auth';
import type { RootStackParamList } from '../navigation';
import { clearTenantData, countReadModels } from '../storage/database';
import { pendingCount } from '../storage/outbox';
import { clearSession } from '../storage/secure';
import { syncNow, type SyncSummary } from '../sync/engine';

type Props = NativeStackScreenProps<RootStackParamList, 'Home'> & {
  onSignedOut: () => void;
};

export default function HomeScreen({ route, navigation, onSignedOut }: Props) {
  const { user } = route.params;
  const [pending, setPending] = useState(0);
  const [cached, setCached] = useState(0);
  const [orders, setOrders] = useState(0);
  const [busy, setBusy] = useState(false);
  const [summary, setSummary] = useState<SyncSummary | null>(null);

  async function refreshCounts() {
    const [queued, records, orderCount] = await Promise.all([
      pendingCount(user.tenant_id),
      countReadModels(user.tenant_id),
      countReadModels(user.tenant_id, 'order'),
    ]);
    setPending(queued);
    setCached(records);
    setOrders(orderCount);
  }

  useEffect(() => {
    refreshCounts().catch(() => undefined);
  }, [user.tenant_id]);

  async function runSync() {
    setBusy(true);
    try {
      const result = await syncNow();
      setSummary(result);
      await refreshCounts();
    } catch (error) {
      Alert.alert('Sync failed', error instanceof Error ? error.message : 'Unable to sync');
    } finally {
      setBusy(false);
    }
  }

  async function signOut() {
    try {
      await logout();
    } catch {}
    await clearTenantData(user.tenant_id);
    await clearSession();
    onSignedOut();
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>PharmaOS Field</Text>
      <Text style={styles.label}>Signed in as</Text>
      <Text style={styles.value}>{user.email}</Text>
      <Text style={styles.label}>Tenant</Text>
      <Text style={styles.value}>{user.tenant_id}</Text>
      <Text style={styles.label}>Data scope</Text>
      <Text style={styles.value}>{user.data_scope}</Text>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>Offline queue</Text>
        <Text>Pending operations: {pending}</Text>
        <Text>Cached records: {cached}</Text>
        <Text>Cached orders: {orders}</Text>
      </View>

      {summary ? (
        <View style={styles.card}>
          <Text style={styles.cardTitle}>Last sync</Text>
          <Text>Changes pulled: {summary.changesPulled}</Text>
          <Text>Operations pushed: {summary.operationsPushed}</Text>
          <Text>Applied/replayed: {summary.applied}</Text>
          <Text>Conflicts: {summary.conflicts}</Text>
          <Text>Failures: {summary.failures}</Text>
        </View>
      ) : null}

      <Button title="Create visit" onPress={() => navigation.navigate('VisitForm')} />
      <Button title="Create order" onPress={() => navigation.navigate('OrderForm')} />
      <Button title={busy ? 'Syncing…' : 'Sync now'} onPress={runSync} disabled={busy} />
      <Button title="Sign out" onPress={signOut} color="#b91c1c" />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { padding: 24, gap: 10, backgroundColor: '#f7f9fc' },
  title: { fontSize: 28, fontWeight: '700', marginBottom: 12, color: '#16324f' },
  label: { fontSize: 12, color: '#64748b', marginTop: 6 },
  value: { fontSize: 16, color: '#0f172a' },
  card: { backgroundColor: '#fff', borderWidth: 1, borderColor: '#e2e8f0', borderRadius: 10, padding: 16, gap: 4 },
  cardTitle: { fontSize: 18, fontWeight: '600', marginBottom: 4 },
});
