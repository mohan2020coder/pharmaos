import NetInfo from '@react-native-community/netinfo';
import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, type ReactNode } from 'react';
import { AppState, type AppStateStatus } from 'react-native';

import { useAuthStore } from '../stores/auth';
import { useSyncStore, type SyncRunReason } from '../stores/sync';

type SyncProviderProps = {
  children: ReactNode;
};

export function SyncProvider({ children }: SyncProviderProps) {
  const queryClient = useQueryClient();
  const authStatus = useAuthStore((state) => state.status);
  const setOnline = useSyncStore((state) => state.setOnline);
  const storeRun = useSyncStore((state) => state.run);

  const run = useCallback(
    (reason: SyncRunReason) => {
      void storeRun(reason).then((summary) => {
        if (summary) {
          void queryClient.invalidateQueries();
        }
      });
    },
    [queryClient, storeRun],
  );

  useEffect(() => {
    let mounted = true;

    void NetInfo.fetch().then((state) => {
      if (!mounted) {
        return;
      }
      const online = Boolean(state.isConnected);
      setOnline(online);
      if (online) {
        run('startup');
      }
    });

    const unsubscribeNetInfo = NetInfo.addEventListener((state) => {
      const online = Boolean(state.isConnected);
      setOnline(online);
      if (online) {
        run('connectivity');
      }
    });

    const handleAppState = (nextState: AppStateStatus) => {
      if (nextState === 'active') {
        run('foreground');
      }
    };
    const appStateSubscription = AppState.addEventListener('change', handleAppState);
    const interval = setInterval(() => run('interval'), 60000);

    return () => {
      mounted = false;
      unsubscribeNetInfo();
      appStateSubscription.remove();
      clearInterval(interval);
    };
  }, [run, setOnline]);

  useEffect(() => {
    if (authStatus === 'authenticated') {
      run('startup');
    }
  }, [authStatus, run]);

  return <>{children}</>;
}
