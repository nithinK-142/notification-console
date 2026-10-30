import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from './api';
import type { NotificationStatus } from './types';

export const queryKeys = {
  notifications: ['notifications'] as const,
  list: (params: { limit: number; offset: number; q?: string; status?: NotificationStatus }) =>
    ['notifications', 'list', params] as const,
  details: (id: number) => ['notifications', 'details', id] as const,
  batches: (id: number) => ['notifications', 'batches', id] as const,
  stats: ['dashboard', 'stats'] as const,
  recent: ['dashboard', 'recent'] as const,
};

export function useNotifications(params: { limit: number; offset: number; q?: string; status?: NotificationStatus }) {
  return useQuery({
    queryKey: queryKeys.list(params),
    queryFn: () => api.notifications(params),
    placeholderData: (previous) => previous,
  });
}

export function useNotification(id: number | null) {
  return useQuery({
    queryKey: queryKeys.details(id ?? 0),
    queryFn: () => api.notification(id as number),
    enabled: id !== null,
  });
}

export function useNotificationBatches(id: number | null) {
  return useInfiniteQuery({
    queryKey: queryKeys.batches(id ?? 0),
    queryFn: ({ pageParam }) => api.batches(id as number, pageParam),
    initialPageParam: undefined as number | undefined,
    getNextPageParam: (lastPage) => lastPage.next_before ?? undefined,
    enabled: id !== null,
  });
}

export function useCreateNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.createNotification,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.notifications }),
        queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      ]);
    },
  });
}

export function useDeleteNotification() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.deleteNotification,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.notifications }),
        queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      ]);
    },
  });
}

export function useNotificationStats() {
  return useQuery({ queryKey: queryKeys.stats, queryFn: api.stats });
}

export function useRecentNotifications() {
  return useQuery({
    queryKey: queryKeys.recent,
    queryFn: () => api.notifications({ limit: 5, offset: 0 }),
  });
}

export function useApiHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: api.health,
    retry: false,
    refetchInterval: 30_000,
  });
}
