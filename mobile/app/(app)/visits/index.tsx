import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { listVisits } from '../../../src/api/domain';
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
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatDateTime } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { colors, radius, spacing } from '../../../src/theme/tokens';
import type { VisitStatus } from '../../../src/types';

type StatusFilter = 'ALL' | VisitStatus;

export default function VisitsIndexScreen() {
  const router = useRouter();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const status = useQuery({
    queryKey: queryKeys.visits({ status: undefined }),
    queryFn: () => listVisits(),
    staleTime: 30000
  });
  const [filter, setFilter] = useState<StatusFilter>('ALL');

  const visits = (status.data?.data ?? []).filter((visit) => filter === 'ALL' || visit.status === filter);

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View>
          <AppText variant="caption" tone="primary">
            Field execution
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Visits
          </AppText>
          <AppText variant="body" tone="muted">
            {status.data?.meta?.total ?? 0} recorded
          </AppText>
        </View>
        {can(user, appPermissions.visitCreate) ? (
          <Button label="New" onPress={() => router.push('/visits/new')} />
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
        {status.isPending ? <LoadingState message="Loading visits…" /> : null}
        {status.isError ? (
          <ErrorState
            message={status.error instanceof Error ? status.error.message : 'Unable to load visits'}
            onRetry={() => void status.refetch()}
          />
        ) : null}
        {!status.isPending && !status.isError && visits.length === 0 ? (
          <EmptyState title="No visits" message="No visits match the selected filter." />
        ) : null}
        {visits.map((visit) => (
          <Pressable
            accessibilityRole="button"
            key={visit.id}
            onPress={() => router.push(`/visits/${visit.id}`)}
            style={({ pressed }) => [styles.card, pressed ? styles.cardPressed : null]}
          >
            <Card>
              <View style={styles.cardHeader}>
                <AppText variant="section" tone="primaryDark">
                  {visit.entity_type.toLowerCase()}
                </AppText>
                <Tag label={visit.status} tone={visit.status === 'APPROVED' ? 'success' : 'neutral'} />
              </View>
              <AppText variant="section" tone="primaryDark">
                {labelFor(visit.entity_id, visit.entity_type)}
              </AppText>
              <AppText variant="body" tone="muted" style={styles.cardMeta}>
                {formatDateTime(visit.visited_at)}
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
