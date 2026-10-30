import { useCallback, useEffect, useState } from 'react';
import { Center, Loader } from '@mantine/core';
import { useAuth } from './auth';
import { ConsoleLayout, type ConsolePage } from './components/ConsoleLayout';
import { CreateNotificationModal } from './components/CreateNotificationModal';
import { NotificationDetailsDrawer } from './components/NotificationDetailsDrawer';
import { DashboardPage } from './pages/DashboardPage';
import { LoginPage } from './pages/LoginPage';
import { NotificationsPage } from './pages/NotificationsPage';
import { readConsoleUrlState, writeConsoleUrlState, type ConsoleUrlState } from './url-state';

export default function App() {
  const { user, loading } = useAuth();
  const [urlState, setUrlState] = useState<ConsoleUrlState>(() => readConsoleUrlState());
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    const syncFromHistory = () => setUrlState(readConsoleUrlState());
    window.addEventListener('popstate', syncFromHistory);
    return () => window.removeEventListener('popstate', syncFromHistory);
  }, []);

  const updateUrl = useCallback((patch: Partial<ConsoleUrlState>, replace = false) => {
    setUrlState(writeConsoleUrlState(patch, replace));
  }, []);

  const navigate = useCallback((page: ConsolePage) => {
    updateUrl({ view: page, notificationId: null });
  }, [updateUrl]);

  const closeNotification = useCallback(() => {
    updateUrl({ notificationId: null }, true);
  }, [updateUrl]);

  const changeQuery = useCallback((query: string) => {
    updateUrl({ query, page: 1 });
  }, [updateUrl]);

  const changeStatus = useCallback((status: ConsoleUrlState['status']) => {
    updateUrl({ status, page: 1 });
  }, [updateUrl]);

  const changePage = useCallback((page: number) => {
    updateUrl({ page });
  }, [updateUrl]);

  const selectNotification = useCallback((id: number) => {
    updateUrl({ notificationId: id });
  }, [updateUrl]);

  if (loading) {
    return <Center h="100vh"><Loader size="md" /></Center>;
  }

  if (!user) return <LoginPage />;

  return (
    <>
      <ConsoleLayout page={urlState.view} onNavigate={navigate}>
        {urlState.view === 'overview' ? (
          <DashboardPage
            onNewNotification={() => setCreateOpen(true)}
            onSelectNotification={selectNotification}
            onViewAll={() => navigate('notifications')}
          />
        ) : (
          <NotificationsPage
            query={urlState.query}
            status={urlState.status}
            page={urlState.page}
            onQueryChange={changeQuery}
            onStatusChange={changeStatus}
            onPageChange={changePage}
            onNewNotification={() => setCreateOpen(true)}
            onSelectNotification={selectNotification}
          />
        )}
      </ConsoleLayout>
      <CreateNotificationModal opened={createOpen} onClose={() => setCreateOpen(false)} />
      <NotificationDetailsDrawer
        key={urlState.notificationId ?? 'closed'}
        id={urlState.notificationId}
        onClose={closeNotification}
        onDeleted={closeNotification}
      />
    </>
  );
}
