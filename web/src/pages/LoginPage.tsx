import { useState, type FormEvent } from 'react';
import { IconLock } from '@tabler/icons-react';
import { Alert, Box, Button, Paper, PasswordInput, Stack, Text, TextInput, ThemeIcon, Title } from '@mantine/core';
import { useAuth } from '../auth';
import { BrandMark } from '../components/BrandMark';

export function LoginPage() {
  const { login } = useAuth();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      await login(email.trim(), password);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : 'Sign in failed.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="login-page grid min-h-screen bg-white md:grid-cols-2">
      <section className="login-hero relative flex min-h-[280px] flex-col overflow-hidden px-7 py-8 text-white sm:px-10 md:min-h-screen md:px-[clamp(32px,5vw,72px)] md:py-14">
        <div className="relative z-10 flex items-center gap-3">
          <BrandMark size={46} />
          <Box>
            <Text fw={750} size="lg" c="white">Notification Console</Text>
            <Text size="xs" c="rgba(255,255,255,.58)" className="tracking-[0.1em]">MESSAGING SYSTEM</Text>
          </Box>
        </div>
        <div className="relative z-10 my-12 md:my-auto md:py-14">
          <Text className="eyebrow" c="copper.2">PUSH NOTIFICATIONS, UNDER CONTROL</Text>
          <Title order={1} c="white" className="text-[2.25rem] leading-[1.08] tracking-[-0.055em] sm:text-[clamp(2.6rem,4.3vw,4.3rem)]">Every message.<br />Accounted for.</Title>
          <Text c="rgba(255,255,255,.68)" size="lg" maw={430} mt="lg">Create notifications, track delivery, and inspect worker activity from one focused workspace.</Text>
        </div>
        <div className="login-decoration decoration-one" />
        <div className="login-decoration decoration-two" />
      </section>

      <section className="login-form-panel relative flex flex-col items-center justify-center bg-slate-50 px-4 py-7 sm:px-7 sm:py-10">
        <Paper radius="xl" p={{ base: 'xl', sm: 38 }} className="login-card w-full max-w-[440px] border-slate-200 shadow-[0_20px_60px_rgba(27,35,70,0.07)]" withBorder>
          <Title order={2} mb="sm" className="text-center">Sign in</Title>

          {error && <Alert color="brick" mb="lg" title="Sign in failed">{error}</Alert>}

          <form onSubmit={handleSubmit}>
            <Stack gap="md">
              <TextInput
                label="Admin email"
                placeholder="admin@example.com"
                type="email"
                autoComplete="username"
                value={email}
                onChange={(event) => setEmail(event.currentTarget.value)}
                required
                autoFocus
              />
              <PasswordInput
                label="Password"
                placeholder="Enter your password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.currentTarget.value)}
                required
              />
              <Button
                type="submit"
                fullWidth
                size="md"
                mt="sm"
                color="copper"
                variant="filled"
                className="login-submit"
                loading={submitting}
              >
                Sign in to console
              </Button>
            </Stack>
          </form>
          <Text size="xs" c="dimmed" mt="xl" ta="center">Session expires after 8 hours.</Text>
        </Paper>
      </section>
    </main>
  );
}
