import { zodResolver } from '@hookform/resolvers/zod';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { KeyboardAvoidingView, Platform, StyleSheet, View } from 'react-native';

import { forgotPassword } from '../src/api/auth';
import { AppText, Button, Card, Screen, SecondaryButton, TextField } from '../src/components/ui';
import {
  forgotPasswordSchema,
  type ForgotPasswordFormValues
} from '../src/lib/validation';
import { colors, radius, spacing } from '../src/theme/tokens';

export default function ForgotPasswordScreen() {
  const router = useRouter();
  const [formError, setFormError] = useState<string | null>(null);
  const [confirmation, setConfirmation] = useState<string | null>(null);
  const {
    control,
    handleSubmit,
    formState: { errors, isSubmitting }
  } = useForm<ForgotPasswordFormValues>({
    resolver: zodResolver(forgotPasswordSchema),
    mode: 'onBlur',
    defaultValues: { email: '' }
  });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    setConfirmation(null);
    try {
      const result = await forgotPassword(values.email);
      setConfirmation(result.message || 'If the account exists, reset instructions have been sent.');
    } catch (error) {
      setFormError(error instanceof Error ? error.message : 'Unable to request a reset link');
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
            Reset password
          </AppText>
          <AppText variant="body" tone="muted">
            We will send reset instructions to your work email if an account exists.
          </AppText>
        </View>

        <Card>
          {confirmation ? (
            <View accessibilityRole="alert" style={styles.successBanner}>
              <AppText variant="body" tone="success">
                {confirmation}
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
            name="email"
            render={({ field }) => (
              <TextField
                label="Work email"
                placeholder="name@company.com"
                autoCapitalize="none"
                autoCorrect={false}
                keyboardType="email-address"
                textContentType="emailAddress"
                autoComplete="email"
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.email?.message}
                editable={!isSubmitting}
              />
            )}
          />

          <Button
            label={isSubmitting ? 'Sending…' : 'Send reset instructions'}
            onPress={onSubmit}
            loading={isSubmitting}
            disabled={isSubmitting}
          />
          <SecondaryButton label="Back to sign in" onPress={() => router.replace('/login')} />
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
