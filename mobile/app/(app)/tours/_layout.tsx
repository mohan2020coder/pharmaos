import { Stack } from 'expo-router';

import { PermissionGate } from '../../../src/components/auth/PermissionGate';
import { appPermissions } from '../../../src/lib/permissions';

export default function ToursLayout() {
  return (
    <PermissionGate permission={appPermissions.tourView}>
      <Stack
        screenOptions={{
          contentStyle: { backgroundColor: '#F7F9FC' },
          headerBackTitle: 'Back'
        }}
      >
        <Stack.Screen name="index" options={{ title: 'Tour plans' }} />
        <Stack.Screen name="new" options={{ title: 'New tour plan' }} />
        <Stack.Screen name="[id]" options={{ title: 'Tour plan details' }} />
      </Stack>
    </PermissionGate>
  );
}
