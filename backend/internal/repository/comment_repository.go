package repository

import (
	"context"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	db *pgxpool.Pool
}

func NewCommentRepository(db *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create saves a new comment for a ticket.
func (r *CommentRepository) Create(
	ctx context.Context,
	comment *models.Comment,
) error {
	query := `
		INSERT INTO ticket_comments (
			ticket_id,
			author_id,
			author_type,
			body,
			is_internal
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		comment.TicketID,
		comment.AuthorID,
		comment.AuthorType,
		comment.Body,
		comment.IsInternal,
	).Scan(&comment.ID, &comment.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// ListByTicket retrieves comments belonging to a ticket.
func (r *CommentRepository) ListByTicket(
	ctx context.Context,
	ticketID string,
) ([]models.Comment, error) {
	query := `
		SELECT
			id,
			ticket_id,
			author_id,
			author_type,
			body,
			is_internal,
			created_at
		FROM ticket_comments
		WHERE ticket_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := r.db.Query(ctx, query, ticketID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve comments: %w", err)
	}
	defer rows.Close()

	comments := make([]models.Comment, 0)

	for rows.Next() {
		var comment models.Comment

		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&comment.AuthorID,
			&comment.AuthorType,
			&comment.Body,
			&comment.IsInternal,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate comments: %w", err)
	}

	return comments, nil
}