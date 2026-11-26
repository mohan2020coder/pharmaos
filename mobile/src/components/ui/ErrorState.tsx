import { StyleSheet, View } from 'react-native';

import { colors, radius, spacing } from '../../theme/tokens';
import { AppText } from './AppText';
import { SecondaryButton } from './Button';

type ErrorStateProps = {
  message: string;
  onRetry?: () => void;
};

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  return (
    <View style={styles.container}>
      <AppText variant="body" tone="danger" style={styles.message}>
        {message}
      </AppText>
      {onRetry ? <SecondaryButton label="Retry" onPress={onRetry} /> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    backgroundColor: colors.dangerSoft,
    borderColor: colors.danger,
    borderRadius: radius.md,
    gap: spacing.md,
    padding: spacing.lg
  },
  message: {
    textAlign: 'center'
  }
});
