package repository

import (
	"context"
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
	req tickets.UpdateTicketRequest,
) (*models.Ticket, error) {
	query := `
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

	err := r.db.QueryRow(
		ctx,
		query,
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

	return &ticket, nil
}

func optionalStringValue(field tickets.OptionalString) any {
	if !field.Set || field.Value == nil {
		return nil
	}

	return *field.Value
}
