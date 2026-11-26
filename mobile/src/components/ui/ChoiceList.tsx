import { Pressable, StyleSheet, View } from 'react-native';

import { colors, radius, spacing } from '../../theme/tokens';
import { AppText } from './AppText';
import { EmptyState } from './EmptyState';

export type Choice = {
  id: string;
  label: string;
  detail?: string;
};

type ChoiceListProps = {
  choices: Choice[];
  selectedId?: string;
  onSelect: (id: string) => void;
  emptyMessage?: string;
};

export function ChoiceList({ choices, selectedId, onSelect, emptyMessage }: ChoiceListProps) {
  if (choices.length === 0) {
    return <EmptyState title="No matches" message={emptyMessage ?? 'Try a different search term.'} />;
  }

  return (
    <View style={styles.list}>
      {choices.map((choice) => {
        const selected = choice.id === selectedId;
        return (
          <Pressable
            accessibilityRole="button"
            accessibilityState={{ selected }}
            key={choice.id}
            onPress={() => onSelect(choice.id)}
            style={({ pressed }) => [
              styles.option,
              selected ? styles.selected : null,
              pressed ? styles.pressed : null
            ]}
          >
            <AppText variant="section" tone={selected ? 'primary' : 'default'}>
              {choice.label}
            </AppText>
            {choice.detail ? (
              <AppText variant="caption" tone="muted" style={styles.detail}>
                {choice.detail}
              </AppText>
            ) : null}
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.sm
  },
  option: {
    borderWidth: 1,
    borderColor: colors.borderSoft,
    borderRadius: radius.md,
    backgroundColor: colors.surface,
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.md
  },
  selected: {
    borderColor: colors.primary,
    backgroundColor: colors.primarySoft
  },
  pressed: {
    opacity: 0.75
  },
  detail: {
    marginTop: spacing.xs
  }
});
