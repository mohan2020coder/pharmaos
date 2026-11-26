import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Pressable, StyleSheet, View } from 'react-native';

import { listTourPlans } from '../../../src/api/domain';
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
import type { TourPlanStatus } from '../../../src/types';

type StatusFilter = 'ALL' | TourPlanStatus;

export default function ToursIndexScreen() {
  const router = useRouter();
  const user = useAuthStore((state) => state.user);
  const plans = useQuery({
    queryKey: queryKeys.tourPlans({}),
    queryFn: () => listTourPlans(),
    staleTime: 30000
  });
  const [filter, setFilter] = useState<StatusFilter>('ALL');
  const visible = (plans.data?.data ?? []).filter((plan) => filter === 'ALL' || plan.status === filter);

  return (
    <Screen scroll>
      <View style={styles.header}>
        <View>
          <AppText variant="caption" tone="primary">
            Field execution
          </AppText>
          <AppText variant="title" tone="primaryDark">
            Tour plans
          </AppText>
          <AppText variant="body" tone="muted">
            {plans.data?.meta?.total ?? 0} plans
          </AppText>
        </View>
        {can(user, appPermissions.tourCreate) ? (
          <Button label="New" onPress={() => router.push('/tours/new')} />
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
        {plans.isPending ? <LoadingState message="Loading tour plans…" /> : null}
        {plans.isError ? (
          <ErrorState
            message={plans.error instanceof Error ? plans.error.message : 'Unable to load tour plans'}
            onRetry={() => void plans.refetch()}
          />
        ) : null}
        {!plans.isPending && !plans.isError && visible.length === 0 ? (
          <EmptyState title="No tour plans" message="No plans match the selected filter." />
        ) : null}
        {visible.map((plan) => (
          <Pressable
            accessibilityRole="button"
            key={plan.id}
            onPress={() => router.push(`/tours/${plan.id}`)}
            style={({ pressed }) => [styles.card, pressed ? styles.cardPressed : null]}
          >
            <Card>
              <View style={styles.cardHeader}>
                <AppText variant="section" tone="primaryDark">
                  {plan.title}
                </AppText>
                <Tag label={plan.status} tone={plan.status === 'APPROVED' ? 'success' : 'neutral'} />
              </View>
              <AppText variant="body" tone="muted" style={styles.cardMeta}>
                {formatDate(plan.period_start)} — {formatDate(plan.period_end)}
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
