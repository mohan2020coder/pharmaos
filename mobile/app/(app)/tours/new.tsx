import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import { createTourPlan } from '../../../src/api/domain';
import { AppText, Button, Card, ErrorState, LoadingState, Screen, TextField } from '../../../src/components/ui';
import { useCurrentEmployee } from '../../../src/hooks/useCurrentEmployee';
import { todayInput } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { useAuthStore } from '../../../src/stores/auth';
import { spacing } from '../../../src/theme/tokens';

export default function NewTourPlanScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const employee = useCurrentEmployee();
  const [title, setTitle] = useState('');
  const [periodStart, setPeriodStart] = useState(todayInput());
  const [periodEnd, setPeriodEnd] = useState(todayInput());

  const save = useMutation({
    mutationFn: createTourPlan,
    onSuccess: (plan) => {
      void queryClient.invalidateQueries({ queryKey: ['tour-plans'] });
      router.push(`/tours/${plan.id}`);
    },
    onError: (error) => {
      Alert.alert('Unable to create tour plan', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleSubmit() {
    if (!employee.data) {
      Alert.alert('Employee profile required', 'No employee profile is linked to this account.');
      return;
    }
    if (title.trim().length < 2) {
      Alert.alert('Title required', 'Enter a title with at least two characters.');
      return;
    }
    save.mutate({
      employee_id: employee.data.id,
      title: title.trim(),
      period_start: periodStart,
      period_end: periodEnd
    });
  }

  return (
    <Screen scroll>
      <AppText variant="title" tone="primaryDark">
        New tour plan
      </AppText>
      <AppText variant="body" tone="muted" style={styles.intro}>
        Plans are created as drafts and can be submitted after adding line items.
      </AppText>

      <Card style={styles.section}>
        <AppText variant="section">Employee</AppText>
        {!can(user, appPermissions.tourCreate) ? (
          <ErrorState message="You do not have permission to create tour plans." />
        ) : null}
        {employee.isPending ? <LoadingState message="Finding employee profile…" /> : null}
        {employee.data ? (
          <AppText variant="body" tone="muted" style={styles.spaced}>
            {employee.data.first_name} {employee.data.last_name} · {employee.data.employee_code}
          </AppText>
        ) : !employee.isPending ? (
          <AppText variant="body" tone="danger" style={styles.spaced}>
            No employee profile linked to this account.
          </AppText>
        ) : null}
      </Card>

      <Card style={styles.section}>
        <AppText variant="section">Plan details</AppText>
        <View style={styles.spaced}>
          <TextField
            label="Title"
            value={title}
            onChangeText={setTitle}
            placeholder="Weekly territory plan"
          />
        </View>
        <View style={styles.spaced}>
          <TextField
            label="Period start (YYYY-MM-DD)"
            value={periodStart}
            onChangeText={setPeriodStart}
          />
        </View>
        <View style={styles.spaced}>
          <TextField
            label="Period end (YYYY-MM-DD)"
            value={periodEnd}
            onChangeText={setPeriodEnd}
          />
        </View>
      </Card>

      <Button label={save.isPending ? 'Creating…' : 'Create plan'} onPress={handleSubmit} loading={save.isPending} />
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
  }
});
