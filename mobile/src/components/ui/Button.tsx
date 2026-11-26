import { ActivityIndicator, Pressable, StyleSheet } from 'react-native';

import { colors, radius, spacing, typography } from '../../theme/tokens';
import { AppText } from './AppText';

type ButtonProps = {
  label: string;
  onPress: () => void;
  loading?: boolean;
  disabled?: boolean;
};

export function Button({ label, onPress, loading = false, disabled = false }: ButtonProps) {
  const unavailable = disabled || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: unavailable, busy: loading }}
      disabled={unavailable}
      onPress={onPress}
      style={({ pressed }) => [
        styles.button,
        pressed && !unavailable ? styles.pressed : null,
        unavailable ? styles.unavailable : null
      ]}
    >
      {loading ? (
        <ActivityIndicator color={colors.inverse} />
      ) : (
        <AppText variant="button" tone="inverse" style={styles.label}>
          {label}
        </AppText>
      )}
    </Pressable>
  );
}

const secondary = StyleSheet.create({
  button: {
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    paddingHorizontal: spacing.lg
  },
  pressed: {
    backgroundColor: colors.background
  },
  unavailable: {
    opacity: 0.6
  },
  label: {
    color: colors.text
  }
});

export function SecondaryButton({ label, onPress, loading = false, disabled = false }: ButtonProps) {
  const unavailable = disabled || loading;

  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: unavailable, busy: loading }}
      disabled={unavailable}
      onPress={onPress}
      style={({ pressed }) => [
        secondary.button,
        pressed && !unavailable ? secondary.pressed : null,
        unavailable ? secondary.unavailable : null
      ]}
    >
      {loading ? (
        <ActivityIndicator color={colors.text} />
      ) : (
        <AppText variant="button" style={secondary.label}>
          {label}
        </AppText>
      )}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  button: {
    minHeight: 48,
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: radius.md,
    backgroundColor: colors.primary,
    paddingHorizontal: spacing.lg,
    ...typography.button
  },
  pressed: {
    backgroundColor: colors.primaryDark
  },
  unavailable: {
    opacity: 0.6
  },
  label: {
    color: colors.inverse
  }
});
