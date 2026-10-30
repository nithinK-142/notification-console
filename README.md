# Notification Console

Go-based notification platform with a React dashboard, PostgreSQL-backed job processing, and Firebase Cloud Messaging for delivery.

## How delivery works

1. The API validates the request and creates the notification and its `pending` job in one PostgreSQL transaction.
2. Workers poll every 2 seconds and claim due jobs with `FOR UPDATE SKIP LOCKED`. This prevents two workers from claiming the same job.
3. The worker records the last available device ID, then reads active device tokens in ID order, up to 500 at a time.
4. Each batch is recorded as `sending` before the FCM multicast call. FCM returns one result per token. The worker saves batch results, updates job counters and cursor, and deactivates invalid tokens.
5. After processing all devices that existed when the job started, the worker marks the job as `done`.

## Concurrency and recovery

- One worker processes a job's batches sequentially. Multiple worker instances can process different jobs concurrently.
- Each active job has a heartbeat goroutine that renews its 2-minute lease every 30 seconds. The goroutine stops when processing ends.
- If a worker stops, the lease expires. Another worker can reclaim the job and resume from the last committed cursor.
- FCM and PostgreSQL cannot share a transaction. A crash after FCM accepts a batch but before its results are saved can cause that batch to be sent again.

## States and retries

- **Job:** `pending` → `processing` → `done` or `failed`. Retryable call failures return the job to `pending`.
- **Batch:** `sending` → `done` or `failed`. `done` means FCM returned per-token results; some tokens may still have failed.
- Transient token failures get up to 3 retries with 1, 2, and 4-second delays.
- Whole-call failures use 15, 30, 60, and 120-second backoff. The job fails after the fifth whole-call failure.
- Invalid device tokens are deactivated.

## Local setup

### Requirements

- Go 1.27.3
- Node.js 20.19+ or 22.12+
- PostgreSQL
- Atlas CLI
- Firebase service-account credentials

### Configure

```sh
make setup
```

This creates missing environment files from the examples and installs project dependencies and development tools. Existing files are not overwritten.

Set these values in the root `.env`:

- `DATABASE_URL` — PostgreSQL connection URL
- `JWT_SECRET` — at least 32 bytes
- `APP_API_KEY` — at least 32 bytes
- `ALLOWED_ORIGINS` — dashboard origin, normally `http://localhost:5173`
- `COOKIE_SECURE` — use `false` for local HTTP development
- `API_ADDR` — defaults to `:8080`

Set `VITE_API_URL` in `web/.env` (normally `http://localhost:8080`).

Save the Firebase Admin SDK service-account JSON as `notification-service-key.json` in the project root. The worker requires this file.

### Start

```sh
make db-up
make admin-create EMAIL=example@mail.com
make dev
```

The admin command prompts for a password. The dashboard runs at `http://127.0.0.1:5173`; the API listens on port `8080`.

## Commands

| Command | Purpose |
|---|---|
| `make setup` | Create env files and install dependencies and tools |
| `make dev` | Run API, worker, and dashboard together |
| `make api` | Run API with live reload |
| `make worker` | Run worker with live reload |
| `make worker-once` | Process currently claimable jobs, then exit |
| `make web` | Run the dashboard |
| `make build` | Build API, worker, and dashboard |
| `make test` | Run Go tests |
| `make test-race` | Run Go tests with the race detector |
| `make check` | Run formatting checks, vet, tests, and dashboard build |
| `make health` | Check API and PostgreSQL health |
| `make clean` | Remove generated build files and dependencies |
| `make help` | List available commands |

## API collection

Bruno requests are in [`bruno/`](bruno/).
