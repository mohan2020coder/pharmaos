import { StyleSheet, View } from 'react-native';

import { colors, radius, spacing } from '../../theme/tokens';
import { AppText } from './AppText';

type TagProps = {
  label: string;
  tone?: 'neutral' | 'primary' | 'success';
};

export function Tag({ label, tone = 'neutral' }: TagProps) {
  return (
    <View style={[styles.tag, toneStyles[tone]]}>
      <AppText variant="caption" tone={tone === 'neutral' ? 'muted' : tone}>
        {label}
      </AppText>
    </View>
  );
}

const toneStyles = {
  neutral: { backgroundColor: colors.background, borderColor: colors.borderSoft },
  primary: { backgroundColor: colors.primarySoft, borderColor: colors.primary },
  success: { backgroundColor: colors.successSoft, borderColor: colors.success }
} as const;

const styles = StyleSheet.create({
  tag: {
    borderRadius: radius.pill,
    borderWidth: 1,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs
  }
});
