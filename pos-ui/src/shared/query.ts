import { QueryClient } from '@tanstack/vue-query';

import { ApiError } from './api';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry(failureCount, error) {
        if (failureCount >= 1) return false;
        if (error instanceof ApiError) {
          return error.category === 'network' || error.category === 'timeout' || error.category === 'server';
        }
        return false;
      },
      refetchOnWindowFocus: false,
      staleTime: 10_000,
    },
    mutations: {
      retry: false,
    },
  },
});
