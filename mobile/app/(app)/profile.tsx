import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Alert, StyleSheet, View } from 'react-native';

import { changePassword, sessions } from '../../src/api/auth';
import {
  AppText,
  Button,
  Card,
  DetailRow,
  ErrorState,
  LoadingState,
  Screen,
  SecondaryButton,
  Tag,
  TextField
} from '../../src/components/ui';
import { useCurrentEmployee } from '../../src/hooks/useCurrentEmployee';
import { queryKeys } from '../../src/lib/queryKeys';
import { useAuthStore } from '../../src/stores/auth';
import { spacing } from '../../src/theme/tokens';

export default function ProfileScreen() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const user = useAuthStore((state) => state.user);
  const signOut = useAuthStore((state) => state.signOut);
  const employee = useCurrentEmployee();
  const sessionsQuery = useQuery({
    queryKey: queryKeys.sessions,
    queryFn: sessions,
    staleTime: 30000
  });
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');

  const passwordMutation = useMutation({
    mutationFn: () => changePassword(currentPassword, newPassword),
    onSuccess: () => {
      setCurrentPassword('');
      setNewPassword('');
      Alert.alert('Password updated');
    },
    onError: (error) => {
      Alert.alert('Unable to change password', error instanceof Error ? error.message : 'Please try again.');
    }
  });

  async function handleSignOut() {
    await signOut();
    router.replace('/login');
  }

  function handleChangePassword() {
    if (!currentPassword || newPassword.length < 10) {
      Alert.alert('Password too short', 'New passwords must contain at least 10 characters.');
      return;
    }
    passwordMutation.mutate();
  }

  return (
    <Screen scroll>
      <AppText variant="title" tone="primaryDark">
        Profile
      </AppText>
      <AppText variant="body" tone="muted" style={styles.intro}>
        Account, access and session management.
      </AppText>

      <Card style={styles.section}>
        <AppText variant="heading">Account</AppText>
        <DetailRow
          label="Name"
          value={
            <AppText variant="body">
              {employee.data
                ? `${employee.data.first_name} ${employee.data.last_name}`.trim()
                : employee.isPending
                  ? 'Loading…'
                  : user?.email ?? '—'}
            </AppText>
          }
        />
        <DetailRow
          label="Employee code"
          value={<AppText variant="body">{employee.data?.employee_code ?? '—'}</AppText>}
        />
        <DetailRow label="Email" value={<AppText variant="body">{user?.email ?? '—'}</AppText>} />
        <DetailRow
          label="Access"
          value={
            <AppText variant="body">
              {user?.data_scope
                ? user.data_scope.charAt(0) + user.data_scope.slice(1).toLowerCase()
                : '—'}
            </AppText>
          }
        />
        <View style={styles.tags}>
          {(user?.roles ?? []).map((role) => (
            <Tag key={role} label={role} tone={role === 'SUPER_ADMIN' ? 'primary' : 'neutral'} />
          ))}
        </View>
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Permissions</AppText>
        <AppText variant="caption" tone="muted">
          {user?.permissions.length ?? 0} granted
        </AppText>
        <View style={styles.tags}>
          {(user?.permissions ?? []).map((permission) => (
            <Tag key={permission} label={permission} />
          ))}
        </View>
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Sessions</AppText>
        {sessionsQuery.isPending ? <LoadingState message="Loading sessions…" /> : null}
        {sessionsQuery.isError ? (
          <ErrorState
            message={
              sessionsQuery.error instanceof Error
                ? sessionsQuery.error.message
                : 'Unable to load sessions'
            }
            onRetry={() => void sessionsQuery.refetch()}
          />
        ) : null}
        {sessionsQuery.data ? (
          <DetailRow
            label="Active sessions"
            value={<AppText variant="body">{sessionsQuery.data.length}</AppText>}
          />
        ) : null}
      </Card>

      <Card style={styles.section}>
        <AppText variant="heading">Change password</AppText>
        <View style={styles.spaced}>
          <TextField
            label="Current password"
            value={currentPassword}
            onChangeText={setCurrentPassword}
            secureTextEntry
          />
        </View>
        <View style={styles.spaced}>
          <TextField
            label="New password"
            value={newPassword}
            onChangeText={setNewPassword}
            secureTextEntry
            helper="Minimum 10 characters"
          />
        </View>
        <View style={styles.spaced}>
          <Button
            label={passwordMutation.isPending ? 'Updating…' : 'Update password'}
            onPress={handleChangePassword}
            loading={passwordMutation.isPending}
          />
        </View>
      </Card>

      <View style={styles.section}>
        <SecondaryButton label="Sign out" onPress={handleSignOut} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  intro: {
    marginTop: spacing.sm
  },
  section: {
    marginTop: spacing.lg
  },
  spaced: {
    marginTop: spacing.md
  },
  tags: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: spacing.sm,
    marginTop: spacing.md
  }
});
