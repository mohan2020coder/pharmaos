import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useLocalSearchParams } from 'expo-router';
import { Alert, StyleSheet, View } from 'react-native';

import { getVisit, submitVisit } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  DetailRow,
  ErrorState,
  LoadingState,
  Screen,
  Tag
} from '../../../src/components/ui';
import { useEntityNames } from '../../../src/hooks/usePartyChoices';
import { formatDateTime } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { spacing } from '../../../src/theme/tokens';

export default function VisitDetailScreen() {
  const params = useLocalSearchParams<{ id: string }>();
  const id = String(params.id ?? '');
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const visit = useQuery({
    queryKey: ['visits', id],
    queryFn: () => getVisit(id),
    enabled: Boolean(id)
  });
  const submit = useMutation({
    mutationFn: submitVisit,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['visits'] });
      void visit.refetch();
    },
    onError: (error) => {
      Alert.alert('Unable to submit visit', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  return (
    <Screen scroll>
      {visit.isPending ? <LoadingState message="Loading visit…" /> : null}
      {visit.isError ? (
        <ErrorState
          message={visit.error instanceof Error ? visit.error.message : 'Unable to load visit'}
          onRetry={() => void visit.refetch()}
        />
      ) : null}
      {visit.data ? (
        <>
          <View style={styles.header}>
            <View>
              <AppText variant="caption" tone="primary">
                {visit.data.entity_type.toLowerCase()} · {labelFor(visit.data.entity_id, visit.data.entity_type)}
              </AppText>
              <AppText variant="title" tone="primaryDark">
                Visit details
              </AppText>
            </View>
            <Tag
              label={visit.data.status}
              tone={visit.data.status === 'APPROVED' ? 'success' : 'neutral'}
            />
          </View>

          <Card style={styles.section}>
            <AppText variant="heading">Record</AppText>
            <DetailRow
              label="Customer"
              value={<AppText variant="body">{labelFor(visit.data.entity_id, visit.data.entity_type)}</AppText>}
            />
            <DetailRow
              label="Visited at"
              value={<AppText variant="body">{formatDateTime(visit.data.visited_at)}</AppText>}
            />
            <DetailRow
              label="Employee"
              value={<AppText variant="body">{labelFor(visit.data.employee_id, 'EMPLOYEE')}</AppText>}
            />
            {visit.data.remarks ? (
              <DetailRow label="Remarks" value={<AppText variant="body">{visit.data.remarks}</AppText>} />
            ) : null}
          </Card>

          {visit.data.check_in_lat !== undefined && visit.data.check_in_lng !== undefined ? (
            <Card style={styles.section}>
              <AppText variant="heading">Check-in</AppText>
              <DetailRow
                label="Coordinates"
                value={
                  <AppText variant="body">
                    {visit.data.check_in_lat.toFixed(5)}, {visit.data.check_in_lng.toFixed(5)}
                  </AppText>
                }
              />
              {visit.data.check_in_accuracy !== undefined ? (
                <DetailRow
                  label="Accuracy"
                  value={<AppText variant="body">{Math.round(visit.data.check_in_accuracy)}m</AppText>}
                />
              ) : null}
            </Card>
          ) : null}

          {visit.data.status === 'DRAFT' && can(user, appPermissions.visitCreate) ? (
            <Button
              label={submit.isPending ? 'Submitting…' : 'Submit visit'}
              onPress={() => submit.mutate(id)}
              loading={submit.isPending}
            />
          ) : null}
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
  }
});
