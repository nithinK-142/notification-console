import { useState } from 'react';
import { IconAlertCircle, IconCalendar, IconClock, IconRefresh, IconTrash } from '@tabler/icons-react';
import { Alert, Box, Button, Divider, Drawer, Group, LoadingOverlay, Modal, Paper, Progress, SimpleGrid, Skeleton, Stack, Table, Text, ThemeIcon, Title } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useDeleteNotification, useNotification, useNotificationBatches } from '../queries';
import { formatDate, NotificationStatusBadge } from './NotificationStatus';

interface NotificationDetailsDrawerProps {
  id: number | null;
  onClose: () => void;
  onDeleted: () => void;
}

export function NotificationDetailsDrawer({ id, onClose, onDeleted }: NotificationDetailsDrawerProps) {
  const details = useNotification(id);
  const batches = useNotificationBatches(id);
  const deleteNotification = useDeleteNotification();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const job = details.data?.job;
  const progress = job && job.total > 0 ? Math.min(100, (job.sent / job.total) * 100) : 0;

  async function handleDelete() {
    if (id === null) return;
    try {
      await deleteNotification.mutateAsync(id);
      notifications.show({ color: 'moss', title: 'Notification deleted', message: `Notification #${id} was removed.` });
      setConfirmDelete(false);
      onDeleted();
    } catch (error) {
      notifications.show({
        color: 'brick',
        title: 'Could not delete notification',
        message: error instanceof Error ? error.message : 'Request failed.',
      });
    }
  }

  return (
    <>
      <Drawer opened={id !== null} onClose={onClose} position="right" size={660} title="Notification details" overlayProps={{ backgroundOpacity: 0.25, blur: 2 }}>
        {details.isPending ? (
          <Stack gap="md"><Skeleton height={28} /><Skeleton height={100} /><Skeleton height={170} /></Stack>
        ) : details.isError ? (
          <Alert color="brick" icon={<IconAlertCircle size={18} />} title="Could not load notification">
            {details.error instanceof Error ? details.error.message : 'The API request failed.'}
          </Alert>
        ) : details.data ? (
          <Stack gap="lg" pos="relative">
            <LoadingOverlay visible={deleteNotification.isPending} />
            <Group justify="space-between" align="flex-start" wrap="nowrap">
              <Box>
                <Text size="xs" c="dimmed" fw={600} tt="uppercase">Notification #{details.data.id}</Text>
                <Title order={2} mt={6}>{details.data.title}</Title>
              </Box>
              <NotificationStatusBadge status={details.data.job.status} />
            </Group>

            <Paper withBorder radius="md" p="md" className="message-preview border-slate-200 bg-slate-50">
              <Text size="xs" c="dimmed" fw={600} tt="uppercase" mb="sm">Message</Text>
              <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>{details.data.body}</Text>
              <Divider my="md" />
              <Group gap="xs"><IconCalendar size={15} /><Text size="xs" c="dimmed">Created {formatDate(details.data.created_at)}</Text></Group>
            </Paper>

            <Box>
              <Group justify="space-between" align="center" mb="sm">
                <Title order={4}>Delivery progress</Title>
                <Text size="sm" fw={600}>{job?.sent.toLocaleString()} / {job?.total.toLocaleString()} sent</Text>
              </Group>
              <Progress value={progress} size="md" radius="xl" color={job?.failed ? 'brick' : 'copper'} />
              <SimpleGrid cols={{ base: 1, xs: 3 }} mt="md" spacing="sm">
                <Paper withBorder radius="md" p="sm"><Text size="xs" c="dimmed">Failed</Text><Title order={3} mt={4}>{job?.failed.toLocaleString()}</Title></Paper>
                <Paper withBorder radius="md" p="sm"><Text size="xs" c="dimmed">Invalid tokens</Text><Title order={3} mt={4}>{job?.invalid.toLocaleString()}</Title></Paper>
                <Paper withBorder radius="md" p="sm"><Text size="xs" c="dimmed">Attempts</Text><Title order={3} mt={4}>{job?.attempts.toLocaleString()}</Title></Paper>
              </SimpleGrid>
            </Box>

            <Box>
              <Title order={4} mb="sm">Worker state</Title>
              <SimpleGrid cols={{ base: 1, xs: 2 }} spacing="sm">
                <Paper withBorder radius="md" p="sm">
                  <Text size="xs" c="dimmed">Cursor / high watermark</Text>
                  <Text fw={600} mt={5}>{job?.cursor.toLocaleString()} / {job?.high_watermark.toLocaleString()}</Text>
                </Paper>
                <Paper withBorder radius="md" p="sm">
                  <Text size="xs" c="dimmed">Next attempt</Text>
                  <Text fw={600} size="sm" mt={5}>{job?.next_attempt_at ? formatDate(job.next_attempt_at) : '—'}</Text>
                </Paper>
                <Paper withBorder radius="md" p="sm">
                  <Text size="xs" c="dimmed">Started</Text>
                  <Text fw={600} size="sm" mt={5}>{job?.started_at ? formatDate(job.started_at) : 'Not started'}</Text>
                </Paper>
                <Paper withBorder radius="md" p="sm">
                  <Text size="xs" c="dimmed">Completed</Text>
                  <Text fw={600} size="sm" mt={5}>{job?.completed_at ? formatDate(job.completed_at) : 'Not completed'}</Text>
                </Paper>
                {job?.lease_until && <Paper withBorder radius="md" p="sm">
                  <Text size="xs" c="dimmed">Worker lease until</Text>
                  <Text fw={600} size="sm" mt={5}>{formatDate(job.lease_until)}</Text>
                </Paper>}
              </SimpleGrid>
            </Box>

            {job?.last_error && (
              <Alert color="brick" icon={<IconAlertCircle size={17} />} title="Latest worker error">{job.last_error}</Alert>
            )}

            <Box>
              <Group justify="space-between" mb="sm">
                <Title order={4}>Delivery batches</Title>
                <Text size="xs" c="dimmed">Most recent first</Text>
              </Group>
              {batches.isPending ? <Skeleton height={100} /> : batches.isError ? (
                <Alert color="brick" title="Could not load delivery batches">{batches.error instanceof Error ? batches.error.message : 'The request failed.'}</Alert>
              ) : batches.data.pages.every((page) => page.items.length === 0) ? (
                <Paper withBorder p="lg" radius="md"><Group gap="sm"><ThemeIcon color="gray" variant="light"><IconClock size={16} /></ThemeIcon><Text size="sm" c="dimmed">No batches yet. Worker has not started sending.</Text></Group></Paper>
              ) : (
                <Table.ScrollContainer minWidth={440} className="notification-batches-scroll">
                  <Table verticalSpacing="sm" withTableBorder withColumnBorders className="notification-batches-table">
                    <Table.Thead><Table.Tr><Table.Th>Batch</Table.Th><Table.Th>Results</Table.Th><Table.Th>Status</Table.Th><Table.Th>Started</Table.Th></Table.Tr></Table.Thead>
                    <Table.Tbody>
                      {batches.data.pages.flatMap((page) => page.items).map((batch) => (
                        <Table.Tr key={`${batch.notification_id}-${batch.batch_no}`}>
                          <Table.Td><Text size="sm" fw={600}>#{batch.batch_no}</Text></Table.Td>
                          <Table.Td>
                            <Text size="sm">{batch.success} sent</Text>
                            <Text size="xs" c="dimmed">{batch.failure} failed · {batch.invalid} invalid</Text>
                            {batch.error && <Text size="xs" c="brick" lineClamp={2}>{batch.error}</Text>}
                          </Table.Td>
                          <Table.Td><NotificationStatusBadge status={batch.status} /></Table.Td>
                          <Table.Td><Text size="xs" c="dimmed" style={{ whiteSpace: 'nowrap' }}>{formatDate(batch.started_at)}</Text></Table.Td>
                        </Table.Tr>
                      ))}
                    </Table.Tbody>
                  </Table>
                </Table.ScrollContainer>
              )}
              {batches.hasNextPage && <Button variant="subtle" size="sm" mt="sm" leftSection={<IconRefresh size={15} />} loading={batches.isFetchingNextPage} onClick={() => batches.fetchNextPage()}>Load older batches</Button>}
            </Box>

            <Divider />
            <Group justify="space-between" align="center">
              <Text size="xs" c="dimmed">ID #{details.data.id}</Text>
              <Button color="brick" variant="light" leftSection={<IconTrash size={16} />} onClick={() => setConfirmDelete(true)} disabled={job?.status === 'processing'}>Delete notification</Button>
            </Group>
            {job?.status === 'processing' && <Text size="xs" c="dimmed">Cannot delete while the worker is processing this notification.</Text>}
          </Stack>
        ) : null}
      </Drawer>

      <Modal opened={confirmDelete} onClose={() => setConfirmDelete(false)} title="Delete notification?" centered size="sm">
        <Stack gap="md">
          <Text size="sm" c="dimmed">This removes the notification and its delivery history. This action cannot be undone.</Text>
          <Group justify="flex-end">
            <Button variant="default" onClick={() => setConfirmDelete(false)}>Cancel</Button>
            <Button color="brick" leftSection={<IconTrash size={16} />} loading={deleteNotification.isPending} onClick={handleDelete}>Delete</Button>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
