import * as Location from 'expo-location';
import { useState } from 'react';
import { Alert, Button, StyleSheet, Text, TextInput, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';

import type { RootStackParamList } from '../navigation';
import { createDraftVisit } from '../storage/visits';

type Props = NativeStackScreenProps<RootStackParamList, 'VisitForm'>;

const ENTITY_TYPES = ['DOCTOR', 'CHEMIST', 'STOCKIST'] as const;

export default function VisitFormScreen({ navigation }: Props) {
  const [employeeId, setEmployeeId] = useState('');
  const [entityType, setEntityType] = useState<(typeof ENTITY_TYPES)[number]>('DOCTOR');
  const [entityId, setEntityId] = useState('');
  const [remarks, setRemarks] = useState('');
  const [busy, setBusy] = useState(false);

  async function save() {
    setBusy(true);
    try {
      let latitude: number | undefined;
      let longitude: number | undefined;
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.granted) {
        const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
        latitude = position.coords.latitude;
        longitude = position.coords.longitude;
      }
      await createDraftVisit({
        employeeId: employeeId.trim(),
        entityType,
        entityId: entityId.trim(),
        remarks: remarks.trim(),
        latitude,
        longitude,
      });
      Alert.alert('Visit saved', 'Queued for sync. Open Sync now when you are online.', [
        { text: 'OK', onPress: () => navigation.goBack() },
      ]);
    } catch (error) {
      Alert.alert('Save failed', error instanceof Error ? error.message : 'Unable to save visit');
    } finally {
      setBusy(false);
    }
  }

  return (
    <View style={styles.container}>
      <Text style={styles.title}>New visit</Text>
      <Text style={styles.label}>Employee UUID</Text>
      <TextInput style={styles.input} value={employeeId} onChangeText={setEmployeeId} autoCapitalize="none" />
      <Text style={styles.label}>Entity type</Text>
      <View style={styles.row}>
        {ENTITY_TYPES.map((type) => (
          <Button
            key={type}
            title={type}
            color={entityType === type ? '#2563eb' : '#64748b'}
            onPress={() => setEntityType(type)}
          />
        ))}
      </View>
      <Text style={styles.label}>Entity UUID</Text>
      <TextInput style={styles.input} value={entityId} onChangeText={setEntityId} autoCapitalize="none" />
      <Text style={styles.label}>Remarks</Text>
      <TextInput style={styles.input} value={remarks} onChangeText={setRemarks} multiline />
      <Button title={busy ? 'Saving…' : 'Save visit'} onPress={save} disabled={busy} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 24, gap: 8, backgroundColor: '#f7f9fc' },
  title: { fontSize: 26, fontWeight: '700', marginBottom: 12, color: '#16324f' },
  label: { fontSize: 12, color: '#64748b', marginTop: 8 },
  input: { borderWidth: 1, borderColor: '#cbd5e1', borderRadius: 8, backgroundColor: '#fff', padding: 12 },
  row: { flexDirection: 'row', justifyContent: 'space-between', gap: 4 },
});
