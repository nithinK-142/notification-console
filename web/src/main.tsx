import React from 'react';
import ReactDOM from 'react-dom/client';
import { Input, MantineProvider, createTheme } from '@mantine/core';
import { Notifications } from '@mantine/notifications';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider } from './auth';
import App from './App';
import './styles.css';

const theme = createTheme({
  primaryColor: 'copper',
  primaryShade: { light: 6, dark: 5 },
  colors: {
    copper: [
      '#faf1ef', '#f4e4e1', '#ebcfca', '#dbafa9', '#cc8b83',
      '#ba6b62', '#ad5049', '#963f39', '#78322e', '#592522',
    ],
    moss: [
      '#f1f5f0', '#e4ebe2', '#d0ddcf', '#b7cbb5', '#95b095',
      '#78957a', '#617d63', '#4c654e', '#394d3c', '#28382b',
    ],
    ochre: [
      '#f8f4e9', '#f1e9d4', '#e5d7b6', '#d4bf8e', '#bda36b',
      '#a18650', '#836a3f', '#695333', '#504027', '#3b301e',
    ],
    stone: [
      '#f4f5f3', '#e8ebe8', '#d6ddd8', '#bbc6bf', '#9eaea5',
      '#81958b', '#687b71', '#526259', '#3e4b44', '#2c3831',
    ],
    brick: [
      '#f8f0ed', '#f1ded9', '#e6c6bd', '#d7a69a', '#c58679',
      '#ad6c60', '#91574d', '#74443e', '#593430', '#402521',
    ],
  },
  fontFamily: '-apple-system, BlinkMacSystemFont, \"Segoe UI\", system-ui, sans-serif',
  headings: { fontFamily: '-apple-system, BlinkMacSystemFont, \"Segoe UI\", system-ui, sans-serif', fontWeight: '620' },
  defaultRadius: 'sm',
  components: {
    Input: Input.extend({
      classNames: { input: 'app-text-field' },
    }),
  },
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 15_000, retry: false, refetchOnWindowFocus: false },
  },
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <MantineProvider theme={theme} defaultColorScheme="light">
      <Notifications position="top-right" />
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </QueryClientProvider>
    </MantineProvider>
  </React.StrictMode>,
);
