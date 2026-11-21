# ResolveFlow AI

An AI-powered customer support ticket management system.

## Architecture

| Service | Technology | Port |
|---|---|---|
| Go REST API | Go 1.26, `net/http`, pgx/v5 | 8080 |
| AI classifier | Python 3.13, FastAPI, uvicorn | 8001 |
| Database | PostgreSQL 17 | 5433 (host) / 5432 (internal) |

The Go backend handles authentication, ticket lifecycle, teams, and comments.
The Python AI service classifies incoming tickets by category, priority, and sentiment.
PostgreSQL stores all application data with schema managed by golang-migrate.

## Prerequisites

- Docker Desktop 24+ with Compose v2
- Go 1.26+ (local development only)
- Python 3.13+ (local development only)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate) (migrations only)

## Quick start with Docker

```bash
# 1. Copy and configure environment variables
cp .env.example .env
# Edit .env — set POSTGRES_PASSWORD, JWT_SECRET, and optionally AI_SERVICE_API_KEY

# 2. Build and start all services
docker compose up -d

# 3. Run database migrations (first run only)
docker compose run --rm backend \
  migrate -path migrations -database "$DATABASE_URL" up

# 4. Verify all services are healthy
docker compose ps
```

The API is available at `http://localhost:8080`.
The AI service is available at `http://localhost:8001`.

## Environment variables

Copy `.env.example` to `.env` and fill in the required values.

| Variable | Required | Description |
|---|---|---|
| `POSTGRES_PASSWORD` | Yes | PostgreSQL superuser password |
| `JWT_SECRET` | Yes | HS256 signing key, minimum 32 characters |
| `AI_SERVICE_API_KEY` | No | Shared secret for backend→AI service auth |
| `BOOTSTRAP_ADMIN_*` | No | One-time admin account creation on startup |

`DATABASE_URL` and `AI_SERVICE_URL` are constructed automatically by Docker Compose
from the other variables. Set them explicitly only when running services outside Docker.

Generate secure secrets:

```bash
openssl rand -base64 32
```

## Local development (without Docker)

### Go backend

```bash
cd backend

# Run tests
go test ./...

# Run static analysis
go vet ./...

# Start the server (requires a running PostgreSQL instance)
go run ./cmd/api
```

### Python AI service

```bash
cd ai-service

pip install -r requirements.txt
uvicorn app.main:app --reload --port 8001
```

### Database migrations

```bash
cd backend

# Apply all pending migrations
make migrate-up

# Roll back one migration
make migrate-down

# Check current version
make migrate-version

# Create a new migration pair
make migrate-create NAME=add_something
```

## API endpoints

### Health

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness check |
| GET | `/health/ready` | Readiness — verifies database connectivity |
| GET | `/health/ai` | AI service reachability check |

### Authentication

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | None | Register a new agent (development only) |
| POST | `/api/v1/auth/login` | None | Obtain a JWT access token |
| GET | `/api/v1/auth/me` | JWT | Return the authenticated user's profile |

### Tickets

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/tickets` | JWT | Create a ticket (triggers AI classification) |
| GET | `/api/v1/tickets` | JWT | List tickets with pagination and filters |
| GET | `/api/v1/tickets/{id}` | JWT | Get a single ticket |
| PATCH | `/api/v1/tickets/{id}` | JWT | Update status, priority, category, assignment |
| DELETE | `/api/v1/tickets/{id}` | Admin | Soft-delete a ticket |
| PATCH | `/api/v1/tickets/{id}/assignment` | Admin | Assign or unassign a team |
| GET | `/api/v1/tickets/{id}/history` | JWT | Paginated audit history |
| GET | `/api/v1/tickets/{id}/ai-prediction` | JWT | Latest AI classification result |
| POST | `/api/v1/tickets/{id}/comments` | JWT | Add a comment |
| GET | `/api/v1/tickets/{id}/comments` | JWT | List comments |

### Teams

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/teams` | Admin | Create a team |
| GET | `/api/v1/teams` | JWT | List all teams |
| GET | `/api/v1/teams/{id}` | JWT | Get a team |
| PATCH | `/api/v1/teams/{id}` | Admin | Update a team |
| POST | `/api/v1/teams/{id}/members` | Admin | Add a member |
| GET | `/api/v1/teams/{id}/members` | JWT | List members |
| DELETE | `/api/v1/teams/{id}/members/{userId}` | Admin | Remove a member |

### Admin

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/admin/users` | Admin | Provision a user with an explicit role |

## Docker operations

```bash
# View logs
docker compose logs -f backend
docker compose logs -f ai-service

# Rebuild after code changes
docker compose build backend
docker compose up -d backend

# Stop all services (data is preserved)
docker compose down

# Stop and remove volumes (DESTRUCTIVE — deletes all data)
docker compose down -v
```

## Project structure

```
ResolveFlow-AI/
├── backend/                  Go REST API
│   ├── cmd/api/              Application entry point
│   ├── internal/
│   │   ├── auth/             JWT and password handling
│   │   ├── config/           Environment variable loading
│   │   ├── database/         PostgreSQL connection pool
│   │   ├── handlers/         HTTP request handlers
│   │   ├── middleware/        Auth, logging, rate limiting, recovery
│   │   ├── models/           Domain types
│   │   ├── repository/       Database access layer
│   │   ├── routes/           Route registration
│   │   ├── services/         AI client
│   │   ├── tickets/          Ticket validation
│   │   └── users/            User repository
│   ├── migrations/           SQL migration files
│   ├── Dockerfile
│   └── Makefile
├── ai-service/               Python FastAPI classifier
│   ├── app/
│   │   ├── api/              Route handlers
│   │   ├── dependencies/     API key auth
│   │   ├── schemas/          Pydantic request/response models
│   │   └── services/         Keyword-based classifier
│   ├── tests/
│   ├── Dockerfile
│   └── requirements.txt
├── docs/                     Architecture and design documentation
├── docker-compose.yml
└── .env.example
```
