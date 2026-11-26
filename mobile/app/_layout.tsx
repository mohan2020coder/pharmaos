import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { useEffect, useState } from 'react';
import { StyleSheet } from 'react-native';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { SafeAreaProvider } from 'react-native-safe-area-context';

import { setAuthLostHandler } from '../src/api/client';
import { LoadingState } from '../src/components/ui';
import { useAuthStore } from '../src/stores/auth';
import { SyncProvider } from '../src/sync/SyncProvider';

export default function RootLayout() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: { staleTime: 30000, retry: 1 },
          mutations: { retry: 0 }
        }
      })
  );
  const status = useAuthStore((state) => state.status);

  useEffect(() => {
    void useAuthStore.getState().bootstrap();
    setAuthLostHandler(() => useAuthStore.getState().markUnauthenticated());
    return () => setAuthLostHandler(null);
  }, []);

  return (
    <GestureHandlerRootView style={styles.root}>
      <QueryClientProvider client={queryClient}>
        <SafeAreaProvider>
          <SyncProvider>
            <StatusBar style="dark" />
            {status === 'loading' ? (
              <LoadingState />
            ) : (
              <Stack screenOptions={{ headerShown: false }}>
                <Stack.Screen name="index" />
                <Stack.Screen name="login" />
                <Stack.Screen name="forgot-password" />
                <Stack.Screen name="reset-password" />
                <Stack.Screen name="(app)" />
              </Stack>
            )}
          </SyncProvider>
        </SafeAreaProvider>
      </QueryClientProvider>
    </GestureHandlerRootView>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1
  }
});
