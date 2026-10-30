import { Badge } from '@mantine/core';
import type { NotificationStatus } from '../types';

const statusStyles: Record<NotificationStatus | 'sending', { color: string; label: string }> = {
  pending: { color: 'ochre', label: 'Pending' },
  processing: { color: 'stone', label: 'Processing' },
  done: { color: 'moss', label: 'Completed' },
  failed: { color: 'brick', label: 'Failed' },
  sending: { color: 'copper', label: 'Sending' },
};

export function NotificationStatusBadge({ status }: { status: NotificationStatus | 'sending' }) {
  const style = statusStyles[status];
  return <Badge color={style.color} variant="light" radius="sm">{style.label}</Badge>;
}

export function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}
