import { Redirect } from 'expo-router';
import type { ReactNode } from 'react';

import { can, canAny } from '../../lib/permissions';
import { useAuthStore } from '../../stores/auth';

type PermissionGateProps = {
  permission?: string;
  anyPermission?: string[];
  children: ReactNode;
};

export function PermissionGate({ permission, anyPermission, children }: PermissionGateProps) {
  const user = useAuthStore((state) => state.user);
  const allowed = permission
    ? can(user, permission)
    : anyPermission
      ? canAny(user, anyPermission)
      : true;

  if (!allowed) {
    return <Redirect href="/home" />;
  }

  return <>{children}</>;
}
