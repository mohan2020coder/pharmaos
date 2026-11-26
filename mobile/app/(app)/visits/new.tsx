import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import { createVisit } from '../../../src/api/domain';
import {
  AppText,
  Button,
  Card,
  ChoiceList,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Segmented,
  TextField
} from '../../../src/components/ui';
import { useCurrentEmployee } from '../../../src/hooks/useCurrentEmployee';
import { usePartyChoices } from '../../../src/hooks/usePartyChoices';
import { getCurrentCoordinates } from '../../../src/lib/location';
import { toRFC3339, todayInput } from '../../../src/lib/format';
import { appPermissions, can } from '../../../src/lib/permissions';
import { useAuthStore } from '../../../src/stores/auth';
import { useSyncStore } from '../../../src/stores/sync';
import { createDraftVisit } from '../../../src/storage/visits';
import { spacing } from '../../../src/theme/tokens';
import type { EntityType } from '../../../src/types';

const entityOptions: { label: string; value: EntityType }[] = [
  { label: 'Doctor', value: 'DOCTOR' },
  { label: 'Chemist', value: 'CHEMIST' },
  { label: 'Stockist', value: 'STOCKIST' }
];

export default function NewVisitScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const employee = useCurrentEmployee();
  const [entityType, setEntityType] = useState<EntityType>('DOCTOR');
  const [search, setSearch] = useState('');
  const [entityId, setEntityId] = useState<string>();
  const [visitedDate, setVisitedDate] = useState(todayInput());
  const [visitedTime, setVisitedTime] = useState('09:00');
  const [remarks, setRemarks] = useState('');
  const [coordinates, setCoordinates] = useState<{ latitude: number; longitude: number; accuracy: number }>();
  const storeRun = useSyncStore((state) => state.run);

  const parties = usePartyChoices(entityType, search);

  const save = useMutation({
    mutationFn: createVisit,
    onSuccess: (visit) => {
      void queryClient.invalidateQueries({ queryKey: ['visits'] });
      router.push(`/visits/${visit.id}`);
    },
    onError: (error) => {
      Alert.alert('Unable to record visit', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  function handleEntityTypeChange(value: EntityType) {
    setEntityType(value);
    setEntityId(undefined);
  }

  async function handleLocation() {
    try {
      const position = await getCurrentCoordinates();
      if (!position) {
        Alert.alert(
          'Location unavailable',
          'Allow location access and try again. If the device has no GPS fix, enable location services or set an emulator location.'
        );
        return;
      }
      setCoordinates(position);
    } catch (error) {
      Alert.alert('Location unavailable', error instanceof Error ? error.message : 'Please try again.');
    }
  }

  function handleSubmit() {
    if (!employee.data) {
      Alert.alert('Employee profile required', 'No employee profile is linked to this account.');
      return;
    }
    if (!entityId) {
      Alert.alert('Customer required', 'Select the doctor, chemist or stockist you visited.');
      return;
    }
    const visitedAt = toRFC3339(visitedDate, visitedTime || '09:00');
    save.mutate({
      employee_id: employee.data.id,
      entity_type: entityType,
      entity_id: entityId,
      territory_id: employee.data.territory_id,
      visited_at: visitedAt,
      check_in_lat: coordinates?.latitude,
      check_in_lng: coordinates?.longitude,
      check_in_accuracy: coordinates?.accuracy,
      remarks: remarks.trim() || undefined
    });
  }

  async function handleQueueOffline() {
    if (!employee.data) {
      Alert.alert('Employee profile required', 'No employee profile is linked to this account.');
      return;
    }
    if (!entityId) {
      Alert.alert('Customer required', 'Select the doctor, chemist or stockist you visited.');
      return;
    }
    try {
      await createDraftVisit({
        employeeId: employee.data.id,
        entityType,
        entityId,
        remarks: remarks.trim(),
        latitude: coordinates?.latitude,
        longitude: coordinates?.longitude
      });
      void storeRun('manual');
      void queryClient.invalidateQueries({ queryKey: ['outbox-counts'] });
      void queryClient.invalidateQueries({ queryKey: ['outbox-problems'] });
      Alert.alert('Queued for sync', 'The visit is saved locally and will sync when a connection is available.', [
        { text: 'View sync center', onPress: () => router.replace('/sync') },
        { text: 'OK', style: 'cancel' }
      ]);
    } catch (error) {
      Alert.alert('Unable to queue visit', error instanceof Error ? error.message : 'Please try again.');
    }
  }

  return (
    <Screen scroll>
      <AppText variant="title" tone="primaryDark">
        Record visit
      </AppText>
      <AppText variant="body" tone="muted" style={styles.intro}>
        Capture a field check-in using live master data and the current bearer session.
      </AppText>

      <Card style={styles.section}>
        <AppText variant="section">Employee</AppText>
        {!can(user, appPermissions.visitCreate) ? (
          <ErrorState message="You do not have permission to create visits." />
        ) : null}
        {employee.isPending ? <LoadingState message="Finding employee profile…" /> : null}
        {employee.isError ? (
          <ErrorState
            message={employee.error instanceof Error ? employee.error.message : 'Unable to load employee profile'}
            onRetry={() => void employee.refetch()}
          />
        ) : null}
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
        <AppText variant="section">Customer</AppText>
        <View style={styles.spaced}>
          <Segmented value={entityType} onChange={handleEntityTypeChange} options={entityOptions} />
        </View>
        <View style={styles.spaced}>
          <TextField
            label="Search customer"
            value={search}
            onChangeText={setSearch}
            placeholder="Name, specialty or registration"
          />
        </View>
        {parties.isPending ? <LoadingState message="Loading customers…" /> : null}
        {parties.isError ? (
          <ErrorState
            message={parties.error instanceof Error ? parties.error.message : 'Unable to load customers'}
            onRetry={() => void parties.refetch()}
          />
        ) : null}
        {parties.data ? (
          <View style={styles.spaced}>
            <ChoiceList
              choices={parties.data}
              selectedId={entityId}
              onSelect={setEntityId}
              emptyMessage="No active customers matched this search."
            />
          </View>
        ) : null}
      </Card>

      <Card style={styles.section}>
        <AppText variant="section">Visit details</AppText>
        <View style={styles.spaced}>
          <TextField label="Visit date (YYYY-MM-DD)" value={visitedDate} onChangeText={setVisitedDate} />
        </View>
        <View style={styles.spaced}>
          <TextField label="Visit time (HH:mm)" value={visitedTime} onChangeText={setVisitedTime} />
        </View>
        <View style={styles.spaced}>
          <TextField
            label="Remarks"
            value={remarks}
            onChangeText={setRemarks}
            multiline
            placeholder="Purpose and outcome"
          />
        </View>
        <View style={styles.spaced}>
          <SecondaryButton label={coordinates ? 'Location captured' : 'Use current location'} onPress={handleLocation} />
        </View>
        {coordinates ? (
          <AppText variant="caption" tone="success" style={styles.spaced}>
            {coordinates.latitude.toFixed(5)}, {coordinates.longitude.toFixed(5)} · ±
            {Math.round(coordinates.accuracy)}m
          </AppText>
        ) : null}
      </Card>

      <View style={styles.actions}>
        <Button label={save.isPending ? 'Saving…' : 'Save visit'} onPress={handleSubmit} loading={save.isPending} />
        <SecondaryButton label="Queue offline" onPress={() => void handleQueueOffline()} />
      </View>
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
  },
  actions: {
    gap: spacing.md,
    marginTop: spacing.lg
  }
});
