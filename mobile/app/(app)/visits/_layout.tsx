import { Stack } from 'expo-router';

import { PermissionGate } from '../../../src/components/auth/PermissionGate';
import { appPermissions } from '../../../src/lib/permissions';

export default function VisitsLayout() {
  return (
    <PermissionGate permission={appPermissions.visitView}>
      <Stack
        screenOptions={{
          contentStyle: { backgroundColor: '#F7F9FC' },
          headerBackTitle: 'Back'
        }}
      >
        <Stack.Screen name="index" options={{ title: 'Visits' }} />
        <Stack.Screen name="new" options={{ title: 'Record visit' }} />
        <Stack.Screen name="[id]" options={{ title: 'Visit details' }} />
      </Stack>
    </PermissionGate>
  );
}
