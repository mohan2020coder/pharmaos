import { zodResolver } from '@hookform/resolvers/zod';
import { useLocalSearchParams, useRouter } from 'expo-router';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { KeyboardAvoidingView, Platform, StyleSheet, View } from 'react-native';

import { resetPassword } from '../src/api/auth';
import { AppText, Button, Card, Screen, SecondaryButton, TextField } from '../src/components/ui';
import { resetPasswordSchema, type ResetPasswordFormValues } from '../src/lib/validation';
import { colors, radius, spacing } from '../src/theme/tokens';

export default function ResetPasswordScreen() {
  const router = useRouter();
  const params = useLocalSearchParams<{ token?: string | string[] }>();
  const initialToken = typeof params.token === 'string' ? params.token : '';
  const [formError, setFormError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting }
  } = useForm<ResetPasswordFormValues>({
    resolver: zodResolver(resetPasswordSchema),
    mode: 'onBlur',
    defaultValues: { token: initialToken, newPassword: '', confirmPassword: '' }
  });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await resetPassword(values.token, values.newPassword);
      setDone(true);
    } catch (error) {
      setFormError(error instanceof Error ? error.message : 'Unable to reset password');
    }
  });

  return (
    <Screen scroll>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboard}
      >
        <View style={styles.header}>
          <AppText variant="display" tone="primaryDark">
            Choose a new password
          </AppText>
          <AppText variant="body" tone="muted">
            Enter the reset token from your email and a new password of at least 10 characters.
          </AppText>
        </View>

        <Card>
          {done ? (
            <View accessibilityRole="alert" style={styles.successBanner}>
              <AppText variant="body" tone="success">
                Password updated. Active sessions have been revoked.
              </AppText>
            </View>
          ) : null}

          {formError ? (
            <View accessibilityRole="alert" style={styles.errorBanner}>
              <AppText variant="body" tone="danger">
                {formError}
              </AppText>
            </View>
          ) : null}

          <Controller
            control={control}
            name="token"
            render={({ field }) => (
              <TextField
                label="Reset token"
                placeholder="Paste the token from your email"
                autoCapitalize="none"
                autoCorrect={false}
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.token?.message}
                editable={!isSubmitting && !done}
              />
            )}
          />

          <Controller
            control={control}
            name="newPassword"
            render={({ field }) => (
              <TextField
                label="New password"
                placeholder="At least 10 characters"
                secureTextEntry
                textContentType="newPassword"
                autoComplete="password-new"
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.newPassword?.message}
                editable={!isSubmitting && !done}
              />
            )}
          />

          <Controller
            control={control}
            name="confirmPassword"
            render={({ field }) => (
              <TextField
                label="Confirm new password"
                placeholder="Repeat the new password"
                secureTextEntry
                textContentType="newPassword"
                autoComplete="password-new"
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.confirmPassword?.message}
                editable={!isSubmitting && !done}
              />
            )}
          />

          {!done ? (
            <Button
              label={isSubmitting ? 'Updating…' : 'Update password'}
              onPress={onSubmit}
              loading={isSubmitting}
              disabled={isSubmitting}
            />
          ) : null}
          <SecondaryButton
            label="Back to sign in"
            onPress={() => router.replace('/login')}
            disabled={isSubmitting}
          />
        </Card>
      </KeyboardAvoidingView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  keyboard: {
    gap: spacing.xl
  },
  header: {
    gap: spacing.sm,
    paddingTop: spacing.xxl
  },
  successBanner: {
    borderRadius: radius.md,
    backgroundColor: colors.successSoft,
    borderWidth: 1,
    borderColor: colors.success,
    padding: spacing.md,
    marginBottom: spacing.md
  },
  errorBanner: {
    borderRadius: radius.md,
    backgroundColor: colors.dangerSoft,
    borderWidth: 1,
    borderColor: colors.danger,
    padding: spacing.md,
    marginBottom: spacing.md
  }
});
