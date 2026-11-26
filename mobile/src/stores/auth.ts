import { create } from 'zustand';

import { login as apiLogin, logout as apiLogout, me as apiMe } from '../api/auth';
import type { LoginFormValues } from '../lib/validation';
import { clearTenantData } from '../storage/database';
import { getDeviceId } from '../storage/device';
import { clearSession, getTokens, getUser, saveSession } from '../storage/secure';
import type { SessionUser } from '../types';

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated';

type AuthState = {
  status: AuthStatus;
  user: SessionUser | null;
  bootstrap: () => Promise<void>;
  signIn: (values: LoginFormValues) => Promise<void>;
  signOut: () => Promise<void>;
  markUnauthenticated: () => void;
};

async function enrichSessionUser(fallback: SessionUser): Promise<SessionUser> {
  try {
    return await apiMe();
  } catch {
    return fallback;
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  status: 'loading',
  user: null,

  bootstrap: async () => {
    const [tokens, user] = await Promise.all([getTokens(), getUser()]);
    if (!tokens || !user) {
      await clearSession();
      set({ status: 'unauthenticated', user: null });
      return;
    }
    const enriched = await enrichSessionUser(user);
    await saveSession(tokens, enriched);
    set({ status: 'authenticated', user: enriched });
  },

  signIn: async (values) => {
    const deviceId = await getDeviceId();
    const response = await apiLogin(
      values.tenantCode.trim().toUpperCase(),
      values.email.trim(),
      values.password,
      deviceId
    );
    const previous = get().user ?? (await getUser());
    if (previous && previous.tenant_id !== response.user.tenant_id) {
      await clearTenantData(previous.tenant_id);
    }
    await saveSession(response.tokens, response.user);
    const enriched = await enrichSessionUser(response.user);
    await saveSession(response.tokens, enriched);
    set({ status: 'authenticated', user: enriched });
  },

  signOut: async () => {
    const current = get().user ?? (await getUser());
    try {
      await apiLogout();
    } catch {
      void 0;
    }
    if (current) {
      await clearTenantData(current.tenant_id);
    }
    await clearSession();
    set({ status: 'unauthenticated', user: null });
  },

  markUnauthenticated: () => {
    set({ status: 'unauthenticated', user: null });
  }
}));
