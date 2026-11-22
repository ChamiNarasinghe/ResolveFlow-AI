package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupAIPredictionRepository(t *testing.T) (*AIPredictionRepository, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("failed to ping PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	return NewAIPredictionRepository(pool), pool
}

// createTicketForAIPredictionTest creates a minimal ticket and returns its ID.
func createTicketForAIPredictionTest(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	userID := createTestUser(t, pool)

	var ticketID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO tickets (subject, description, customer_name, customer_email, channel, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		"AI prediction test ticket",
		"Testing AI prediction repository.",
		"AI Test Customer",
		fmt.Sprintf("ai-pred-test-%d@example.com", time.Now().UnixNano()),
		"web",
		userID,
	).Scan(&ticketID)
	if err != nil {
		t.Fatalf("failed to create ticket for AI prediction test: %v", err)
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM ai_predictions WHERE ticket_id = $1", ticketID) //nolint:errcheck
		pool.Exec(context.Background(), "DELETE FROM tickets WHERE id = $1", ticketID)               //nolint:errcheck
	})

	return ticketID
}

// TestAIPredictionCreateNullableFields verifies that a prediction with only
// required fields (all optional columns NULL) round-trips correctly.
func TestAIPredictionCreateNullableFields(t *testing.T) {
	repo, pool := setupAIPredictionRepository(t)
	ticketID := createTicketForAIPredictionTest(t, pool)

	prediction := &models.AIPrediction{
		TicketID:     ticketID,
		ModelVersion: "tfidf-logreg-v1",
		// Category, PredictedPriority, Sentiment, Confidence,
		// SuggestedTeamID, ResponseSuggestion all nil.
	}

	if err := repo.Create(context.Background(), prediction); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prediction.ID == "" {
		t.Error("expected non-empty ID after create")
	}
	if prediction.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at after create")
	}

	got, err := repo.GetLatestByTicketID(context.Background(), ticketID)
	if err != nil {
		t.Fatalf("unexpected error on get: %v", err)
	}

	if got.Category != nil {
		t.Errorf("expected nil category, got %v", *got.Category)
	}
	if got.PredictedPriority != nil {
		t.Errorf("expected nil predicted_priority, got %v", *got.PredictedPriority)
	}
	if got.Sentiment != nil {
		t.Errorf("expected nil sentiment, got %v", *got.Sentiment)
	}
	if got.Confidence != nil {
		t.Errorf("expected nil confidence, got %v", *got.Confidence)
	}
	if got.SuggestedTeamID != nil {
		t.Errorf("expected nil suggested_team_id, got %v", *got.SuggestedTeamID)
	}
	if got.ResponseSuggestion != nil {
		t.Errorf("expected nil response_suggestion, got %v", *got.ResponseSuggestion)
	}
}

// TestAIPredictionCreateAllFields verifies that all optional fields are
// persisted and retrieved correctly when populated.
func TestAIPredictionCreateAllFields(t *testing.T) {
	repo, pool := setupAIPredictionRepository(t)
	ticketID := createTicketForAIPredictionTest(t, pool)
	teamID := createTestTeam(t, pool)

	category := "billing"
	priority := "high"
	sentiment := "negative"
	confidence := 0.9250
	response := "Please check your payment method."

	prediction := &models.AIPrediction{
		TicketID:           ticketID,
		ModelVersion:       "tfidf-logreg-v1",
		Category:           &category,
		PredictedPriority:  &priority,
		Sentiment:          &sentiment,
		Confidence:         &confidence,
		SuggestedTeamID:    &teamID,
		ResponseSuggestion: &response,
	}

	if err := repo.Create(context.Background(), prediction); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := repo.GetLatestByTicketID(context.Background(), ticketID)
	if err != nil {
		t.Fatalf("unexpected error on get: %v", err)
	}

	if got.Category == nil || *got.Category != category {
		t.Errorf("expected category %q, got %v", category, got.Category)
	}
	if got.PredictedPriority == nil || *got.PredictedPriority != priority {
		t.Errorf("expected priority %q, got %v", priority, got.PredictedPriority)
	}
	if got.Sentiment == nil || *got.Sentiment != sentiment {
		t.Errorf("expected sentiment %q, got %v", sentiment, got.Sentiment)
	}
	if got.Confidence == nil || *got.Confidence != confidence {
		t.Errorf("expected confidence %v, got %v", confidence, got.Confidence)
	}
	if got.SuggestedTeamID == nil || *got.SuggestedTeamID != teamID {
		t.Errorf("expected suggested_team_id %q, got %v", teamID, got.SuggestedTeamID)
	}
	if got.ResponseSuggestion == nil || *got.ResponseSuggestion != response {
		t.Errorf("expected response_suggestion %q, got %v", response, got.ResponseSuggestion)
	}
}

// TestAIPredictionGetLatestReturnsNewest verifies that GetLatestByTicketID
// returns the most recently inserted prediction when multiple exist.
func TestAIPredictionGetLatestReturnsNewest(t *testing.T) {
	repo, pool := setupAIPredictionRepository(t)
	ticketID := createTicketForAIPredictionTest(t, pool)

	first := "technical"
	second := "billing"

	if err := repo.Create(context.Background(), &models.AIPrediction{
		TicketID:     ticketID,
		ModelVersion: "tfidf-logreg-v1",
		Category:     &first,
	}); err != nil {
		t.Fatalf("failed to create first prediction: %v", err)
	}

	if err := repo.Create(context.Background(), &models.AIPrediction{
		TicketID:     ticketID,
		ModelVersion: "rules-v2",
		Category:     &second,
	}); err != nil {
		t.Fatalf("failed to create second prediction: %v", err)
	}

	got, err := repo.GetLatestByTicketID(context.Background(), ticketID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ModelVersion != "rules-v2" {
		t.Errorf("expected latest model_version rules-v2, got %s", got.ModelVersion)
	}
	if got.Category == nil || *got.Category != second {
		t.Errorf("expected latest category %q, got %v", second, got.Category)
	}
}

// TestAIPredictionGetLatestNotFound verifies that a missing prediction
// returns an error (not a silent zero-value struct).
func TestAIPredictionGetLatestNotFound(t *testing.T) {
	repo, _ := setupAIPredictionRepository(t)

	missingTicketID := "00000000-0000-4000-8000-000000000000"
	_, err := repo.GetLatestByTicketID(context.Background(), missingTicketID)
	if err == nil {
		t.Fatal("expected error for ticket with no predictions, got nil")
	}
}
