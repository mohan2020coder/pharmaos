import type { ReactNode } from 'react';
import { StyleSheet, View } from 'react-native';

import { spacing } from '../../theme/tokens';
import { AppText } from './AppText';

type DetailRowProps = {
  label: string;
  value: ReactNode;
};

export function DetailRow({ label, value }: DetailRowProps) {
  return (
    <View style={styles.row}>
      <AppText variant="caption" tone="muted">
        {label}
      </AppText>
      <View style={styles.value}>{value}</View>
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    gap: spacing.md,
    paddingVertical: spacing.xs
  },
  value: {
    flexShrink: 1,
    maxWidth: '60%'
  }
});
