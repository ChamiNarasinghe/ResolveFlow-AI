# ResolveFlow AI – Requirements

## Overview

ResolveFlow AI is a customer support ticket management system with an AI-powered classification layer. It provides a REST API for managing tickets, teams, users, and comments, with automatic ticket classification on creation.

---

## Functional Requirements

### Authentication & Users

- FR-AUTH-1: Agents can register via `POST /api/v1/auth/register` (development mode only).
- FR-AUTH-2: Users authenticate via `POST /api/v1/auth/login` and receive a JWT Bearer token.
- FR-AUTH-3: Authenticated users can retrieve their own profile via `GET /api/v1/auth/me`.
- FR-AUTH-4: Admins can provision users with explicit roles via `POST /api/v1/admin/users`.
- FR-AUTH-5: Passwords must be 8–72 characters and contain at least one letter and one number.
- FR-AUTH-6: Inactive accounts are rejected at login and profile retrieval.
- FR-AUTH-7: A one-time bootstrap admin account can be created via `BOOTSTRAP_ADMIN_*` environment variables on startup.

### Tickets

- FR-TICKET-1: Authenticated users can create tickets with subject, description, customer name, customer email, and channel.
- FR-TICKET-2: Tickets are created with status `open` and priority `medium` by default.
- FR-TICKET-3: Ticket creation triggers AI classification; the result is persisted asynchronously (ticket creation does not block on AI).
- FR-TICKET-4: Authenticated users can list tickets with pagination (page, page_size) and filters (status, priority).
- FR-TICKET-5: Authenticated users can retrieve a single ticket by UUID.
- FR-TICKET-6: Authenticated users can update ticket status, priority, category, assigned_team_id, and assigned_to.
- FR-TICKET-7: Every ticket update records an audit entry in `ticket_history` for each changed field.
- FR-TICKET-8: Setting status to `resolved` stamps `resolved_at`; any other status clears it.
- FR-TICKET-9: Admins can soft-delete tickets; soft-deleted tickets are excluded from all queries.
- FR-TICKET-10: Admins can assign or unassign a ticket to/from a team; assignment changes are audited.
- FR-TICKET-11: Authenticated users can retrieve paginated audit history for a ticket.
- FR-TICKET-12: Authenticated users can retrieve the latest AI prediction for a ticket.

### Comments

- FR-COMMENT-1: Authenticated users can add comments to tickets.
- FR-COMMENT-2: Comments have a body (max 5000 characters) and an `is_internal` flag.
- FR-COMMENT-3: Comments are attributed to the authenticated user with `author_type=agent`.
- FR-COMMENT-4: Authenticated users can list all comments for a ticket.

### Teams

- FR-TEAM-1: Admins can create teams with a unique name and optional description.
- FR-TEAM-2: Authenticated users can list all teams.
- FR-TEAM-3: Authenticated users can retrieve a team by UUID.
- FR-TEAM-4: Admins can update a team's name and/or description.
- FR-TEAM-5: Admins can add users to teams with a role of `lead` or `member`.
- FR-TEAM-6: Authenticated users can list team members.
- FR-TEAM-7: Admins can remove users from teams.

### AI Classification

- FR-AI-1: The AI service classifies tickets by category, priority, and sentiment.
- FR-AI-2: Categories: `billing`, `technical_support`, `account`, `general_inquiry`.
- FR-AI-3: Priorities: `low`, `medium`, `high`, `urgent`.
- FR-AI-4: Sentiments: `positive`, `neutral`, `negative`.
- FR-AI-5: A confidence score in [0.0, 1.0] is returned with each classification.
- FR-AI-6: Classification failures are logged and do not prevent ticket creation.
- FR-AI-7: The AI service health is exposed via `GET /health/ai` on the Go backend.

---

## Non-Functional Requirements

### Security

- NFR-SEC-1: Passwords are hashed with bcrypt (cost 10).
- NFR-SEC-2: JWTs are signed with HS256; the secret must be at least 32 characters.
- NFR-SEC-3: Role-based access control is enforced server-side on every protected route.
- NFR-SEC-4: The AI service API key is never included in error responses.
- NFR-SEC-5: Unknown JSON fields in request bodies are rejected (DisallowUnknownFields).
- NFR-SEC-6: All UUIDs in path parameters are validated before database queries.

### Performance

- NFR-PERF-1: AI classification requests time out after 5 seconds.
- NFR-PERF-2: Database connections use a pgx/v5 connection pool.
- NFR-PERF-3: The HTTP server has read (10s), write (15s), and idle (60s) timeouts.

### Reliability

- NFR-REL-1: The server handles panics gracefully via recovery middleware (returns 500).
- NFR-REL-2: Graceful shutdown drains in-flight requests within 10 seconds on SIGTERM/SIGINT.
- NFR-REL-3: Ticket assignment and soft-delete use database transactions with row-level locking.

### Observability

- NFR-OBS-1: Every request is assigned a unique request ID (X-Request-ID header).
- NFR-OBS-2: Structured JSON logs (slog) include request ID, method, path, status, and duration.
- NFR-OBS-3: AI classification failures are logged as warnings with ticket ID and error.

### Rate Limiting

- NFR-RATE-1: Login is limited to 5 requests/minute per instance.
- NFR-RATE-2: Registration is limited to 3 requests/minute per instance.
- NFR-RATE-3: All other API routes are limited to 100 requests/minute per instance.

---

## Constraints

- CON-1: The Go backend requires PostgreSQL 17 with migration version 3 applied.
- CON-2: The AI service uses a TF-IDF + Logistic Regression pipeline (scikit-learn) trained on a synthetic dataset of 1,597 tickets. Three independent models predict category, priority, and sentiment.
- CON-3: `suggested_team_id` and `response_suggestion` fields exist in the schema but are not populated by the current classifier.
- CON-4: The `manager` role is stored in the database but has no additional route permissions beyond `agent` in the current implementation.
- CON-5: Comment listing has no pagination (returns all comments for a ticket).
- CON-6: Team listing has no pagination (returns all teams).

---

## Out of Scope (Current Implementation)

- Customer-facing portal or external ticket submission API.
- Email/webhook integrations.
- Calibrated model probabilities (current confidence scores are raw `predict_proba` outputs, not calibrated posteriors).
- Suggested team assignment from AI predictions.
- Automated response suggestions.
- Multi-tenancy.
- Audit log for user management actions.
