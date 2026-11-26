import { randomUUID } from 'expo-crypto';
import { useState } from 'react';
import { Alert, Button, StyleSheet, Text, TextInput, View } from 'react-native';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';

import { login } from '../api/auth';
import type { RootStackParamList } from '../navigation';
import { clearTenantData } from '../storage/database';
import { getUser, saveSession } from '../storage/secure';
import type { SessionUser } from '../types';

type Props = NativeStackScreenProps<RootStackParamList, 'Login'> & {
  onSignedIn: (user: SessionUser) => void;
};

export default function LoginScreen({ onSignedIn }: Props) {
  const [tenantCode, setTenantCode] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    try {
      const current = await getUser();
      const response = await login(tenantCode.trim().toUpperCase(), email.trim(), password, randomUUID());
      if (current && current.tenant_id !== response.user.tenant_id) {
        await clearTenantData(current.tenant_id);
      }
      await saveSession(response.tokens, response.user);
      onSignedIn(response.user);
    } catch (error) {
      Alert.alert('Login failed', error instanceof Error ? error.message : 'Unable to sign in');
    } finally {
      setBusy(false);
    }
  }

  return (
    <View style={styles.container}>
      <Text style={styles.title}>PharmaOS</Text>
      <TextInput
        style={styles.input}
        placeholder="Tenant code"
        autoCapitalize="characters"
        value={tenantCode}
        onChangeText={setTenantCode}
      />
      <TextInput
        style={styles.input}
        placeholder="Email"
        autoCapitalize="none"
        keyboardType="email-address"
        value={email}
        onChangeText={setEmail}
      />
      <TextInput
        style={styles.input}
        placeholder="Password"
        secureTextEntry
        value={password}
        onChangeText={setPassword}
      />
      <Button title={busy ? 'Signing in…' : 'Sign in'} onPress={submit} disabled={busy} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, justifyContent: 'center', padding: 24, gap: 12, backgroundColor: '#f7f9fc' },
  title: { fontSize: 32, fontWeight: '700', textAlign: 'center', marginBottom: 24, color: '#16324f' },
  input: {
    borderWidth: 1,
    borderColor: '#cbd5e1',
    borderRadius: 8,
    backgroundColor: '#fff',
    padding: 12,
    fontSize: 16,
  },
});
