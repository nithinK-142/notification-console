import { IconAlertCircle, IconArrowUpRight, IconBellRinging, IconChevronRight, IconCircleCheck, IconClock, IconExclamationCircle, IconPlus } from '@tabler/icons-react';
import { Alert, Box, Button, Group, Paper, SimpleGrid, Skeleton, Stack, Text, ThemeIcon, Title, UnstyledButton } from '@mantine/core';
import { NotificationStatusBadge, formatDate } from '../components/NotificationStatus';
import { useNotificationStats, useRecentNotifications } from '../queries';
import type { NotificationSummary } from '../types';

interface DashboardPageProps {
  onNewNotification: () => void;
  onSelectNotification: (id: number) => void;
  onViewAll: () => void;
}

export function DashboardPage({ onNewNotification, onSelectNotification, onViewAll }: DashboardPageProps) {
  const stats = useNotificationStats();
  const recent = useRecentNotifications();
  const values = stats.data;

  const cards = [
    { label: 'Total notifications', value: values?.total, icon: IconBellRinging, color: 'copper', note: 'All time' },
    { label: 'Completed', value: values?.done, icon: IconCircleCheck, color: 'moss', note: 'Finished jobs' },
    { label: 'In queue', value: (values?.pending ?? 0) + (values?.processing ?? 0), icon: IconClock, color: 'ochre', note: `${values?.processing ?? 0} currently processing` },
    { label: 'Failed', value: values?.failed, icon: IconExclamationCircle, color: 'brick', note: 'Need attention' },
  ];

  return (
    <Stack gap="xl" className="min-w-0">
      <Group justify="space-between" align="flex-end" className="page-heading flex-col items-start gap-4 md:flex-row md:items-end">
        <Box>
          <Text size="sm" c="dimmed" mb={6}>Workspace / Overview</Text>
          <Title order={1}>Overview</Title>
          <Text c="dimmed" mt={6}>Monitor notification jobs and delivery progress.</Text>
        </Box>
        <Button leftSection={<IconPlus size={17} />} onClick={onNewNotification}>New notification</Button>
      </Group>

      {stats.isError && (
        <Alert color="brick" icon={<IconAlertCircle size={18} />} title="Could not load overview">
          {stats.error instanceof Error ? stats.error.message : 'The API request failed.'}
        </Alert>
      )}

      <SimpleGrid cols={{ base: 1, xs: 2, xl: 4 }} spacing="md">
        {cards.map((card) => (
          <Paper key={card.label} p="lg" radius="lg" withBorder className="stat-card transition-shadow hover:shadow-md">
            <Group justify="space-between" align="flex-start">
              <Text size="sm" c="dimmed" fw={500}>{card.label}</Text>
              <ThemeIcon color={card.color} variant="light" size={38} radius="md"><card.icon size={19} /></ThemeIcon>
            </Group>
            {stats.isPending ? <Skeleton height={32} width={72} mt="lg" /> : <Title order={2} mt="lg" className="stat-value">{(card.value ?? 0).toLocaleString()}</Title>}
            <Text size="xs" c="dimmed" mt="xs">{card.note}</Text>
          </Paper>
        ))}
      </SimpleGrid>

      <Paper radius="lg" withBorder className="panel-card" p={0}>
        <Group justify="space-between" p="lg" className="panel-header">
          <Box>
            <Title order={3}>Recent notifications</Title>
            <Text size="sm" c="dimmed" mt={4}>Latest jobs across your workspace.</Text>
          </Box>
          <Button variant="subtle" rightSection={<IconArrowUpRight size={16} />} onClick={onViewAll}>View all</Button>
        </Group>
        {recent.isPending ? (
          <Box className="recent-notification-list">{Array.from({ length: 4 }, (_, index) => <Skeleton key={index} height={78} radius="md" />)}</Box>
        ) : recent.isError ? (
          <Alert color="brick" m="lg" title="Could not load notifications">{recent.error instanceof Error ? recent.error.message : 'The API request failed.'}</Alert>
        ) : recent.data.items.length === 0 ? (
          <Box px="lg" py="xl"><Text size="sm" c="dimmed">No notifications yet. Create a notification to start sending messages to your apps.</Text></Box>
        ) : (
          <Box className="recent-notification-list">
            {recent.data.items.map((item) => (
              <RecentNotificationRow key={item.id} item={item} onSelect={onSelectNotification} />
            ))}
          </Box>
        )}
      </Paper>
    </Stack>
  );
}

function RecentNotificationRow({ item, onSelect }: { item: NotificationSummary; onSelect: (id: number) => void }) {
  return (
    <UnstyledButton
      className="recent-notification-row"
      onClick={() => onSelect(item.id)}
      aria-label={`View notification ${item.id}: ${item.title}`}
    >
      <Group justify="space-between" align="center" gap="sm" wrap="nowrap" className="recent-notification-heading">
        <Text size="sm" fw={620} lineClamp={1} className="min-w-0">{item.title}</Text>
        <NotificationStatusBadge status={item.status} />
      </Group>
      <Group justify="space-between" align="center" gap="sm" wrap="nowrap" className="recent-notification-meta">
        <Group gap="xs" wrap="nowrap" className="min-w-0">
          <Text size="xs" c="dimmed">#{item.id}</Text>
          <Text size="xs" c="dimmed" lineClamp={1}>{formatDate(item.created_at)}</Text>
        </Group>
        <Group gap="xs" wrap="nowrap" className="shrink-0">
          <Text size="xs" c="dimmed">{item.sent.toLocaleString()} / {item.total.toLocaleString()} sent</Text>
          <IconChevronRight size={15} className="recent-notification-chevron" />
        </Group>
      </Group>
    </UnstyledButton>
  );
}
