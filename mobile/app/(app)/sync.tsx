import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import {
  AppText,
  Button,
  Card,
  DetailRow,
  EmptyState,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Tag
} from '../../src/components/ui';
import { useAuthStore } from '../../src/stores/auth';
import { useSyncStore } from '../../src/stores/sync';
import { getDatabase, countReadModels, getSyncState } from '../../src/storage/database';
import {
  listProblemOperations,
  pendingCount,
  problemCount,
  resolveOperation,
  retryOperation,
} from '../../src/storage/outbox';
import { colors, spacing } from '../../src/theme/tokens';

export default function SyncCenterScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const tenantId = user?.tenant_id;
  const isOnline = useSyncStore((state) => state.isOnline);
  const isSyncing = useSyncStore((state) => state.isSyncing);
  const lastRunAt = useSyncStore((state) => state.lastRunAt);
  const lastReason = useSyncStore((state) => state.lastReason);
  const lastSummary = useSyncStore((state) => state.lastSummary);
  const lastError = useSyncStore((state) => state.lastError);
  const storeRun = useSyncStore((state) => state.run);
  const [busyOperationId, setBusyOperationId] = useState<string>();

  const problems = useQuery({
    queryKey: ['outbox-problems', tenantId],
    queryFn: () => listProblemOperations(tenantId as string, 100),
    enabled: Boolean(tenantId)
  });
  const counts = useQuery({
    queryKey: ['outbox-counts', tenantId],
    queryFn: async () => ({
      pending: await pendingCount(tenantId as string),
      problems: await problemCount(tenantId as string),
      cached: await countReadModels(tenantId as string),
    }),
    enabled: Boolean(tenantId)
  });
  const localState = useQuery({
    queryKey: ['sync-local-state', tenantId],
    queryFn: () => getSyncState(tenantId as string),
    enabled: Boolean(tenantId)
  });

  function invalidateLocal() {
    void queryClient.invalidateQueries({ queryKey: ['outbox-problems'] });
    void queryClient.invalidateQueries({ queryKey: ['outbox-counts'] });
    void queryClient.invalidateQueries({ queryKey: ['sync-local-state'] });
    void queryClient.invalidateQueries();
  }

  function handleSync() {
    void storeRun('manual').then((summary) => {
      if (summary) {
        invalidateLocal();
      }
    });
  }

  async function handleRetry(id: string) {
    setBusyOperationId(id);
    try {
      const database = await getDatabase();
      await retryOperation(database, id);
      await storeRun('manual');
      invalidateLocal();
    } catch (error) {
      Alert.alert('Unable to retry', error instanceof Error ? error.message : 'Please try again.');
    } finally {
      setBusyOperationId(undefined);
    }
  }

  function handleResolve(id: string) {
    Alert.alert(
      'Resolve conflict',
      'Use the server value to close the local operation. Local changes for this operation will be discarded.',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Use server',
          onPress: async () => {
            setBusyOperationId(id);
            try {
              const database = await getDatabase();
              await resolveOperation(database, id, 'USE_SERVER');
              invalidateLocal();
            } catch (error) {
              Alert.alert('Unable to resolve', error instanceof Error ? error.message : 'Please try again.');
            } finally {
              setBusyOperationId(undefined);
            }
          },
        },
      ],
    );
  }

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View>
          <AppText variant="caption" tone="primary">
            Offline readiness
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Sync center
          </AppText>
        </View>
        <Tag
          label={isOnline === false ? 'Offline' : isSyncing ? 'Syncing' : 'Online'}
          tone={isOnline === false ? 'neutral' : isSyncing ? 'primary' : 'success'}
        />
      </View>

      <Card style={styles.section}>
        <AppText variant="heading">Overview</AppText>
        {counts.isPending ? <LoadingState message="Loading queue counts…" /> : null}
        {counts.isError ? (
          <ErrorState
            message={counts.error instanceof Error ? counts.error.message : 'Unable to load queue counts'}
            onRetry={() => void counts.refetch()}
          />
        ) : null}
        {counts.data ? (
          <View style={styles.metrics}>
            <Metric label="Pending" value={String(counts.data.pending)} />
            <Metric label="Problems" value={String(counts.data.problems)} />
            <Metric label="Cached" value={String(counts.data.cached)} />
          </View>
        ) : null}
        <DetailRow
          label="Last local sync"
          value={
            <AppText variant="body">
              {localState.data?.last_synced_at
                ? new Date(localState.data.last_synced_at).toLocaleString()
                : 'Not synced'}
            </AppText>
          }
        />
        <DetailRow
          label="Last attempt"
          value={
            <AppText variant="body">
              {lastRunAt ? `${new Date(lastRunAt).toLocaleString()} · ${lastReason ?? 'manual'}` : 'No attempt'}
            </AppText>
          }
        />
        {lastError ? (
          <AppText variant="body" tone="danger" style={styles.spaced}>
            {lastError}
          </AppText>
        ) : null}
        <View style={styles.spaced}>
          <Button label={isSyncing ? 'Syncing…' : 'Sync now'} onPress={handleSync} loading={isSyncing} disabled={isOnline === false} />
        </View>
      </Card>

      {lastSummary ? (
        <Card style={styles.section}>
          <AppText variant="heading">Last sync result</AppText>
          <View style={styles.metrics}>
            <Metric label="Pulled" value={String(lastSummary.changesPulled)} />
            <Metric label="Pushed" value={String(lastSummary.operationsPushed)} />
            <Metric label="Applied" value={String(lastSummary.applied)} />
            <Metric label="Conflicts" value={String(lastSummary.conflicts)} />
            <Metric label="Failures" value={String(lastSummary.failures)} />
          </View>
        </Card>
      ) : null}

      <Card style={styles.section}>
        <AppText variant="heading">Operations needing review</AppText>
        {problems.isPending ? <LoadingState message="Loading operations…" /> : null}
        {problems.isError ? (
          <ErrorState
            message={problems.error instanceof Error ? problems.error.message : 'Unable to load operations'}
            onRetry={() => void problems.refetch()}
          />
        ) : null}
        {problems.data && problems.data.length === 0 ? (
          <EmptyState title="No problems" message="No operations are waiting for manual review." />
        ) : null}
        {problems.data?.map((operation) => (
          <View key={operation.id} style={styles.operation}>
            <View style={styles.operationHeader}>
              <AppText variant="section" tone="primaryDark">
                {operation.operation}
              </AppText>
              <Tag label={operation.status} />
            </View>
            <DetailRow label="Entity" value={<AppText variant="caption">{operation.entity_id}</AppText>} />
            <DetailRow label="Error" value={<AppText variant="caption">{operation.error_code ?? 'SYNC_ERROR'}</AppText>} />
            <DetailRow label="Attempts" value={<AppText variant="body">{operation.attempts}</AppText>} />
            <DetailRow label="Created" value={<AppText variant="body">{new Date(operation.created_at).toLocaleString()}</AppText>} />
            {operation.last_error ? (
              <AppText variant="caption" tone="danger" style={styles.spaced}>
                {operation.last_error}
              </AppText>
            ) : null}
            <View style={styles.operationActions}>
              <SecondaryButton
                label="Retry"
                onPress={() => void handleRetry(operation.id)}
                disabled={busyOperationId === operation.id}
              />
              {operation.status === 'CONFLICT' ? (
                <SecondaryButton
                  label="Use server"
                  onPress={() => handleResolve(operation.id)}
                  disabled={busyOperationId === operation.id}
                />
              ) : null}
            </View>
          </View>
        ))}
      </Card>

      <View style={styles.section}>
        <SecondaryButton label="Back to home" onPress={() => router.replace('/home')} />
      </View>
    </Screen>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <View style={styles.metric}>
      <AppText variant="section" tone="primaryDark">
        {value}
      </AppText>
      <AppText variant="caption" tone="muted">
        {label}
      </AppText>
    </View>
  );
}

const styles = StyleSheet.create({
  header: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: spacing.md,
    marginBottom: spacing.lg,
  },
  section: {
    marginBottom: spacing.lg,
  },
  spaced: {
    marginTop: spacing.md,
  },
  metrics: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
    marginTop: spacing.md,
    marginBottom: spacing.md,
  },
  metric: {
    minWidth: 72,
    flexGrow: 1,
    borderRadius: 12,
    backgroundColor: colors.background,
    padding: spacing.md,
  },
  operation: {
    borderTopWidth: 1,
    borderTopColor: colors.borderSoft,
    paddingTop: spacing.md,
    paddingBottom: spacing.md,
  },
  operationHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: spacing.md,
  },
  operationActions: {
    flexDirection: 'row',
    gap: spacing.md,
    marginTop: spacing.md,
  },
});
