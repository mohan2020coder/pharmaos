import { Stack } from 'expo-router';

import { PermissionGate } from '../../../src/components/auth/PermissionGate';
import { appPermissions } from '../../../src/lib/permissions';

export default function DcrLayout() {
  return (
    <PermissionGate permission={appPermissions.dcrView}>
      <Stack
        screenOptions={{
          contentStyle: { backgroundColor: '#F7F9FC' },
          headerBackTitle: 'Back'
        }}
      >
        <Stack.Screen name="index" options={{ title: 'Daily call reports' }} />
        <Stack.Screen name="new" options={{ title: 'New DCR' }} />
        <Stack.Screen name="[id]" options={{ title: 'DCR details' }} />
      </Stack>
    </PermissionGate>
  );
}
