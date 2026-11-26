import { randomUUID } from 'expo-crypto';
import * as SecureStore from 'expo-secure-store';

const DEVICE_ID_KEY = 'pharmaos.device_id.v1';

export async function getDeviceId(): Promise<string> {
  const existing = await SecureStore.getItemAsync(DEVICE_ID_KEY);
  if (existing) {
    return existing;
  }
  const created = randomUUID();
  await SecureStore.setItemAsync(DEVICE_ID_KEY, created);
  return created;
}
