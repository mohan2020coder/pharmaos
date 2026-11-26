import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { Pressable, StyleSheet, View } from 'react-native';

import { getDistributionReport, getFieldActivity } from '../../src/api/domain';
import { get } from '../../src/api/client';
import {
  AppText,
  Card,
  DetailRow,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Tag
} from '../../src/components/ui';
import { daysAgoInput, formatMoney, todayInput } from '../../src/lib/format';
import { appPermissions, can } from '../../src/lib/permissions';
import { queryKeys } from '../../src/lib/queryKeys';
import { useAuthStore } from '../../src/stores/auth';
import { useSyncStore } from '../../src/stores/sync';
import { pendingCount, problemCount } from '../../src/storage/outbox';
import { colors, radius, shadows, spacing } from '../../src/theme/tokens';
import type { SyncStatus } from '../../src/types';

type Shortcut = {
  label: string;
  caption: string;
  href: string;
  permission: string;
};

const shortcuts: Shortcut[] = [
  { label: 'Record visit', caption: 'Check in and log a field call', href: '/visits/new', permission: appPermissions.visitCreate },
  { label: 'Plan tour', caption: 'Build a territory plan', href: '/tours/new', permission: appPermissions.tourCreate },
  { label: 'Daily call report', caption: 'Capture product details', href: '/dcr/new', permission: appPermissions.dcrCreate },
  { label: 'Create order', caption: 'Raise a distribution order', href: '/orders/new', permission: appPermissions.orderManage }
];

export default function HomeScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const signOut = useAuthStore((state) => state.signOut);
  const isOnline = useSyncStore((state) => state.isOnline);
  const isSyncing = useSyncStore((state) => state.isSyncing);
  const lastRunAt = useSyncStore((state) => state.lastRunAt);
  const storeRun = useSyncStore((state) => state.run);
  const from = daysAgoInput(30);
  const to = todayInput();

  const sync = useQuery({
    queryKey: queryKeys.syncStatus,
    queryFn: () => get<SyncStatus>('/api/v1/sync/status'),
    enabled: Boolean(user),
    staleTime: 30000
  });
  const localQueue = useQuery({
    queryKey: ['outbox-counts', user?.tenant_id],
    queryFn: async () => ({
      pending: await pendingCount(user?.tenant_id as string),
      problems: await problemCount(user?.tenant_id as string)
    }),
    enabled: Boolean(user)
  });
  const field = useQuery({
    queryKey: queryKeys.fieldActivity(from, to),
    queryFn: () => getFieldActivity(from, to),
    enabled: can(user, appPermissions.reportsView),
    staleTime: 60000
  });
  const distribution = useQuery({
    queryKey: queryKeys.distributionReport(from, to),
    queryFn: () => getDistributionReport(from, to),
    enabled: can(user, appPermissions.reportsView),
    staleTime: 60000
  });
  const availableShortcuts = shortcuts.filter((shortcut) => can(user, shortcut.permission));

  async function handleSignOut() {
    await signOut();
    router.replace('/login');
  }

  function handleSync() {
    void storeRun('manual').then(() => {
      void queryClient.invalidateQueries();
    });
  }

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View style={styles.headerCopy}>
          <AppText variant="caption" tone="primary">
            Workspace
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Field overview
          </AppText>
          <AppText variant="body" tone="muted">
            {user?.email ?? 'Signed in'}
          </AppText>
        </View>
        <SecondaryButton label="Sign out" onPress={handleSignOut} />
      </View>

      {availableShortcuts.length > 0 ? (
        <View style={styles.section}>
          <AppText variant="section" tone="primaryDark">
            Quick actions
          </AppText>
          <View style={styles.shortcutGrid}>
            {availableShortcuts.map((shortcut) => (
              <Pressable
                accessibilityRole="button"
                key={shortcut.href}
                onPress={() => router.push(shortcut.href as never)}
                style={({ pressed }) => [styles.shortcut, pressed ? styles.shortcutPressed : null]}
              >
                <AppText variant="section">{shortcut.label}</AppText>
                <AppText variant="caption" tone="muted" style={styles.shortcutCaption}>
                  {shortcut.caption}
                </AppText>
              </Pressable>
            ))}
          </View>
        </View>
      ) : null}

      {field.isPending ? <LoadingState message="Loading field activity…" /> : null}
      {field.isError ? (
        <ErrorState
          message={field.error instanceof Error ? field.error.message : 'Unable to load field activity'}
          onRetry={() => void field.refetch()}
        />
      ) : null}
      {field.data ? (
        <Card style={styles.section}>
          <AppText variant="heading">Field activity</AppText>
          <AppText variant="caption" tone="muted">
            Last 30 days · {field.data.scope}
          </AppText>
          <View style={styles.metrics}>
            <Metric label="Visits" value={String(field.data.visits.total)} />
            <Metric label="Submitted" value={String(field.data.visits.submitted)} />
            <Metric label="Customers" value={String(field.data.visits.unique_customers)} />
            <Metric label="DCRs" value={String(field.data.daily_call_reports.total)} />
          </View>
        </Card>
      ) : null}

      {distribution.isPending ? <LoadingState message="Loading distribution summary…" /> : null}
      {distribution.isError ? (
        <ErrorState
          message={
            distribution.error instanceof Error
              ? distribution.error.message
              : 'Unable to load distribution summary'
          }
          onRetry={() => void distribution.refetch()}
        />
      ) : null}
      {distribution.data ? (
        <Card style={styles.section}>
          <AppText variant="heading">Distribution</AppText>
          <AppText variant="caption" tone="muted">
            Last 30 days
          </AppText>
          <View style={styles.metrics}>
            <Metric label="Orders" value={String(distribution.data.orders.total)} />
            <Metric label="Approved" value={String(distribution.data.orders.approved)} />
            <Metric label="Value" value={formatMoney(distribution.data.orders.value)} />
            <Metric label="On hand" value={String(distribution.data.inventory.on_hand)} />
          </View>
        </Card>
      ) : null}

      <Card style={styles.section}>
        <AppText variant="heading">Session</AppText>
        <DetailRow label="Tenant" value={<AppText variant="body">{user?.tenant_id ?? 'Unavailable'}</AppText>} />
        <DetailRow label="Data scope" value={<AppText variant="body">{user?.data_scope ?? 'Unavailable'}</AppText>} />
        <DetailRow label="Permissions" value={<AppText variant="body">{user?.permissions.length ?? 0}</AppText>} />
        <View style={styles.tags}>
          {(user?.roles ?? []).map((role) => (
            <Tag key={role} label={role} tone={role === 'SUPER_ADMIN' ? 'primary' : 'neutral'} />
          ))}
        </View>
      </Card>

      <Card style={styles.section}>
        <View style={styles.cardHeader}>
          <AppText variant="heading">Offline sync</AppText>
          <Tag
            label={isOnline === false ? 'Offline' : isSyncing ? 'Syncing' : 'Online'}
            tone={isOnline === false ? 'neutral' : isSyncing ? 'primary' : 'success'}
          />
        </View>
        <View style={styles.metrics}>
          <Metric label="Pending" value={String(localQueue.data?.pending ?? 0)} />
          <Metric label="Review" value={String(localQueue.data?.problems ?? 0)} />
        </View>
        <DetailRow
          label="Last attempt"
          value={
            <AppText variant="body">
              {lastRunAt ? new Date(lastRunAt).toLocaleString() : 'No attempt'}
            </AppText>
          }
        />
        <View style={styles.cardActions}>
          <SecondaryButton label="Sync now" onPress={handleSync} disabled={isOnline === false || isSyncing} />
          <SecondaryButton label="Sync center" onPress={() => router.push('/sync')} />
        </View>
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Backend sync status</AppText>
        {sync.isPending ? <LoadingState message="Loading sync status…" /> : null}
        {sync.isError ? (
          <ErrorState
            message={sync.error instanceof Error ? sync.error.message : 'Unable to load sync status'}
            onRetry={() => void sync.refetch()}
          />
        ) : null}
        {sync.data ? (
          <View style={styles.details}>
            <DetailRow
              label="Pending operations"
              value={<AppText variant="body">{sync.data.pending_operations}</AppText>}
            />
            <DetailRow
              label="Supported entities"
              value={<AppText variant="body">{sync.data.supported_entities.length}</AppText>}
            />
            <DetailRow
              label="Server time"
              value={<AppText variant="body">{new Date(sync.data.server_time).toLocaleString()}</AppText>}
            />
          </View>
        ) : null}
      </Card>
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
    gap: spacing.lg,
    marginBottom: spacing.xl
  },
  headerCopy: {
    flex: 1
  },
  section: {
    marginBottom: spacing.xl
  },
  shortcutGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
    marginTop: spacing.md
  },
  shortcut: {
    width: '47%',
    minHeight: 92,
    justifyContent: 'center',
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.borderSoft,
    backgroundColor: colors.surface,
    padding: spacing.lg,
    ...shadows.card
  },
  shortcutPressed: {
    opacity: 0.75
  },
  shortcutCaption: {
    marginTop: spacing.xs
  },
  metrics: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
    marginTop: spacing.md
  },
  metric: {
    minWidth: 72,
    flexGrow: 1,
    borderRadius: radius.md,
    backgroundColor: colors.background,
    padding: spacing.md
  },
  cardHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: spacing.md
  },
  cardActions: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.md,
    marginTop: spacing.md
  },
  details: {
    gap: spacing.sm,
    marginTop: spacing.md
  },
  tags: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.sm,
    marginTop: spacing.md
  }
});
