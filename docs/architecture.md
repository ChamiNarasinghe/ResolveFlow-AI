# Architecture

## System overview

ResolveFlow AI is a three-tier application composed of a Go REST API, a Python AI
classifier, and a PostgreSQL database. All three services run as Docker containers
on a shared bridge network.

```
                        ┌─────────────────────────────────────────┐
                        │           resolveflow-net (bridge)       │
                        │                                          │
  Client ──HTTP──► :8080│  ┌──────────────┐    ┌───────────────┐  │
                        │  │  Go backend  │───►│  AI service   │  │
                        │  │  :8080       │    │  :8001        │  │
                        │  └──────┬───────┘    └───────────────┘  │
                        │         │                                │
                        │  ┌──────▼───────┐                       │
                        │  │  PostgreSQL  │                       │
                        │  │  :5432       │                       │
                        │  └──────────────┘                       │
                        └─────────────────────────────────────────┘
```

## Services

### Go backend (`backend/`)

- Language: Go 1.26
- HTTP: standard library `net/http` with Go 1.22 pattern routing
- Database: `pgx/v5` connection pool (max 10 connections)
- Auth: HS256 JWT tokens via `golang-jwt/jwt`
- Passwords: bcrypt via `golang.org/x/crypto`
- Configuration: environment variables via `godotenv` (dev) or injected (container)

Responsibilities:
- User registration, login, and JWT issuance
- Ticket CRUD with soft-delete and audit history
- Team and membership management
- AI classification on ticket creation (non-blocking — ticket is created even if AI fails)
- Storing and serving AI predictions
- Rate limiting per IP (login: 5/min, register: 3/min, API: 100/min)

### Python AI service (`ai-service/`)

- Language: Python 3.13
- Framework: FastAPI 0.165 with uvicorn[standard]
- Auth: `X-API-Key` header validated against `AI_SERVICE_API_KEY`

Responsibilities:
- Keyword-based ticket classification (category, priority, sentiment, confidence)
- Stateless — no database dependency
- Exposes `POST /api/v1/tickets/classify` and `GET /health`

### PostgreSQL (`postgres:17`)

- Persistent data via named Docker volume `resolveflow_postgres_data`
- Schema managed by golang-migrate (sequential numbered SQL files)
- Current schema version: 3

## Request flow — ticket creation

```
Client
  │  POST /api/v1/tickets
  ▼
Auth middleware (JWT validation)
  ▼
TicketHandler.CreateTicket
  ├── INSERT INTO tickets → returns ticket row
  ├── POST http://ai-service:8001/api/v1/tickets/classify  (5s timeout)
  │     ├── success → INSERT INTO ai_predictions
  │     └── failure → log warning, continue (ticket already created)
  └── 201 Created with ticket JSON
```

The AI classification is fire-and-forget from the client's perspective. A classification
failure never blocks ticket creation.

## Authentication and authorisation

- All API endpoints except `/health*` and `/api/v1/auth/login` require a valid JWT.
- Tokens are signed with HS256 using `JWT_SECRET` and expire after `JWT_EXPIRATION_MINUTES`.
- Role-based access: `admin` > `manager` > `agent`.
- Admin-only endpoints: ticket deletion, team assignment, team management, user provisioning.
- Registration (`POST /api/v1/auth/register`) is enabled only when `APP_ENV=development`.

## Inter-service security

The Go backend authenticates to the AI service using a shared secret sent as the
`X-API-Key` request header. The AI service reads `AI_SERVICE_API_KEY` from its
environment. If the variable is unset, authentication is skipped (development mode).

Both services receive the same `AI_SERVICE_API_KEY` value via Docker Compose
environment variable injection.

## Startup and dependency ordering

```
postgres (healthcheck: pg_isready)
  └── backend (depends_on: postgres healthy)

ai-service (no dependencies — starts in parallel with postgres)
```

The backend will not start until PostgreSQL passes its `pg_isready` health check.
The backend does not depend on the AI service at startup — it connects to the AI
service only when classifying a ticket, and handles failures gracefully.

## Database migrations

Migrations are plain SQL files in `backend/migrations/` managed by the
golang-migrate CLI. They are intentionally kept separate from application startup:

- The application binary does not run migrations automatically.
- Migrations are run manually via `make migrate-up` or as a one-off Docker command.
- This prevents accidental schema changes during rolling restarts or crash loops.

The migration binary and SQL files are included in the backend Docker image so
migrations can be run against the production database using the same image:

```bash
docker compose run --rm backend \
  migrate -path migrations -database "$DATABASE_URL" up
```

## Container design

### Go backend image

- Builder: `golang:1.26-alpine` — compiles a fully static binary (`CGO_ENABLED=0`)
- Runtime: `alpine:3.21` — minimal OS with `ca-certificates` and `tzdata`
- Final image size: ~31 MB
- Runs as non-root user `appuser`

### Python AI service image

- Base: `python:3.13-slim`
- Dependencies installed with `--no-cache-dir`
- Final image size: ~247 MB
- Runs as non-root user `appuser`

## Security considerations

- No secrets are hardcoded in source code or Docker images.
- All secrets are injected at runtime via environment variables.
- `.env` is excluded from version control via `.gitignore`.
- Both application containers run as non-root users.
- The PostgreSQL port (5433) is exposed on the host for development tooling only;
  inter-service communication uses the internal Docker network on port 5432.
- `POSTGRES_PASSWORD` and `JWT_SECRET` use the `:?` Compose syntax — Compose
  refuses to start if either is unset.
