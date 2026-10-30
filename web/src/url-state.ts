import type { NotificationStatus } from './types';

export type ConsoleView = 'overview' | 'notifications';
export type ConsoleStatus = 'all' | NotificationStatus;

export interface ConsoleUrlState {
  view: ConsoleView;
  query: string;
  status: ConsoleStatus;
  page: number;
  notificationId: number | null;
}

const validStatuses: NotificationStatus[] = ['pending', 'processing', 'done', 'failed'];

export function readConsoleUrlState(): ConsoleUrlState {
  const params = new URLSearchParams(window.location.search);
  const rawStatus = params.get('status');
  const rawPage = Number(params.get('page') ?? 1);
  const rawNotificationId = Number(params.get('notification'));

  return {
    view: params.get('view') === 'notifications' ? 'notifications' : 'overview',
    query: (params.get('q') ?? '').trim().slice(0, 200),
    status: validStatuses.includes(rawStatus as NotificationStatus)
      ? rawStatus as NotificationStatus
      : 'all',
    page: Number.isSafeInteger(rawPage) && rawPage > 0 ? rawPage : 1,
    notificationId: Number.isSafeInteger(rawNotificationId) && rawNotificationId > 0
      ? rawNotificationId
      : null,
  };
}

export function writeConsoleUrlState(
  patch: Partial<ConsoleUrlState>,
  replace = false,
): ConsoleUrlState {
  const current = readConsoleUrlState();
  const next: ConsoleUrlState = { ...current, ...patch };
  next.query = next.query.trim().slice(0, 200);
  if (!Number.isSafeInteger(next.page) || next.page < 1) next.page = 1;
  if (next.notificationId === null || !Number.isSafeInteger(next.notificationId) || next.notificationId <= 0) {
    next.notificationId = null;
  }
  const url = new URL(window.location.href);

  if (next.view === 'notifications') url.searchParams.set('view', 'notifications');
  else url.searchParams.delete('view');

  if (next.query.trim()) url.searchParams.set('q', next.query.trim().slice(0, 200));
  else url.searchParams.delete('q');

  if (next.status !== 'all') url.searchParams.set('status', next.status);
  else url.searchParams.delete('status');

  if (next.page > 1) url.searchParams.set('page', String(next.page));
  else url.searchParams.delete('page');

  if (next.notificationId !== null && next.notificationId > 0) {
    url.searchParams.set('notification', String(next.notificationId));
  } else {
    url.searchParams.delete('notification');
  }

  const destination = `${url.pathname}${url.search}${url.hash}`;
  const currentDestination = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (destination !== currentDestination) {
    if (replace) window.history.replaceState(null, '', destination);
    else window.history.pushState(null, '', destination);
  }

  return next;
}
