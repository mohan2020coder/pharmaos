import { zodResolver } from '@hookform/resolvers/zod';
import { useRouter } from 'expo-router';
import { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { KeyboardAvoidingView, Platform, Pressable, StyleSheet, View } from 'react-native';

import { ApiError } from '../src/api/client';
import { AppText, Button, Card, Screen, TextField } from '../src/components/ui';
import { loginSchema, type LoginFormValues } from '../src/lib/validation';
import { useAuthStore } from '../src/stores/auth';
import { colors, radius, spacing } from '../src/theme/tokens';

function mapServerField(field: string): keyof LoginFormValues | null {
  if (field === 'tenant_code' || field === 'tenantCode') return 'tenantCode';
  if (field === 'email') return 'email';
  if (field === 'password') return 'password';
  return null;
}

export default function LoginScreen() {
  const router = useRouter();
  const signIn = useAuthStore((state) => state.signIn);
  const [formError, setFormError] = useState<string | null>(null);
  const {
    control,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting }
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    mode: 'onBlur',
    defaultValues: { tenantCode: '', email: '', password: '' }
  });

  const onSubmit = handleSubmit(async (values) => {
    setFormError(null);
    try {
      await signIn(values);
      router.replace('/home');
    } catch (error) {
      if (error instanceof ApiError) {
        for (const [field, message] of Object.entries(error.fieldErrors)) {
          const target = mapServerField(field);
          if (target) {
            setError(target, { message });
          }
        }
      }
      setFormError(error instanceof Error ? error.message : 'Unable to sign in');
    }
  });

  return (
    <Screen scroll>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : 'height'}
        style={styles.keyboard}
      >
        <View style={styles.brand}>
          <View style={styles.brandMark}>
            <AppText variant="heading" tone="inverse">
              P
            </AppText>
          </View>
          <AppText variant="display" tone="primaryDark">
            PharmaOS
          </AppText>
          <AppText variant="body" tone="muted" style={styles.brandCopy}>
            Field execution and distribution, connected in one place.
          </AppText>
        </View>

        <Card>
          <AppText variant="title">
            Sign in
          </AppText>
          <AppText variant="body" tone="muted" style={styles.cardCopy}>
            Use your tenant code and work email.
          </AppText>

          {formError ? (
            <View accessibilityRole="alert" style={styles.errorBanner}>
              <AppText variant="body" tone="danger">
                {formError}
              </AppText>
            </View>
          ) : null}

          <Controller
            control={control}
            name="tenantCode"
            render={({ field }) => (
              <TextField
                label="Tenant code"
                placeholder="e.g. ACME"
                autoCapitalize="characters"
                autoCorrect={false}
                textContentType="organizationName"
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.tenantCode?.message}
                editable={!isSubmitting}
              />
            )}
          />

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

          <Controller
            control={control}
            name="password"
            render={({ field }) => (
              <TextField
                label="Password"
                placeholder="Enter your password"
                secureTextEntry
                textContentType="password"
                autoComplete="password"
                value={field.value}
                onChangeText={field.onChange}
                onBlur={field.onBlur}
                error={errors.password?.message}
                editable={!isSubmitting}
              />
            )}
          />

          <Button label="Sign in" onPress={onSubmit} loading={isSubmitting} disabled={isSubmitting} />
          <Pressable
            accessibilityRole="link"
            onPress={() => router.push('/forgot-password')}
            style={styles.forgotLink}
          >
            <AppText variant="caption" tone="primary">
              Forgot password?
            </AppText>
          </Pressable>
        </Card>

        <AppText variant="caption" tone="muted" style={styles.footer}>
          PharmaOS uses your tenant and role permissions to scope every request.
        </AppText>
      </KeyboardAvoidingView>
    </Screen>
  );
}

const styles = StyleSheet.create({
  keyboard: {
    gap: spacing.xl
  },
  brand: {
    gap: spacing.sm,
    paddingTop: spacing.xxl
  },
  brandMark: {
    width: 54,
    height: 54,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.lg,
    backgroundColor: colors.primary,
    marginBottom: spacing.md
  },
  brandCopy: {
    maxWidth: 320
  },
  cardCopy: {
    marginTop: spacing.xs
  },
  forgotLink: {
    alignSelf: 'center',
    paddingVertical: spacing.sm,
    paddingHorizontal: spacing.md
  },
  errorBanner: {
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.danger,
    backgroundColor: colors.dangerSoft,
    padding: spacing.md
  },
  footer: {
    textAlign: 'center'
  }
});
