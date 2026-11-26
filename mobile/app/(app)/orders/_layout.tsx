import { Stack } from 'expo-router';

import { PermissionGate } from '../../../src/components/auth/PermissionGate';
import { appPermissions } from '../../../src/lib/permissions';

export default function OrdersLayout() {
  return (
    <PermissionGate permission={appPermissions.orderView}>
      <Stack
        screenOptions={{
          contentStyle: { backgroundColor: '#F7F9FC' },
          headerBackTitle: 'Back'
        }}
      >
        <Stack.Screen name="index" options={{ title: 'Orders' }} />
        <Stack.Screen name="new" options={{ title: 'New order' }} />
        <Stack.Screen name="[id]" options={{ title: 'Order details' }} />
      </Stack>
    </PermissionGate>
  );
}
