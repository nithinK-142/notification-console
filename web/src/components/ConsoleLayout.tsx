import { useRef, useState, type KeyboardEvent, type PointerEvent, type ReactNode } from 'react';
import { useLocalStorage } from '@mantine/hooks';
import { IconChevronDown, IconLayoutDashboard, IconLayoutSidebarLeftCollapse, IconLayoutSidebarLeftExpand, IconLogout, IconSend } from '@tabler/icons-react';
import { ActionIcon, AppShell, Avatar, Box, Burger, Divider, Group, Menu, Text, Tooltip, UnstyledButton } from '@mantine/core';
import { useAuth } from '../auth';
import { useApiHealth } from '../queries';
import { BrandMark } from './BrandMark';
import { ColorSchemeToggle } from './ColorSchemeToggle';

export type ConsolePage = 'overview' | 'notifications';

interface ConsoleLayoutProps {
  page: ConsolePage;
  onNavigate: (page: ConsolePage) => void;
  children: ReactNode;
}

const minSidebarWidth = 228;
const maxSidebarWidth = 400;
const collapsedSidebarWidth = 84;

export function ConsoleLayout({ page, onNavigate, children }: ConsoleLayoutProps) {
  const { user, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [sidebarWidth, setSidebarWidth] = useState(272);
  const [sidebarCollapsed, setSidebarCollapsed] = useLocalStorage<boolean>({
    key: 'notification-console-sidebar-collapsed',
    defaultValue: false,
  });
  const [resizing, setResizing] = useState(false);
  const resizeStart = useRef<{ x: number; width: number } | null>(null);

  async function signOut() {
    try {
      await logout();
    } catch {
      // The API request failed. Local auth state is still cleared by the provider.
    }
  }

  function navigate(nextPage: ConsolePage) {
    onNavigate(nextPage);
    setMobileOpen(false);
  }

  function startResize(event: PointerEvent<HTMLDivElement>) {
    if (event.button !== 0) return;
    resizeStart.current = { x: event.clientX, width: sidebarWidth };
    event.currentTarget.setPointerCapture(event.pointerId);
    setResizing(true);
    event.preventDefault();
  }

  function moveResize(event: PointerEvent<HTMLDivElement>) {
    if (!resizeStart.current) return;
    const nextWidth = resizeStart.current.width + event.clientX - resizeStart.current.x;
    setSidebarWidth(Math.min(maxSidebarWidth, Math.max(minSidebarWidth, nextWidth)));
  }

  function stopResize() {
    resizeStart.current = null;
    setResizing(false);
  }

  function resizeWithKeyboard(event: KeyboardEvent<HTMLDivElement>) {
    const delta = event.key === 'ArrowRight' ? 12 : event.key === 'ArrowLeft' ? -12 : 0;
    if (!delta) return;
    event.preventDefault();
    setSidebarWidth((width) => Math.min(maxSidebarWidth, Math.max(minSidebarWidth, width + delta)));
  }

  return (
    <AppShell
      navbar={{ width: sidebarCollapsed ? collapsedSidebarWidth : sidebarWidth, breakpoint: 'sm', collapsed: { mobile: !mobileOpen } }}
      padding={{ base: 'md', md: 'xl' }}
      className={`min-h-screen bg-slate-50 ${resizing ? 'is-resizing' : ''}`}
      transitionDuration={resizing ? 0 : 180}
    >
      <AppShell.Navbar p={sidebarCollapsed ? 'xs' : 'lg'} className="console-sidebar border-r border-slate-200 bg-white">
        {sidebarCollapsed ? (
          <Box className="mb-9 flex flex-col items-center gap-3">
            <BrandMark size={36} />
            <Tooltip label="Expand sidebar" position="right" offset={10}>
              <ActionIcon
                variant="subtle"
                color="gray"
                size={40}
                radius="md"
                className="sidebar-toggle sidebar-toggle-collapsed"
                aria-label="Expand sidebar"
                aria-expanded={false}
                onClick={() => setSidebarCollapsed(false)}
              >
                <IconLayoutSidebarLeftExpand size={19} stroke={1.8} />
              </ActionIcon>
            </Tooltip>
          </Box>
        ) : (
          <Group gap="sm" wrap="nowrap" justify="space-between" className="mb-9 min-w-0">
            <Group gap="sm" wrap="nowrap" className="min-w-0">
              <BrandMark size={40} />
              <Box className="min-w-0">
                <Text fw={750} size="sm" c="dark.9" className="brand-title">Notification Console</Text>
                <Text size="xs" c="dimmed" className="brand-subtitle tracking-[0.08em]">MESSAGING SYSTEM</Text>
              </Box>
            </Group>
            <Tooltip label="Collapse sidebar" position="right">
              <ActionIcon
                variant="subtle"
                color="gray"
                size={34}
                radius="md"
                className="sidebar-toggle shrink-0"
                aria-label="Collapse sidebar"
                aria-expanded={true}
                onClick={() => setSidebarCollapsed(true)}
              >
                <IconLayoutSidebarLeftCollapse size={19} stroke={1.8} />
              </ActionIcon>
            </Tooltip>
          </Group>
        )}

        {!sidebarCollapsed && <Text size="xs" c="dimmed" fw={700} tt="uppercase" mb="sm" className="tracking-widest">Workspace</Text>}
        <Box className="grid gap-1.5">
          <Tooltip label="Overview" position="right" disabled={!sidebarCollapsed}>
            <UnstyledButton aria-label="Overview" aria-current={page === 'overview' ? 'page' : undefined} className={`nav-link flex w-full items-center rounded-lg py-2.5 text-left transition-colors ${sidebarCollapsed ? 'justify-center px-0' : 'gap-3 px-3'} ${page === 'overview' ? 'nav-link-active' : ''}`} onClick={() => navigate('overview')}>
              <IconLayoutDashboard size={18} stroke={1.8} />
              {!sidebarCollapsed && <Text size="sm" fw={550}>Overview</Text>}
            </UnstyledButton>
          </Tooltip>
          <Tooltip label="Notifications" position="right" disabled={!sidebarCollapsed}>
            <UnstyledButton aria-label="Notifications" aria-current={page === 'notifications' ? 'page' : undefined} className={`nav-link flex w-full items-center rounded-lg py-2.5 text-left transition-colors ${sidebarCollapsed ? 'justify-center px-0' : 'gap-3 px-3'} ${page === 'notifications' ? 'nav-link-active' : ''}`} onClick={() => navigate('notifications')}>
              <IconSend size={18} stroke={1.8} />
              {!sidebarCollapsed && <Text size="sm" fw={550}>Notifications</Text>}
            </UnstyledButton>
          </Tooltip>
        </Box>

        <Box className="mt-auto min-w-0">
          <Divider my="md" color="gray.2" />
          <Group gap={sidebarCollapsed ? 0 : 'sm'} wrap="nowrap" justify={sidebarCollapsed ? 'center' : 'flex-start'} className="min-w-0">
            {!sidebarCollapsed && <Avatar color="copper" radius="md">{user?.email.slice(0, 1).toUpperCase() ?? 'A'}</Avatar>}
            {!sidebarCollapsed && (
              <Box className="min-w-0 flex-1">
                <Text size="sm" fw={600} lineClamp={1}>{user?.email}</Text>
                <Text size="xs" c="dimmed">Administrator</Text>
              </Box>
            )}
            <Menu position="top-end" shadow="md" width={180}>
              <Menu.Target>
                {sidebarCollapsed ? (
                  <UnstyledButton aria-label="Account options" title="Account options" className="rounded-lg outline-none account-menu-button">
                    <Avatar color="copper" radius="md">{user?.email.slice(0, 1).toUpperCase() ?? 'A'}</Avatar>
                  </UnstyledButton>
                ) : (
                  <ActionIcon variant="subtle" color="gray" aria-label="Account options"><IconChevronDown size={16} /></ActionIcon>
                )}
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item leftSection={<IconLogout size={15} />} onClick={() => void signOut()}>Sign out</Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </Group>
        </Box>

        {!sidebarCollapsed && (
          <div
            className="sidebar-resize-handle"
            role="separator"
            tabIndex={0}
            aria-label="Resize sidebar"
            aria-orientation="vertical"
            aria-valuemin={minSidebarWidth}
            aria-valuemax={maxSidebarWidth}
            aria-valuenow={sidebarWidth}
            onPointerDown={startResize}
            onPointerMove={moveResize}
            onPointerUp={stopResize}
            onPointerCancel={stopResize}
            onLostPointerCapture={stopResize}
            onKeyDown={resizeWithKeyboard}
          />
        )}
      </AppShell.Navbar>

      <AppShell.Main className="console-main">
        <Box className="console-theme-toggle">
          <ColorSchemeToggle compact />
        </Box>
        <Box className="console-main-content mx-auto w-full max-w-[1480px]">
          <Box hiddenFrom="sm" className="mb-4 flex justify-start">
            <Burger opened={mobileOpen} onClick={() => setMobileOpen((open) => !open)} size="sm" aria-label="Toggle navigation" />
          </Box>
          {children}
        </Box>
        <Box component="footer" className="console-footer mx-auto w-full max-w-[1480px]">
          <Text fz="sm" className="console-footer-copy">© {new Date().getFullYear()} Notification Console · All rights reserved.</Text>
          <SystemHealthIndicator />
        </Box>
      </AppShell.Main>
    </AppShell>
  );
}

function SystemHealthIndicator() {
  const health = useApiHealth();
  const connected = health.data?.status === 'ok' && !health.isError;
  const checking = health.isPending;
  const label = connected ? 'Operational' : checking ? 'Checking' : 'Unavailable';
  const state = connected ? 'online' : checking ? 'checking' : 'offline';

  return (
    <Tooltip label={`${label} · API and Database`} position="top" withArrow>
      <Box className={`system-health-indicator system-health-${state}`}
        role="status"
        aria-live="polite"
        aria-label={`${label}. API and Database.`}
      >
        <span className="system-health-indicator-dot" aria-hidden="true" />
        <Box className="system-health-indicator-copy">
          <Text className="system-health-indicator-label">{label}</Text>
        </Box>
      </Box>
    </Tooltip>
  );
}
