import { useEffect, useState } from 'react';
import { IconAlertCircle, IconPlus, IconSearch, IconSortAscending, IconSortDescending, IconTrash, IconX } from '@tabler/icons-react';
import { ActionIcon, Alert, Box, Button, Group, Modal, Pagination, Paper, Skeleton, Stack, Text, TextInput, Title, UnstyledButton } from '@mantine/core';
import { notifications as toastNotifications } from '@mantine/notifications';
import { NotificationTable } from '../components/NotificationTable';
import { useDeleteNotification, useNotificationStats, useNotifications } from '../queries';
import type { NotificationSort, NotificationStatus, NotificationSummary } from '../types';
import type { ConsoleStatus } from '../url-state';

interface NotificationsPageProps {
  query: string;
  status: ConsoleStatus;
  page: number;
  sort: NotificationSort;
  onQueryChange: (query: string) => void;
  onStatusChange: (status: ConsoleStatus) => void;
  onPageChange: (page: number) => void;
  onSortChange: (sort: NotificationSort) => void;
  onNewNotification: () => void;
  onSelectNotification: (id: number) => void;
}

const filters = [
  { label: 'All', value: 'all' },
  { label: 'Pending', value: 'pending' },
  { label: 'Processing', value: 'processing' },
  { label: 'Completed', value: 'done' },
  { label: 'Failed', value: 'failed' },
];

export function NotificationsPage({
  query, status: filter, page, sort, onQueryChange, onStatusChange, onPageChange, onSortChange,
  onNewNotification, onSelectNotification,
}: NotificationsPageProps) {
  const [search, setSearch] = useState(query);
  const [deleteTarget, setDeleteTarget] = useState<NotificationSummary | null>(null);
  const deleteNotification = useDeleteNotification();
  const limit = 10;
  const status = filter === 'all' ? undefined : filter as NotificationStatus;

  useEffect(() => setSearch(query), [query]);
  const notifications = useNotifications({
    limit,
    offset: (page - 1) * limit,
    q: query || undefined,
    status,
    sort,
  });
  const stats = useNotificationStats();
  const totalPages = Math.max(1, Math.ceil((notifications.data?.total ?? 0) / limit));
  useEffect(() => {
    if (notifications.data && page > totalPages) onPageChange(totalPages);
  }, [notifications.data, page, totalPages, onPageChange]);

  const counts: Record<string, number | undefined> = {
    all: stats.data?.total ?? notifications.data?.total,
    pending: stats.data?.pending,
    processing: stats.data?.processing,
    done: stats.data?.done,
    failed: stats.data?.failed,
  };

  function applySearch() {
    onQueryChange(search.trim());
  }

  function clearSearch() {
    setSearch('');
    onQueryChange('');
  }

  async function confirmDelete() {
    if (!deleteTarget || deleteTarget.status === 'processing') return;

    try {
      await deleteNotification.mutateAsync(deleteTarget.id);
      toastNotifications.show({ color: 'moss', title: 'Notification deleted', message: `Notification #${deleteTarget.id} was removed.` });
      setDeleteTarget(null);
    } catch (error) {
      toastNotifications.show({
        color: 'brick',
        title: 'Could not delete notification',
        message: error instanceof Error ? error.message : 'Request failed.',
      });
    }
  }

  return (
    <Stack gap="xl" className="min-w-0">
      <Group justify="space-between" align="flex-end" className="page-heading flex-col items-start gap-4 md:flex-row md:items-end">
        <Box>
          <Text size="sm" c="dimmed" mb={6}>Workspace / Notifications</Text>
          <Title order={1}>Notifications</Title>
          <Text c="dimmed" mt={6}>Create, search, and inspect push notification jobs.</Text>
        </Box>
        <Button leftSection={<IconPlus size={17} />} onClick={onNewNotification}>New notification</Button>
      </Group>

      <Paper radius="xl" withBorder className="notifications-toolbar" p={{ base: 'sm', sm: 'md' }}>
        <Stack gap="sm">
          <Group align="center" gap="sm" wrap="nowrap" className="notification-search-row">
            <TextInput
              aria-label="Search title or message"
              placeholder="Search title or message"
              leftSection={<IconSearch size={15} />}
              value={search}
              maxLength={200}
              onChange={(event) => setSearch(event.currentTarget.value)}
              onKeyDown={(event) => { if (event.key === 'Enter') applySearch(); }}
              rightSection={search ? <ActionIcon variant="subtle" color="gray" size="sm" aria-label="Clear search" onClick={clearSearch}><IconX size={14} /></ActionIcon> : null}
              className="min-w-0 flex-1 notification-search-input"
              size="sm"
            />
            <Button variant="default" leftSection={<IconSearch size={14} />} size="sm" onClick={applySearch}>Search</Button>
            <Button
              variant="default"
              leftSection={sort === 'desc' ? <IconSortDescending size={15} /> : <IconSortAscending size={15} />}
              size="sm"
              aria-pressed={sort === 'asc'}
              aria-label={`Sort order ${sort.toUpperCase()}; switch to ${sort === 'desc' ? 'ascending' : 'descending'}`}
              title={sort === 'desc' ? 'Newest first · click for oldest first' : 'Oldest first · click for newest first'}
              onClick={() => onSortChange(sort === 'desc' ? 'asc' : 'desc')}
            >
              {sort.toUpperCase()}
            </Button>
          </Group>

          <Box className="notification-filter-track" role="group" aria-label="Filter notifications by status">
            {filters.map((item) => {
              const active = filter === item.value;
              const count = counts[item.value];
              return (
                <UnstyledButton
                  key={item.value}
                  aria-pressed={active}
                  className={`notification-filter ${active ? 'notification-filter-active' : ''}`}
                  onClick={() => onStatusChange(item.value as ConsoleStatus)}
                >
                  <Text size="xs" fw={active ? 650 : 500}>{item.label}</Text>
                  <span className={`notification-filter-count ${active ? 'notification-filter-count-active' : ''}`}>
                    {count === undefined ? '—' : count.toLocaleString()}
                  </span>
                </UnstyledButton>
              );
            })}
          </Box>
        </Stack>
      </Paper>

      <Paper radius="xl" withBorder className="panel-card" p={0} aria-busy={notifications.isFetching}>
        {notifications.isPending ? (
          <Stack p="lg" gap="md">{Array.from({ length: 6 }, (_, index) => <Skeleton key={index} height={54} radius="md" />)}</Stack>
        ) : notifications.isError ? (
          <Alert color="brick" icon={<IconAlertCircle size={18} />} m="lg" title="Could not load notifications">
            {notifications.error instanceof Error ? notifications.error.message : 'The API request failed.'}
          </Alert>
        ) : (
          <NotificationTable
            items={notifications.data.items}
            onSelect={onSelectNotification}
            onDelete={setDeleteTarget}
          />
        )}

        <Group justify="space-between" p="lg" className="pagination-footer border-t border-slate-100" wrap="wrap" gap="sm">
          <Text size="sm" c="dimmed">Page {page} of {totalPages}</Text>
          <Pagination value={page} onChange={onPageChange} total={totalPages} size="sm" withEdges disabled={notifications.isPending || notifications.isError || totalPages < 2} />
        </Group>
      </Paper>

      <Modal
        opened={deleteTarget !== null}
        onClose={() => setDeleteTarget(null)}
        title={`Delete notification${deleteTarget ? ` #${deleteTarget.id}` : ''}?`}
        centered
        size="sm"
      >
        <Stack gap="md">
          <Text size="sm" fw={600}>{deleteTarget?.title}</Text>
          <Text size="sm" c="dimmed">This removes the notification and its delivery history. This action cannot be undone.</Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button color="brick" leftSection={<IconTrash size={15} />} loading={deleteNotification.isPending} onClick={() => void confirmDelete()}>Delete</Button>
          </Group>
        </Stack>
      </Modal>
    </Stack>
  );
}
