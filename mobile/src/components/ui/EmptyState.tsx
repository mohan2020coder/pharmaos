import { StyleSheet, View } from 'react-native';

import { spacing } from '../../theme/tokens';
import { AppText } from './AppText';

type EmptyStateProps = {
  title: string;
  message: string;
};

export function EmptyState({ title, message }: EmptyStateProps) {
  return (
    <View style={styles.container}>
      <AppText variant="section">{title}</AppText>
      <AppText variant="body" tone="muted" style={styles.message}>
        {message}
      </AppText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    paddingVertical: spacing.xxl
  },
  message: {
    marginTop: spacing.sm,
    textAlign: 'center'
  }
});
