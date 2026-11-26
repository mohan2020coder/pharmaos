import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import { createDcr } from '../../../src/api/domain';
import { AppText, Button, Card, ErrorState, LoadingState, Screen, TextField } from '../../../src/components/ui';
import { useCurrentEmployee } from '../../../src/hooks/useCurrentEmployee';
import { todayInput } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { useAuthStore } from '../../../src/stores/auth';
import { spacing } from '../../../src/theme/tokens';

export default function NewDcrScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const employee = useCurrentEmployee();
  const [reportDate, setReportDate] = useState(todayInput());

  const save = useMutation({
    mutationFn: () => createDcr(employee.data?.id as string, reportDate),
    onSuccess: (report) => {
      void queryClient.invalidateQueries({ queryKey: ['dcr'] });
      router.push(`/dcr/${report.id}`);
    },
    onError: (error) => {
      Alert.alert('Unable to create report', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleSubmit() {
    if (!employee.data) {
      Alert.alert('Employee profile required', 'No employee profile is linked to this account.');
      return;
    }
    save.mutate();
  }

  return (
    <Screen scroll>
      <AppText variant="title" tone="primaryDark">
        New daily call report
      </AppText>
      <AppText variant="body" tone="muted" style={styles.intro}>
        Reports collect product-level outcomes for a single field date.
      </AppText>

      <Card style={styles.section}>
        <AppText variant="section">Employee</AppText>
        {!can(user, appPermissions.dcrCreate) ? (
          <ErrorState message="You do not have permission to create reports." />
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
        <AppText variant="section">Report date</AppText>
        <View style={styles.spaced}>
          <TextField label="Date (YYYY-MM-DD)" value={reportDate} onChangeText={setReportDate} />
        </View>
      </Card>

      <Button label={save.isPending ? 'Creating…' : 'Create report'} onPress={handleSubmit} loading={save.isPending} />
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
