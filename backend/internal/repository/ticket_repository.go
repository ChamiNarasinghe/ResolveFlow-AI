package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/tickets"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TicketRepository handles database operations for tickets.
type TicketRepository struct {
	db *pgxpool.Pool
}

// NewTicketRepository creates a ticket repository.
func NewTicketRepository(db *pgxpool.Pool) *TicketRepository {
	return &TicketRepository{db: db}
}

// Create inserts a new ticket into PostgreSQL.
func (r *TicketRepository) Create(
	ctx context.Context,
	req tickets.CreateTicketRequest,
	creatorID string,
) (*models.Ticket, error) {
	query := `
		INSERT INTO tickets (
			subject,
			description,
			customer_name,
			customer_email,
			channel,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id,
			ticket_number,
			subject,
			description,
			customer_name,
			customer_email,
			status,
			priority,
			channel,
			category,
			assigned_team_id,
			assigned_to,
			created_by,
			created_at,
			updated_at,
			resolved_at
	`

	var ticket models.Ticket

	err := r.db.QueryRow(
		ctx,
		query,
		req.Subject,
		req.Description,
		req.CustomerName,
		req.CustomerEmail,
		req.Channel,
		creatorID,
	).Scan(
		&ticket.ID,
		&ticket.TicketNumber,
		&ticket.Subject,
		&ticket.Description,
		&ticket.CustomerName,
		&ticket.CustomerEmail,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Channel,
		&ticket.Category,
		&ticket.AssignedTeamID,
		&ticket.AssignedTo,
		&ticket.CreatedBy,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ResolvedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	return &ticket, nil
}

// List retrieves a page of tickets and the total matching count.
func (r *TicketRepository) List(
	ctx context.Context,
	page int,
	pageSize int,
	status string,
	priority string,
) ([]models.Ticket, int, error) {
	offset := (page - 1) * pageSize

	// Count total matching rows independently of the page offset so that an
	// out-of-range page still returns the correct total.
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		   FROM tickets
		  WHERE ($1 = '' OR status = $1)
		    AND ($2 = '' OR priority = $2)`,
		status, priority,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tickets: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT
			id,
			ticket_number,
			subject,
			description,
			customer_name,
			customer_email,
			status,
			priority,
			channel,
			category,
			assigned_team_id,
			assigned_to,
			created_at,
			updated_at,
			resolved_at
		FROM tickets
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR priority = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`,
		status, priority, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve tickets: %w", err)
	}
	defer rows.Close()

	var ticketsList []models.Ticket

	for rows.Next() {
		var ticket models.Ticket

		if err := rows.Scan(
			&ticket.ID,
			&ticket.TicketNumber,
			&ticket.Subject,
			&ticket.Description,
			&ticket.CustomerName,
			&ticket.CustomerEmail,
			&ticket.Status,
			&ticket.Priority,
			&ticket.Channel,
			&ticket.Category,
			&ticket.AssignedTeamID,
			&ticket.AssignedTo,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
			&ticket.ResolvedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan ticket: %w", err)
		}

		ticketsList = append(ticketsList, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate tickets: %w", err)
	}

	return ticketsList, total, nil
}

// GetByID retrieves a ticket using its UUID.
func (r *TicketRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Ticket, error) {
	query := `
		SELECT
			id,
			ticket_number,
			subject,
			description,
			customer_name,
			customer_email,
			status,
			priority,
			channel,
			category,
			assigned_team_id,
			assigned_to,
			created_at,
			updated_at,
			resolved_at
		FROM tickets
		WHERE id = $1
	`

	var ticket models.Ticket

	err := r.db.QueryRow(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.TicketNumber,
		&ticket.Subject,
		&ticket.Description,
		&ticket.CustomerName,
		&ticket.CustomerEmail,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Channel,
		&ticket.Category,
		&ticket.AssignedTeamID,
		&ticket.AssignedTo,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ResolvedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}

		return nil, fmt.Errorf("failed to retrieve ticket: %w", err)
	}

	return &ticket, nil
}

func (r *TicketRepository) Update(
	ctx context.Context,
	id string,
	actorID string,
	req tickets.UpdateTicketRequest,
) (*models.Ticket, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock the row and read current values for audit comparison.
	var old struct {
		status         string
		priority       string
		category       *string
		assignedTeamID *string
		assignedTo     *string
	}

	err = tx.QueryRow(ctx,
		`SELECT status, priority, category, assigned_team_id, assigned_to
		 FROM tickets WHERE id = $1::uuid FOR UPDATE`, id,
	).Scan(
		&old.status,
		&old.priority,
		&old.category,
		&old.assignedTeamID,
		&old.assignedTo,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("failed to lock ticket: %w", err)
	}

	// Execute the existing UPDATE statement using the transaction.
	updateQuery := `
		UPDATE tickets
		SET
			status = CASE
				WHEN $2::boolean THEN $3::varchar
				ELSE status
			END,
			priority = CASE
				WHEN $4::boolean THEN $5::varchar
				ELSE priority
			END,
			category = CASE
				WHEN $6::boolean THEN $7::varchar
				ELSE category
			END,
			assigned_team_id = CASE
				WHEN $8::boolean THEN $9::uuid
				ELSE assigned_team_id
			END,
			assigned_to = CASE
				WHEN $10::boolean THEN $11::uuid
				ELSE assigned_to
			END,
			resolved_at = CASE
				WHEN $2::boolean THEN
					CASE
						WHEN $3::varchar = 'resolved'
							THEN COALESCE(resolved_at, NOW())
						ELSE NULL
					END
				ELSE resolved_at
			END,
			updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING
			id,
			ticket_number,
			subject,
			description,
			customer_name,
			customer_email,
			status,
			priority,
			channel,
			category,
			assigned_team_id,
			assigned_to,
			created_at,
			updated_at,
			resolved_at
	`

	var ticket models.Ticket

	err = tx.QueryRow(
		ctx,
		updateQuery,
		id,
		req.Status.Set,
		optionalStringValue(req.Status),
		req.Priority.Set,
		optionalStringValue(req.Priority),
		req.Category.Set,
		optionalStringValue(req.Category),
		req.AssignedTeamID.Set,
		optionalStringValue(req.AssignedTeamID),
		req.AssignedTo.Set,
		optionalStringValue(req.AssignedTo),
	).Scan(
		&ticket.ID,
		&ticket.TicketNumber,
		&ticket.Subject,
		&ticket.Description,
		&ticket.CustomerName,
		&ticket.CustomerEmail,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Channel,
		&ticket.Category,
		&ticket.AssignedTeamID,
		&ticket.AssignedTo,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket: %w", err)
	}

	// Build audit diff — only include fields that actually changed.
	oldValues := map[string]any{}
	newValues := map[string]any{}

	if ticket.Status != old.status {
		oldValues["status"] = old.status
		newValues["status"] = ticket.Status
	}
	if ticket.Priority != old.priority {
		oldValues["priority"] = old.priority
		newValues["priority"] = ticket.Priority
	}
	if !stringPtrEqual(ticket.Category, old.category) {
		oldValues["category"] = old.category
		newValues["category"] = ticket.Category
	}
	if !stringPtrEqual(ticket.AssignedTeamID, old.assignedTeamID) {
		oldValues["assigned_team_id"] = old.assignedTeamID
		newValues["assigned_team_id"] = ticket.AssignedTeamID
	}
	if !stringPtrEqual(ticket.AssignedTo, old.assignedTo) {
		oldValues["assigned_to"] = old.assignedTo
		newValues["assigned_to"] = ticket.AssignedTo
	}

	if len(oldValues) > 0 {
		oldJSON, err := json.Marshal(oldValues)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize audit old_value: %w", err)
		}
		newJSON, err := json.Marshal(newValues)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize audit new_value: %w", err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO ticket_history
				(ticket_id, actor_id, action, old_value, new_value)
			 VALUES ($1, $2, 'ticket.updated', $3, $4)`,
			ticket.ID, actorID, oldJSON, newJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to insert audit history: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &ticket, nil
}

// GetHistoryByTicketID retrieves paginated audit history for a ticket, newest first.
func (r *TicketRepository) GetHistoryByTicketID(
	ctx context.Context,
	ticketID string,
	page int,
	pageSize int,
) ([]models.TicketHistory, error) {
	offset := (page - 1) * pageSize

	rows, err := r.db.Query(ctx, `
		SELECT id, ticket_id, actor_id, action, old_value, new_value, created_at
		FROM ticket_history
		WHERE ticket_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		ticketID, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query ticket history: %w", err)
	}
	defer rows.Close()

	var records []models.TicketHistory
	for rows.Next() {
		var h models.TicketHistory
		if err := rows.Scan(
			&h.ID, &h.TicketID, &h.ActorID, &h.Action, &h.OldValue, &h.NewValue, &h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}
		records = append(records, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate history rows: %w", err)
	}

	return records, nil
}

// ErrTicketNotFound is returned when a ticket lookup finds no row.
var ErrTicketNotFound = errors.New("ticket not found")

// AssignTeam updates assigned_team_id for a ticket inside a transaction and
// records a ticket.assigned or ticket.unassigned history entry.
// Passing nil teamID unassigns the ticket.
// If the ticket is already assigned to the same team, no change is made and
// no history record is written.
func (r *TicketRepository) AssignTeam(
	ctx context.Context,
	ticketID string,
	teamID *string,
	actorID string,
) (*models.Ticket, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Lock the row and read the current assigned_team_id.
	var current models.Ticket
	err = tx.QueryRow(ctx,
		`SELECT id, ticket_number, subject, description, customer_name, customer_email,
		        status, priority, channel, category, assigned_team_id, assigned_to,
		        created_by, created_at, updated_at, resolved_at
		   FROM tickets WHERE id = $1::uuid FOR UPDATE`, ticketID,
	).Scan(
		&current.ID, &current.TicketNumber, &current.Subject, &current.Description,
		&current.CustomerName, &current.CustomerEmail, &current.Status, &current.Priority,
		&current.Channel, &current.Category, &current.AssignedTeamID, &current.AssignedTo,
		&current.CreatedBy, &current.CreatedAt, &current.UpdatedAt, &current.ResolvedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTicketNotFound
		}
		return nil, fmt.Errorf("failed to lock ticket: %w", err)
	}

	// No-op: same team already assigned.
	if stringPtrEqual(current.AssignedTeamID, teamID) {
		return &current, nil
	}

	// Verify the team exists when assigning (not when unassigning).
	if teamID != nil {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1::uuid)`, *teamID,
		).Scan(&exists); err != nil {
			return nil, fmt.Errorf("failed to verify team: %w", err)
		}
		if !exists {
			return nil, ErrTeamNotFound
		}
	}

	// Update the ticket.
	var updated models.Ticket
	err = tx.QueryRow(ctx,
		`UPDATE tickets SET assigned_team_id = $2, updated_at = NOW()
		  WHERE id = $1::uuid
		  RETURNING id, ticket_number, subject, description, customer_name, customer_email,
		            status, priority, channel, category, assigned_team_id, assigned_to,
		            created_by, created_at, updated_at, resolved_at`,
		ticketID, teamID,
	).Scan(
		&updated.ID, &updated.TicketNumber, &updated.Subject, &updated.Description,
		&updated.CustomerName, &updated.CustomerEmail, &updated.Status, &updated.Priority,
		&updated.Channel, &updated.Category, &updated.AssignedTeamID, &updated.AssignedTo,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt, &updated.ResolvedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update ticket assignment: %w", err)
	}

	action := "ticket.assigned"
	if teamID == nil {
		action = "ticket.unassigned"
	}

	oldJSON, err := json.Marshal(map[string]any{"assigned_team_id": current.AssignedTeamID})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audit old_value: %w", err)
	}
	newJSON, err := json.Marshal(map[string]any{"assigned_team_id": updated.AssignedTeamID})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize audit new_value: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO ticket_history (ticket_id, actor_id, action, old_value, new_value)
		 VALUES ($1, $2, $3, $4, $5)`,
		ticketID, actorID, action, oldJSON, newJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert audit history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &updated, nil
}

// CountHistoryByTicketID returns the total number of history records for a ticket.
func (r *TicketRepository) CountHistoryByTicketID(
	ctx context.Context,
	ticketID string,
) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM ticket_history WHERE ticket_id = $1::uuid`,
		ticketID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count ticket history: %w", err)
	}
	return count, nil
}

// stringPtrEqual compares two nullable strings for equality.
func stringPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func optionalStringValue(field tickets.OptionalString) any {
	if !field.Set || field.Value == nil {
		return nil
	}

	return *field.Value
}
