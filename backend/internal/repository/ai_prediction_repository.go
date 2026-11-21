package repository

import (
	"context"
	"fmt"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIPredictionRepository handles AI prediction database operations.
type AIPredictionRepository struct {
	db *pgxpool.Pool
}

// NewAIPredictionRepository creates an AI prediction repository.
func NewAIPredictionRepository(db *pgxpool.Pool) *AIPredictionRepository {
	return &AIPredictionRepository{db: db}
}

// Create saves an AI prediction for a ticket.
func (r *AIPredictionRepository) Create(
	ctx context.Context,
	prediction *models.AIPrediction,
) error {
	query := `
		INSERT INTO ai_predictions (
			ticket_id,
			model_version,
			category,
			predicted_priority,
			sentiment,
			confidence
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		prediction.TicketID,
		prediction.ModelVersion,
		prediction.Category,
		prediction.PredictedPriority,
		prediction.Sentiment,
		prediction.Confidence,
	).Scan(
		&prediction.ID,
		&prediction.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create AI prediction: %w", err)
	}

	return nil
}
