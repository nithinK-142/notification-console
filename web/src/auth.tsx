import { createContext, useContext, useEffect, type ReactNode } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from './api';
import type { AdminSession } from './types';

interface AuthContextValue {
  user: AdminSession | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);
const authKey = ['auth', 'me'] as const;

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const session = useQuery({ queryKey: authKey, queryFn: api.me, retry: false });
  const loginMutation = useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) => api.login(email, password),
    onSuccess: (user) => queryClient.setQueryData(authKey, user),
  });
  const logoutMutation = useMutation({
    mutationFn: api.logout,
    onSettled: () => {
      queryClient.removeQueries({ predicate: (query) => query.queryKey[0] !== 'auth' });
      queryClient.setQueryData(authKey, null);
    },
  });

  useEffect(() => {
    const handleExpired = () => queryClient.setQueryData(authKey, null);
    window.addEventListener('notification-console:auth-expired', handleExpired);
    return () => window.removeEventListener('notification-console:auth-expired', handleExpired);
  }, [queryClient]);

  return (
    <AuthContext.Provider
      value={{
        user: session.data ?? null,
        loading: session.isPending,
        login: async (email, password) => { await loginMutation.mutateAsync({ email, password }); },
        logout: async () => { await logoutMutation.mutateAsync(); },
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider');
  return context;
}
