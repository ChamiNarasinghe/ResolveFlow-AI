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
) (*models.Ticket, error) {
	query := `
		INSERT INTO tickets (
			subject,
			description,
			customer_name,
			customer_email,
			channel
		)
		VALUES ($1, $2, $3, $4, $5)
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

	err := r.db.QueryRow(
		ctx,
		query,
		req.Subject,
		req.Description,
		req.CustomerName,
		req.CustomerEmail,
		req.Channel,
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
		return nil, fmt.Errorf("failed to create ticket: %w", err)
	}

	return &ticket, nil
}

// List retrieves all tickets, ordered from newest to oldest.

// List retrieves tickets with pagination and optional filters.
func (r *TicketRepository) List(
	ctx context.Context,
	page int,
	limit int,
	status string,
	priority string,
) ([]models.Ticket, error) {
	offset := (page - 1) * limit

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
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR priority = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(
		ctx,
		query,
		status,
		priority,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tickets: %w", err)
	}
	defer rows.Close()

	var ticketsList []models.Ticket

	for rows.Next() {
		var ticket models.Ticket

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan ticket: %w", err)
		}

		ticketsList = append(ticketsList, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate tickets: %w", err)
	}

	return ticketsList, nil
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
