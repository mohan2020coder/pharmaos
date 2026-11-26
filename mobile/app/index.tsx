import { Redirect } from 'expo-router';

import { useAuthStore } from '../src/stores/auth';

export default function Index() {
  const status = useAuthStore((state) => state.status);
  return <Redirect href={status === 'authenticated' ? '/home' : '/login'} />;
}
