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
			confidence,
			suggested_team_id,
			response_suggestion
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
		prediction.SuggestedTeamID,
		prediction.ResponseSuggestion,
	).Scan(
		&prediction.ID,
		&prediction.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create AI prediction: %w", err)
	}

	return nil
}

// GetLatestByTicketID retrieves the latest AI prediction for a ticket.
func (r *AIPredictionRepository) GetLatestByTicketID(
	ctx context.Context,
	ticketID string,
) (*models.AIPrediction, error) {
	query := `
		SELECT
			id,
			ticket_id,
			model_version,
			category,
			predicted_priority,
			sentiment,
			confidence,
			suggested_team_id,
			response_suggestion,
			created_at
		FROM ai_predictions
		WHERE ticket_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var prediction models.AIPrediction

	err := r.db.QueryRow(ctx, query, ticketID).Scan(
		&prediction.ID,
		&prediction.TicketID,
		&prediction.ModelVersion,
		&prediction.Category,
		&prediction.PredictedPriority,
		&prediction.Sentiment,
		&prediction.Confidence,
		&prediction.SuggestedTeamID,
		&prediction.ResponseSuggestion,
		&prediction.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AI prediction: %w", err)
	}

	return &prediction, nil
}
