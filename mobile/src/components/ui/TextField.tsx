import { useState } from 'react';
import { Pressable, StyleSheet, TextInput, View, type TextInputProps } from 'react-native';

import { colors, radius, spacing, typography } from '../../theme/tokens';
import { AppText } from './AppText';

type TextFieldProps = Omit<TextInputProps, 'style'> & {
  label: string;
  error?: string;
  helper?: string;
};

export function TextField({ label, error, helper, secureTextEntry, ...props }: TextFieldProps) {
  const [visible, setVisible] = useState(false);
  const isSecure = Boolean(secureTextEntry) && !visible;
  const message = error ?? helper;

  return (
    <View style={styles.field}>
      <AppText variant="caption" tone="muted">
        {label}
      </AppText>
      <View style={[styles.inputRow, error ? styles.inputRowError : null]}>
        <TextInput
          {...props}
          accessibilityLabel={label}
          accessibilityHint={error ?? helper}
          secureTextEntry={isSecure}
          placeholderTextColor={colors.muted}
          style={styles.input}
        />
        {secureTextEntry ? (
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={visible ? 'Hide password' : 'Show password'}
            onPress={() => setVisible((current) => !current)}
            style={styles.visibilityToggle}
          >
            <AppText variant="caption" tone="primary">
              {visible ? 'Hide' : 'Show'}
            </AppText>
          </Pressable>
        ) : null}
      </View>
      {message ? (
        <AppText variant="caption" tone={error ? 'danger' : 'muted'}>
          {message}
        </AppText>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  field: {
    gap: spacing.sm
  },
  inputRow: {
    flexDirection: 'row',
    alignItems: 'center',
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
    minHeight: 48
  },
  inputRowError: {
    borderColor: colors.danger,
    backgroundColor: colors.dangerSoft
  },
  input: {
    flex: 1,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md,
    color: colors.text,
    fontSize: typography.body.fontSize,
    fontWeight: typography.body.fontWeight
  },
  visibilityToggle: {
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  }
});
