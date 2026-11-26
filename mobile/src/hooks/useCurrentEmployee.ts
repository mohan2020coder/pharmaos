import { useQuery } from '@tanstack/react-query';

import { ApiError } from '../api/client';
import { getCurrentEmployee } from '../api/domain';
import { queryKeys } from '../lib/queryKeys';
import { useAuthStore } from '../stores/auth';
import type { Employee } from '../types';

export function useCurrentEmployee() {
  const user = useAuthStore((state) => state.user);

  return useQuery<Employee | null>({
    queryKey: [...queryKeys.employees, 'me', user?.id],
    queryFn: async () => {
      try {
        return await getCurrentEmployee();
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    },
    enabled: Boolean(user),
    staleTime: 300000
  });
}
