import { IconMoonStars, IconSun } from '@tabler/icons-react';
import { ActionIcon, Group, Text, Tooltip, UnstyledButton, useComputedColorScheme, useMantineColorScheme } from '@mantine/core';

export function ColorSchemeToggle({ compact = false }: { compact?: boolean }) {
  const computedColorScheme = useComputedColorScheme('light');
  const { setColorScheme } = useMantineColorScheme();
  const isDark = computedColorScheme === 'dark';
  const label = isDark ? 'Light mode' : 'Dark mode';
  const description = `Switch to ${label.toLowerCase()}`;
  const Icon = isDark ? IconSun : IconMoonStars;

  function toggleColorScheme() {
    setColorScheme(isDark ? 'light' : 'dark');
  }

  if (compact) {
    return (
      <Tooltip label={description} position="bottom">
        <ActionIcon
          variant="subtle"
          color="gray"
          size="lg"
          className="theme-toggle-compact"
          aria-label={description}
          onClick={toggleColorScheme}
        >
          <Icon size={18} />
        </ActionIcon>
      </Tooltip>
    );
  }

  return (
    <UnstyledButton className="theme-toggle" onClick={toggleColorScheme} aria-label={description}>
      <Group gap="sm" wrap="nowrap">
        <Icon size={17} />
        <Text size="sm" fw={550}>{label}</Text>
      </Group>
    </UnstyledButton>
  );
}
