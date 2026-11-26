import { ActivityIndicator, StyleSheet, View } from 'react-native';

import { colors, spacing } from '../../theme/tokens';
import { AppText } from './AppText';

type LoadingStateProps = {
  message?: string;
};

export function LoadingState({ message = 'Preparing PharmaOS…' }: LoadingStateProps) {
  return (
    <View style={styles.container}>
      <ActivityIndicator color={colors.primary} size="large" />
      <AppText variant="body" tone="muted" style={styles.message}>
        {message}
      </AppText>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    gap: spacing.lg,
    backgroundColor: colors.background
  },
  message: {
    textAlign: 'center'
  }
});
