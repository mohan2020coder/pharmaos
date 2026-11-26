import * as SecureStore from 'expo-secure-store';

import type { SessionUser, TokenPair } from '../types';

const TOKENS_KEY = 'pharmaos.tokens.v1';
const USER_KEY = 'pharmaos.user.v1';

export async function saveSession(tokens: TokenPair, user: SessionUser): Promise<void> {
  await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
  await SecureStore.setItemAsync(USER_KEY, JSON.stringify(user));
}

export async function getTokens(): Promise<TokenPair | null> {
  const value = await SecureStore.getItemAsync(TOKENS_KEY);
  if (!value) {
    return null;
  }
  try {
    return JSON.parse(value) as TokenPair;
  } catch {
    await clearSession();
    return null;
  }
}

export async function saveTokens(tokens: TokenPair): Promise<void> {
  await SecureStore.setItemAsync(TOKENS_KEY, JSON.stringify(tokens));
}

export async function getUser(): Promise<SessionUser | null> {
  const value = await SecureStore.getItemAsync(USER_KEY);
  if (!value) {
    return null;
  }
  try {
    return JSON.parse(value) as SessionUser;
  } catch {
    await clearSession();
    return null;
  }
}

export async function clearSession(): Promise<void> {
  await SecureStore.deleteItemAsync(TOKENS_KEY);
  await SecureStore.deleteItemAsync(USER_KEY);
}
