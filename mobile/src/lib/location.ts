import * as Location from 'expo-location';

type Coordinates = {
  latitude: number;
  longitude: number;
  accuracy: number;
};

const CURRENT_POSITION_TIMEOUT_MS = 8000;
const LAST_KNOWN_MAX_AGE_MS = 10 * 60 * 1000;

function toCoordinates(position: Location.LocationObject): Coordinates {
  return {
    latitude: position.coords.latitude,
    longitude: position.coords.longitude,
    accuracy: position.coords.accuracy ?? 0
  };
}

async function withTimeout<T>(promise: Promise<T>, timeoutMs: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error('Location request timed out.')), timeoutMs);
      })
    ]);
  } finally {
    if (timer) {
      clearTimeout(timer);
    }
  }
}

export async function getCurrentCoordinates(): Promise<Coordinates | null> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (permission.status !== 'granted') {
    return null;
  }
  const lastKnown = await Location.getLastKnownPositionAsync({
    maxAge: LAST_KNOWN_MAX_AGE_MS,
    requiredAccuracy: 200
  }).catch(() => null);
  try {
    const position = await withTimeout(
      Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced }),
      CURRENT_POSITION_TIMEOUT_MS
    );
    return toCoordinates(position);
  } catch {
    return lastKnown ? toCoordinates(lastKnown) : null;
  }
}
