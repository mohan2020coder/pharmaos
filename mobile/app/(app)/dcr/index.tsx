import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { listDcr } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  EmptyState,
  ErrorState,
  LoadingState,
  Screen,
  Segmented,
  Tag
} from '../../../src/components/ui';
import { formatDate } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { colors, radius, spacing } from '../../../src/theme/tokens';
import type { DcrStatus } from '../../../src/types';

type StatusFilter = 'ALL' | DcrStatus;

export default function DcrIndexScreen() {
  const router = useRouter();
  const user = useAuthStore((state) => state.user);
  const reports = useQuery({
    queryKey: queryKeys.dcr({}),
    queryFn: () => listDcr(),
    staleTime: 30000
  });
  const [filter, setFilter] = useState<StatusFilter>('ALL');
  const visible = (reports.data?.data ?? []).filter((report) => filter === 'ALL' || report.status === filter);

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View>
          <AppText variant="caption" tone="primary">
            Field execution
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Daily call reports
          </AppText>
          <AppText variant="body" tone="muted">
            {reports.data?.meta?.total ?? 0} reports
          </AppText>
        </View>
        {can(user, appPermissions.dcrCreate) ? (
          <Button label="New" onPress={() => router.push('/dcr/new')} />
        ) : null}
      </View>

      <Segmented
        value={filter}
        onChange={setFilter}
        options={[
          { label: 'All', value: 'ALL' },
          { label: 'Draft', value: 'DRAFT' },
          { label: 'Submitted', value: 'SUBMITTED' },
          { label: 'Approved', value: 'APPROVED' }
        ]}
      />

      <View style={styles.section}>
        {reports.isPending ? <LoadingState message="Loading reports…" /> : null}
        {reports.isError ? (
          <ErrorState
            message={reports.error instanceof Error ? reports.error.message : 'Unable to load reports'}
            onRetry={() => void reports.refetch()}
          />
        ) : null}
        {!reports.isPending && !reports.isError && visible.length === 0 ? (
          <EmptyState title="No reports" message="No reports match the selected filter." />
        ) : null}
        {visible.map((report) => (
          <Pressable
            accessibilityRole="button"
            key={report.id}
            onPress={() => router.push(`/dcr/${report.id}`)}
            style={({ pressed }) => [styles.card, pressed ? styles.cardPressed : null]}
          >
            <Card>
              <View style={styles.cardHeader}>
                <AppText variant="section" tone="primaryDark">
                  {formatDate(report.report_date)}
                </AppText>
                <Tag label={report.status} tone={report.status === 'APPROVED' ? 'success' : 'neutral'} />
              </View>
              <AppText variant="body" tone="muted" style={styles.cardMeta}>
                {report.total_visits} visits recorded
              </AppText>
            </Card>
          </Pressable>
        ))}
      </View>
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
    gap: spacing.md
  },
  cardMeta: {
    marginTop: spacing.md
  }
});
