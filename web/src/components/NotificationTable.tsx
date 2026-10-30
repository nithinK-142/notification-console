import { IconBell, IconEye, IconTrash } from '@tabler/icons-react';
import { Box, Button, Group, Progress, Table, Text, ThemeIcon } from '@mantine/core';
import type { NotificationSummary } from '../types';
import { formatDate, NotificationStatusBadge } from './NotificationStatus';

interface NotificationTableProps {
  items: NotificationSummary[];
  onSelect: (id: number) => void;
  onDelete?: (notification: NotificationSummary) => void;
  emptyTitle?: string;
  emptyDescription?: string;
}

export function NotificationTable({
  items,
  onSelect,
  onDelete,
  emptyTitle = 'No notifications found',
  emptyDescription = 'Change the filters or create a new notification.',
}: NotificationTableProps) {
  if (items.length === 0) {
    return (
      <Box className="flex min-h-[220px] flex-col items-center justify-center px-6 py-8 text-center">
        <ThemeIcon size={46} radius="xl" variant="light" color="gray"><IconBell size={22} /></ThemeIcon>
        <Text fw={650} mt="md">{emptyTitle}</Text>
        <Text c="dimmed" size="sm" mt={4} maw={340}>{emptyDescription}</Text>
      </Box>
    );
  }

  return (
    <>
      <Box className="grid gap-3 p-3 sm:p-4 xl:hidden">
        {items.map((item) => {
          const progress = item.total > 0 ? Math.min(100, (item.sent / item.total) * 100) : 0;
          return (
            <Box key={item.id} className="mobile-notification-card w-full rounded-xl border border-slate-200 bg-white p-4 transition-colors hover:border-[var(--accent-border)] hover:bg-slate-50">
                <Group justify="space-between" align="flex-start" gap="sm" wrap="nowrap">
                  <Group gap="sm" wrap="nowrap" className="min-w-0">
                    <ThemeIcon size={38} radius="md" color="copper" variant="light"><IconBell size={19} /></ThemeIcon>
                    <Box className="min-w-0">
                      <Text fw={650} size="sm" lineClamp={2}>{item.title}</Text>
                      <Text size="xs" c="dimmed" mt={3}>Notification #{item.id}</Text>
                    </Box>
                  </Group>
                  <NotificationStatusBadge status={item.status} />
                </Group>

                <Box mt="md">
                  <Group justify="space-between" gap="xs" mb={6}>
                    <Text size="xs" c="dimmed">Delivery progress</Text>
                    <Text size="xs" fw={650}>{item.sent.toLocaleString()} / {item.total.toLocaleString()}</Text>
                  </Group>
                  <Progress value={progress} size="sm" radius="xl" color={item.failed > 0 ? 'brick' : 'copper'} />
                </Box>

                <Group justify="space-between" mt="md" gap="xs" wrap="wrap">
                  <Text size="xs" c="dimmed">{formatDate(item.created_at)}</Text>
                  <Group gap="xs">
                    <Button size="xs" variant="light" color="copper" leftSection={<IconEye size={14} />} aria-label={`View notification ${item.id}`} onClick={() => onSelect(item.id)}>View</Button>
                    {onDelete && <Button size="xs" variant="subtle" color="brick" leftSection={<IconTrash size={14} />} disabled={item.status === 'processing'} title={item.status === 'processing' ? 'Cannot delete while the worker is processing this notification.' : undefined} aria-label={`Delete notification ${item.id}`} onClick={() => onDelete(item)}>Delete</Button>}
                  </Group>
                </Group>
            </Box>
          );
        })}
      </Box>

      <Box className="hidden xl:block">
        <Table.ScrollContainer minWidth={1040} className="notification-table-scroll">
          <Table
            verticalSpacing="md"
            horizontalSpacing="lg"
            withTableBorder
            withColumnBorders
            className="notification-data-table"
            style={{ tableLayout: 'fixed', width: '100%' }}
          >
            <Table.Thead>
              <Table.Tr>
                <Table.Th style={{ width: '34%', textAlign: 'left' }}>Notification</Table.Th>
                <Table.Th style={{ width: '13%', textAlign: 'left' }}>Status</Table.Th>
                <Table.Th style={{ width: '18%', textAlign: 'left' }}>Delivery</Table.Th>
                <Table.Th style={{ width: '17%', textAlign: 'left' }}>Created</Table.Th>
                <Table.Th style={{ width: '18%', textAlign: 'center' }}>Actions</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {items.map((item) => {
                const progress = item.total > 0 ? Math.min(100, (item.sent / item.total) * 100) : 0;
                return (
                  <Table.Tr key={item.id} className="notification-row">
                    <Table.Td style={{ textAlign: 'left' }}>
                      <Group gap="sm" wrap="nowrap">
                        <ThemeIcon size={36} radius="md" color="copper" variant="light"><IconBell size={18} /></ThemeIcon>
                        <Box className="min-w-0 flex-1">
                          <Text fw={600} size="sm" lineClamp={1}>{item.title}</Text>
                          <Text size="xs" c="dimmed">ID #{item.id}</Text>
                        </Box>
                      </Group>
                    </Table.Td>
                    <Table.Td style={{ textAlign: 'left' }}><NotificationStatusBadge status={item.status} /></Table.Td>
                    <Table.Td style={{ textAlign: 'left' }}>
                      <Box w={140} maw="100%">
                        <Group justify="space-between" mb={5} gap="xs">
                          <Text size="xs" fw={600}>{item.sent.toLocaleString()} sent</Text>
                          <Text size="xs" c="dimmed">{item.total.toLocaleString()}</Text>
                        </Group>
                        <Progress value={progress} size="xs" radius="xl" color={item.failed > 0 ? 'brick' : 'copper'} />
                      </Box>
                    </Table.Td>
                    <Table.Td style={{ textAlign: 'left' }}><Text size="sm" c="dimmed" style={{ whiteSpace: 'nowrap' }}>{formatDate(item.created_at)}</Text></Table.Td>
                    <Table.Td style={{ textAlign: 'right' }}>
                      <Group justify="flex-end" gap="xs" wrap="nowrap">
                        <Button size="xs" variant="light" color="copper" leftSection={<IconEye size={14} />} aria-label={`View notification ${item.id}`} onClick={() => onSelect(item.id)}>View</Button>
                        {onDelete && <Button size="xs" variant="subtle" color="brick" leftSection={<IconTrash size={14} />} disabled={item.status === 'processing'} title={item.status === 'processing' ? 'Cannot delete while the worker is processing this notification.' : undefined} aria-label={`Delete notification ${item.id}`} onClick={() => onDelete(item)}>Delete</Button>}
                      </Group>
                    </Table.Td>
                  </Table.Tr>
                );
              })}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      </Box>
    </>
  );
}
