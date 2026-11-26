import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import {
  createTourLine,
  decideTourPlan,
  getTourPlan,
  listTourLines,
  submitTourPlan
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
  Segmented,
  Tag,
  TextField
} from '../../../src/components/ui';
import { useEntityNames, usePartyChoices } from '../../../src/hooks/usePartyChoices';
import { formatDate, todayInput } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { queryKeys } from '../../../src/lib/queryKeys';
import { useAuthStore } from '../../../src/stores/auth';
import { colors, spacing } from '../../../src/theme/tokens';
import type { EntityType } from '../../../src/types';

export default function TourPlanDetailScreen() {
  const params = useLocalSearchParams<{ id: string }>();
  const id = String(params.id ?? '');
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const { labelFor } = useEntityNames();
  const plan = useQuery({
    queryKey: ['tour-plans', id],
    queryFn: () => getTourPlan(id),
    enabled: Boolean(id)
  });
  const lines = useQuery({
    queryKey: queryKeys.tourLines(id),
    queryFn: () => listTourLines(id),
    enabled: Boolean(id)
  });

  const [entityType, setEntityType] = useState<EntityType>('DOCTOR');
  const [search, setSearch] = useState('');
  const [entityId, setEntityId] = useState<string>();
  const [plannedDate, setPlannedDate] = useState(todayInput());
  const [purpose, setPurpose] = useState('');
  const parties = usePartyChoices(entityType, search);

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['tour-plans'] });
    void plan.refetch();
    void lines.refetch();
  }

  const addLine = useMutation({
    mutationFn: () =>
      createTourLine(id, {
        planned_date: plannedDate,
        entity_type: entityType,
        entity_id: entityId as string,
        purpose: purpose.trim() || undefined,
        sequence: (lines.data?.length ?? 0) + 1
      }),
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to add line', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  const submit = useMutation({
    mutationFn: submitTourPlan,
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to submit plan', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  const decide = useMutation({
    mutationFn: (approve: boolean) => decideTourPlan(id, approve),
    onSuccess: refresh,
    onError: (error) => {
      Alert.alert('Unable to update plan', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleAddLine() {
    if (!entityId) {
      Alert.alert('Customer required', 'Select a customer for this tour line.');
      return;
    }
    addLine.mutate();
  }

  return (
    <Screen scroll>
      {plan.isPending ? <LoadingState message="Loading tour plan…" /> : null}
      {plan.isError ? (
        <ErrorState
          message={plan.error instanceof Error ? plan.error.message : 'Unable to load tour plan'}
          onRetry={() => void plan.refetch()}
        />
      ) : null}
      {plan.data ? (
        <>
          <View style={styles.header}>
            <View>
              <AppText variant="caption" tone="primary">
                Tour plan
              </AppText>
              <AppText variant="title" tone="primaryDark">
                {plan.data.title}
              </AppText>
            </View>
            <Tag label={plan.data.status} tone={plan.data.status === 'APPROVED' ? 'success' : 'neutral'} />
          </View>

          <Card style={styles.section}>
            <AppText variant="heading">Plan details</AppText>
            <DetailRow label="Start" value={<AppText variant="body">{formatDate(plan.data.period_start)}</AppText>} />
            <DetailRow label="End" value={<AppText variant="body">{formatDate(plan.data.period_end)}</AppText>} />
            <DetailRow
              label="Employee"
              value={<AppText variant="body">{labelFor(plan.data.employee_id, 'EMPLOYEE')}</AppText>}
            />
            {plan.data.comments ? (
              <DetailRow label="Comments" value={<AppText variant="body">{plan.data.comments}</AppText>} />
            ) : null}
          </Card>

          <Card style={styles.section}>
            <AppText variant="heading">Line items</AppText>
            {lines.isPending ? <LoadingState message="Loading line items…" /> : null}
            {lines.isError ? (
              <ErrorState
                message={lines.error instanceof Error ? lines.error.message : 'Unable to load line items'}
                onRetry={() => void lines.refetch()}
              />
            ) : null}
            {lines.data && lines.data.length === 0 ? (
              <EmptyState title="No line items" message="Add the first planned customer visit." />
            ) : null}
            {lines.data?.map((line) => (
              <View key={line.id} style={styles.line}>
                <AppText variant="section">{labelFor(line.entity_id, line.entity_type)}</AppText>
                <AppText variant="caption" tone="muted">
                  {formatDate(line.planned_date)} · {line.entity_type.toLowerCase()} · {line.status}
                </AppText>
              </View>
            ))}
          </Card>

          {plan.data.status === 'DRAFT' && can(user, appPermissions.tourCreate) ? (
            <Card style={styles.section}>
              <AppText variant="heading">Add line item</AppText>
              <View style={styles.spaced}>
                <Segmented
                  value={entityType}
                  onChange={(value) => {
                    setEntityType(value);
                    setEntityId(undefined);
                  }}
                  options={[
                    { label: 'Doctor', value: 'DOCTOR' },
                    { label: 'Chemist', value: 'CHEMIST' },
                    { label: 'Stockist', value: 'STOCKIST' }
                  ]}
                />
              </View>
              <View style={styles.spaced}>
                <TextField label="Planned date (YYYY-MM-DD)" value={plannedDate} onChangeText={setPlannedDate} />
              </View>
              <View style={styles.spaced}>
                <TextField label="Search customer" value={search} onChangeText={setSearch} />
              </View>
              {parties.isPending ? <LoadingState message="Loading customers…" /> : null}
              {parties.data ? (
                <View style={styles.spaced}>
                  <ChoiceList choices={parties.data} selectedId={entityId} onSelect={setEntityId} />
                </View>
              ) : null}
              <View style={styles.spaced}>
                <TextField label="Purpose" value={purpose} onChangeText={setPurpose} />
              </View>
              <View style={styles.spaced}>
                <Button
                  label={addLine.isPending ? 'Adding…' : 'Add line'}
                  onPress={handleAddLine}
                  loading={addLine.isPending}
                />
              </View>
            </Card>
          ) : null}

          <View style={styles.actions}>
            {plan.data.status === 'DRAFT' && can(user, appPermissions.tourCreate) ? (
              <Button
                label={submit.isPending ? 'Submitting…' : 'Submit plan'}
                onPress={() => submit.mutate(id)}
                loading={submit.isPending}
              />
            ) : null}
            {plan.data.status === 'SUBMITTED' && can(user, appPermissions.tourApprove) ? (
              <>
                <Button
                  label={decide.isPending ? 'Updating…' : 'Approve plan'}
                  onPress={() => decide.mutate(true)}
                  loading={decide.isPending}
                />
                <SecondaryButton
                  label="Reject plan"
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
  line: {
    borderBottomWidth: 1,
    borderBottomColor: colors.borderSoft,
    gap: spacing.xs,
    paddingVertical: spacing.md
  },
  actions: {
    gap: spacing.md,
    marginBottom: spacing.lg
  }
});
