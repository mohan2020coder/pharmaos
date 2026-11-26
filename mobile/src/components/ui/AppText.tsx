import { Text, type TextProps } from 'react-native';

import { colors, typography } from '../../theme/tokens';

type Variant = keyof typeof typography;
type Tone = 'default' | 'muted' | 'primary' | 'primaryDark' | 'danger' | 'success' | 'inverse';

type AppTextProps = TextProps & {
  variant?: Variant;
  tone?: Tone;
};

const toneStyles = {
  default: { color: colors.text },
  muted: { color: colors.muted },
  primary: { color: colors.primary },
  primaryDark: { color: colors.primaryDark },
  danger: { color: colors.danger },
  success: { color: colors.success },
  inverse: { color: colors.inverse }
} as const;

export function AppText({ variant = 'body', tone = 'default', style, ...props }: AppTextProps) {
  return <Text {...props} style={[typography[variant], toneStyles[tone], style]} />;
}
