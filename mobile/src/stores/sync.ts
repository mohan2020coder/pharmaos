import { create } from 'zustand';

import { useAuthStore } from './auth';
import { syncNow, type SyncSummary } from '../sync/engine';

export type SyncRunReason = 'manual' | 'startup' | 'connectivity' | 'foreground' | 'interval';

type SyncStore = {
  isOnline: boolean | null;
  isSyncing: boolean;
  lastRunAt: number | null;
  lastReason: SyncRunReason | null;
  lastSummary: SyncSummary | null;
  lastError: string | null;
  setOnline: (online: boolean) => void;
  run: (reason?: SyncRunReason) => Promise<SyncSummary | null>;
  clear: () => void;
};

let activeRun: Promise<SyncSummary | null> | null = null;

export const useSyncStore = create<SyncStore>((set, get) => ({
  isOnline: null,
  isSyncing: false,
  lastRunAt: null,
  lastReason: null,
  lastSummary: null,
  lastError: null,
  setOnline: (online) => {
    if (get().isOnline !== online) {
      set({ isOnline: online, lastError: online ? null : 'Offline' });
    }
  },
  run: (reason = 'manual') => {
    if (activeRun) {
      return activeRun;
    }
    if (get().isOnline === null) {
      return Promise.resolve(null);
    }
    if (get().isOnline === false) {
      set({ lastError: 'Offline', lastReason: reason });
      return Promise.resolve(null);
    }
    if (useAuthStore.getState().status !== 'authenticated') {
      return Promise.resolve(null);
    }
    set({ isSyncing: true, lastReason: reason, lastError: null });
    activeRun = syncNow()
      .then((summary) => {
        set({ lastSummary: summary, lastRunAt: Date.now(), lastError: null, isSyncing: false });
        return summary;
      })
      .catch((error: unknown) => {
        const message = error instanceof Error ? error.message : 'Sync failed';
        set({ lastError: message, lastRunAt: Date.now(), isSyncing: false });
        return null;
      })
      .finally(() => {
        activeRun = null;
      });
    return activeRun;
  },
  clear: () => set({ lastSummary: null, lastError: null, lastRunAt: null, lastReason: null })
}));
