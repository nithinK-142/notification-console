import { useState, type FormEvent } from 'react';
import { Button, Group, Modal, Stack, Text, TextInput, Textarea } from '@mantine/core';
import { IconSend } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { useCreateNotification } from '../queries';

export function CreateNotificationModal({ opened, onClose }: { opened: boolean; onClose: () => void }) {
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const createNotification = useCreateNotification();

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      const result = await createNotification.mutateAsync({ title: title.trim(), body: body.trim() });
      notifications.show({
        color: 'moss',
        title: 'Notification queued',
        message: `Notification #${result.id} is ready for delivery.`,
      });
      setTitle('');
      setBody('');
      onClose();
    } catch (error) {
      notifications.show({
        color: 'brick',
        title: 'Could not create notification',
        message: error instanceof Error ? error.message : 'Request failed.',
      });
    }
  }

  return (
    <Modal opened={opened} onClose={onClose} title="Create notification" centered size="lg" radius="lg">
      <form onSubmit={handleSubmit}>
        <Stack gap="lg">
          <Text size="sm" c="dimmed">Send a push notification to all active app devices. This action queues delivery immediately.</Text>
          <TextInput
            label="Notification title"
            placeholder="e.g. Your order has shipped"
            value={title}
            onChange={(event) => setTitle(event.currentTarget.value)}
            maxLength={120}
            required
            autoFocus
            description={`${title.length}/120 characters`}
          />
          <Textarea
            label="Message"
            placeholder="Write the message users will receive…"
            value={body}
            onChange={(event) => setBody(event.currentTarget.value)}
            minRows={5}
            maxRows={9}
            maxLength={2000}
            required
            autosize
            description={`${body.length}/2000 characters`}
          />
          <Group justify="space-between" mt="xs">
            <Button variant="default" onClick={onClose}>Cancel</Button>
            <Button type="submit" leftSection={<IconSend size={16} />} loading={createNotification.isPending} disabled={!title.trim() || !body.trim()}>
              Queue notification
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  );
}
