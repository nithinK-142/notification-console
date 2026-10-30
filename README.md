# Notification Console

Go-based notification platform with a React dashboard, PostgreSQL-backed job processing, and Firebase Cloud Messaging for delivery.

## Delivery flow

1. The API stores the notification and a pending job in PostgreSQL.
2. A worker claims the job using a database lease.
3. The worker sends to active device tokens in batches of 500.
4. Transient failures are retried, invalid tokens are deactivated, and delivery results are saved.

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

This installs project dependencies, development tools and creates missing environment files from the examples. Existing files are not overwritten.

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

`make admin-create` prompts for the admin password. The dashboard runs at `http://127.0.0.1:5173`; the API listens on port `8080`.

## Commands

| Command | Purpose |
|---|---|
| `make dev` | Run API, worker, and dashboard |
| `make api` | Run API with live reload |
| `make worker` | Run worker with live reload |
| `make worker-once` | Process currently claimable jobs, then exit |
| `make test` | Run Go tests |
| `make check` | Run formatting checks, vet, tests, and dashboard build |
| `make build` | Build API, worker, and dashboard |
| `make health` | Check API and PostgreSQL health |
| `make help` | List all available commands |

## API collection

Bruno requests are in [`bruno/`](bruno/).
