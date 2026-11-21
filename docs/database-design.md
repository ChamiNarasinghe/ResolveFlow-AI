# Database Design

## Engine

PostgreSQL 17, managed via Docker with a named persistent volume.

Connection: `pgx/v5` pool, max 10 connections, min 0.

## Schema overview

```
users
  └── team_members ──► teams
  └── tickets (created_by, assigned_to)
        └── ticket_comments
        └── ticket_history
        └── ai_predictions
              └── teams (suggested_team_id)
```

## Tables

### `users`

Stores agent, manager, and admin accounts.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `gen_random_uuid()` |
| `full_name` | VARCHAR(150) | NOT NULL |
| `email` | VARCHAR(255) | NOT NULL, UNIQUE |
| `password_hash` | TEXT | bcrypt hash, never returned by API |
| `role` | VARCHAR(30) | `admin`, `manager`, `agent` |
| `is_active` | BOOLEAN | Default `true` |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |

Indexes: `idx_users_email` (unique constraint also creates one).

### `teams`

Support teams that tickets can be assigned to.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `name` | VARCHAR(100) | UNIQUE |
| `description` | TEXT | Nullable |
| `created_at` | TIMESTAMPTZ | |

### `team_members`

Many-to-many join between teams and users.

| Column | Type | Notes |
|---|---|---|
| `team_id` | UUID FK → teams | CASCADE delete |
| `user_id` | UUID FK → users | CASCADE delete |
| `team_role` | VARCHAR(30) | `lead`, `member` |
| `joined_at` | TIMESTAMPTZ | |

Primary key: `(team_id, user_id)`.

### `tickets`

Core entity. Supports soft-delete via `deleted_at`.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `ticket_number` | BIGINT | Generated identity, UNIQUE |
| `subject` | VARCHAR(255) | NOT NULL |
| `description` | TEXT | NOT NULL |
| `customer_name` | VARCHAR(150) | NOT NULL |
| `customer_email` | VARCHAR(255) | NOT NULL |
| `status` | VARCHAR(30) | `open`, `in_progress`, `waiting_for_customer`, `resolved`, `closed` |
| `priority` | VARCHAR(20) | `low`, `medium`, `high`, `urgent` |
| `channel` | VARCHAR(20) | `web`, `email`, `chat`, `phone` |
| `category` | VARCHAR(100) | Nullable |
| `assigned_team_id` | UUID FK → teams | Nullable, SET NULL on delete |
| `assigned_to` | UUID FK → users | Nullable, SET NULL on delete |
| `created_by` | UUID FK → users | Nullable, SET NULL on delete |
| `created_at` | TIMESTAMPTZ | |
| `updated_at` | TIMESTAMPTZ | |
| `resolved_at` | TIMESTAMPTZ | Set when status → `resolved`, cleared on reopen |
| `deleted_at` | TIMESTAMPTZ | NULL = active, non-NULL = soft-deleted |

Indexes: `status`, `priority`, `assigned_to`, `assigned_team_id`, `created_at DESC`,
partial index on `deleted_at WHERE deleted_at IS NULL`.

All queries filter `WHERE deleted_at IS NULL` to exclude soft-deleted tickets.

### `ticket_comments`

Comments and internal notes on tickets.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `ticket_id` | UUID FK → tickets | CASCADE delete |
| `author_id` | UUID FK → users | Nullable, SET NULL on delete |
| `body` | TEXT | NOT NULL |
| `is_internal` | BOOLEAN | Default `false` |
| `author_type` | VARCHAR(20) | `customer`, `agent`, `system` — nullable |
| `created_at` | TIMESTAMPTZ | |

### `ticket_history`

Immutable audit log of all ticket field changes.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `ticket_id` | UUID FK → tickets | CASCADE delete |
| `actor_id` | UUID FK → users | Nullable, SET NULL on delete |
| `action` | VARCHAR(100) | e.g. `ticket.updated`, `ticket.deleted`, `ticket.assigned` |
| `old_value` | JSONB | Field values before the change |
| `new_value` | JSONB | Field values after the change |
| `created_at` | TIMESTAMPTZ | |

### `ai_predictions`

AI classification results linked to tickets.

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | |
| `ticket_id` | UUID FK → tickets | CASCADE delete |
| `model_version` | VARCHAR(100) | NOT NULL |
| `category` | VARCHAR(100) | Nullable |
| `predicted_priority` | VARCHAR(20) | Nullable |
| `sentiment` | VARCHAR(20) | Nullable |
| `confidence` | NUMERIC(5,4) | 0.0000–1.0000, nullable |
| `suggested_team_id` | UUID FK → teams | Nullable, SET NULL on delete |
| `response_suggestion` | TEXT | Nullable |
| `created_at` | TIMESTAMPTZ | |

Multiple predictions can exist per ticket. The API returns the most recent one.

## Migrations

Migrations are managed by the [golang-migrate](https://github.com/golang-migrate/migrate) CLI.
Files live in `backend/migrations/` and follow the naming convention:

```
NNN_description.up.sql
NNN_description.down.sql
```

The `schema_migrations` table tracks the current version and dirty state.

### Current migrations

| Version | Description |
|---|---|
| 001 | Initial schema — all tables, indexes, constraints |
| 002 | Add `author_type` column to `ticket_comments` |
| 003 | Add `deleted_at` column and partial index to `tickets` |

### Running migrations

```bash
# From the backend directory (requires DATABASE_URL in environment)
make migrate-up        # apply all pending
make migrate-down      # roll back one
make migrate-version   # print current version

# Via Docker (uses the same image as production)
docker compose run --rm backend \
  migrate -path migrations -database "$DATABASE_URL" up
```

### Creating a new migration

```bash
make migrate-create NAME=add_something
# Creates: migrations/004_add_something.up.sql
#          migrations/004_add_something.down.sql
```

Every `up` migration must have a corresponding `down` migration that fully reverses it.
Use `ADD COLUMN IF NOT EXISTS`, `DROP COLUMN IF EXISTS`, and `DROP INDEX IF EXISTS`
in down migrations to make them idempotent.
