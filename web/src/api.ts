import type {
  AdminSession,
  BatchList,
  NotificationDetails,
  NotificationList,
  NotificationStats,
  NotificationStatus,
  NotificationSort,
} from './types';

const baseURL = import.meta.env.VITE_API_URL?.trim().replace(/\/+$/, '');

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  if (!baseURL) throw new Error('VITE_API_URL is missing. Set it in web/.env.');

  const response = await fetch(`${baseURL}${path}`, {
    ...init,
    credentials: 'include',
    headers: {
      Accept: 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
    },
  });

  if (response.status === 204) return undefined as T;

  const data = (await response.json().catch(() => ({}))) as {
    error?: string;
  };
  if (response.status === 401 && !['/api/auth/login', '/api/auth/logout', '/api/auth/me'].includes(path)) {
    window.dispatchEvent(new Event('notification-console:auth-expired'));
  }
  if (!response.ok) {
    throw new ApiError(data.error || `Request failed (${response.status})`, response.status);
  }

  return data as T;
}

export const api = {
  health: () => request<{ status: string }>('/api/health'),

  login: (email: string, password: string) =>
    request<AdminSession>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  logout: () => request<{ status: string }>('/api/auth/logout', { method: 'POST' }),

  me: () => request<AdminSession>('/api/auth/me'),

  notifications: (params: { limit: number; offset: number; q?: string; status?: NotificationStatus; sort?: NotificationSort }) => {
    const search = new URLSearchParams({ limit: String(params.limit), offset: String(params.offset) });
    if (params.q) search.set('q', params.q);
    if (params.status) search.set('status', params.status);
    if (params.sort) search.set('sort', params.sort);
    return request<NotificationList>(`/api/notifications?${search.toString()}`);
  },

  notification: (id: number) => request<NotificationDetails>(`/api/notifications/${id}`),

  batches: (id: number, before?: number) => {
    const search = new URLSearchParams({ limit: '20' });
    if (before) search.set('before', String(before));
    return request<BatchList>(`/api/notifications/${id}/batches?${search.toString()}`);
  },

  createNotification: (input: { title: string; body: string }) =>
    request<{ id: number; status: NotificationStatus }>('/api/notifications', {
      method: 'POST',
      body: JSON.stringify(input),
    }),

  deleteNotification: (id: number) =>
    request<void>(`/api/notifications/${id}`, { method: 'DELETE' }),

  stats: async (): Promise<NotificationStats> => {
    const statuses: NotificationStatus[] = ['pending', 'processing', 'done', 'failed'];
    const [all, ...counts] = await Promise.all([
      api.notifications({ limit: 1, offset: 0 }),
      ...statuses.map((status) => api.notifications({ limit: 1, offset: 0, status })),
    ]);
    return {
      total: all.total,
      pending: counts[0].total,
      processing: counts[1].total,
      done: counts[2].total,
      failed: counts[3].total,
    };
  },
};
