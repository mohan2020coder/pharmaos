import { Redirect, Tabs } from 'expo-router';
import { StyleSheet, View } from 'react-native';

import { useAuthStore } from '../../src/stores/auth';
import { visibleTabs } from '../../src/navigation/appTabs';
import { AppText } from '../../src/components/ui';
import { colors, spacing } from '../../src/theme/tokens';

export default function AppGroupLayout() {
  const status = useAuthStore((state) => state.status);
  const user = useAuthStore((state) => state.user);

  if (status !== 'authenticated') {
    return <Redirect href="/login" />;
  }

  return (
    <Tabs
      backBehavior="history"
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.muted,
        tabBarStyle: styles.tabBar,
        tabBarLabelStyle: styles.tabBarLabel
      }}
    >
      {visibleTabs(user).map((tab) => (
        <Tabs.Screen
          key={tab.name}
          name={tab.name}
          options={{
            title: tab.title,
            tabBarIcon: ({ focused }) => (
              <View style={[styles.icon, focused ? styles.iconActive : null]}>
                <AppText variant="caption" tone={focused ? 'primary' : 'muted'}>
                  {tab.short}
                </AppText>
              </View>
            )
          }}
        />
      ))}
      <Tabs.Screen name="sync" options={{ href: null, title: 'Sync center' }} />
    </Tabs>
  );
}

const styles = StyleSheet.create({
  tabBar: {
    backgroundColor: colors.surface,
    borderTopColor: colors.borderSoft,
    paddingTop: spacing.sm,
    paddingBottom: spacing.sm,
    height: 68
  },
  tabBarLabel: {
    fontSize: 11,
    fontWeight: '600'
  },
  icon: {
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: 12,
    minWidth: 34,
    paddingHorizontal: spacing.xs,
    paddingVertical: 2
  },
  iconActive: {
    backgroundColor: colors.primarySoft
  }
});
