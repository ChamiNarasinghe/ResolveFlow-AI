package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/chami2002/resolveflow-api/internal/tickets"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestRepository(t *testing.T) (*TicketRepository, *pgxpool.Pool) {
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

	return NewTicketRepository(pool), pool
}

func TestTicketRepositoryCreate(t *testing.T) {
	repo, pool := setupTestRepository(t)

	ctx := context.Background()

	request := tickets.CreateTicketRequest{
		Subject:       "Integration test payment issue",
		Description:   "Testing ticket creation in PostgreSQL.",
		CustomerName:  "Test Customer",
		CustomerEmail: "integration-test@example.com",
		Channel:       "web",
	}

	result, err := repo.Create(ctx, request)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	if result.ID == "" {
		t.Error("expected ticket ID, got empty string")
	}

	if result.TicketNumber <= 0 {
		t.Errorf("expected positive ticket number, got %d", result.TicketNumber)
	}

	if result.Status != "open" {
		t.Errorf("expected status open, got %s", result.Status)
	}

	if result.Priority != "medium" {
		t.Errorf("expected priority medium, got %s", result.Priority)
	}

	if result.Subject != request.Subject {
		t.Errorf("unexpected subject: %s", result.Subject)
	}

	// Remove the test record after verification.
	_, err = pool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", result.ID)
	if err != nil {
		t.Errorf("failed to clean up test ticket: %v", err)
	}
}

func TestTicketRepositoryCreateDatabaseError(t *testing.T) {
	repo, _ := setupTestRepository(t)

	request := tickets.CreateTicketRequest{
		Subject:       "Invalid channel test",
		Description:   "This should fail at the database.",
		CustomerName:  "Test Customer",
		CustomerEmail: "integration-test@example.com",
		Channel:       "fax",
	}

	result, err := repo.Create(context.Background(), request)

	if err == nil {
		t.Fatal("expected database error for invalid channel")
	}

	if result != nil {
		t.Errorf("expected nil ticket, got %+v", result)
	}
}

func createTicketForUpdateTest(
	t *testing.T,
	repo *TicketRepository,
	pool *pgxpool.Pool,
) string {
	t.Helper()

	request := tickets.CreateTicketRequest{
		Subject:       "Ticket update integration test",
		Description:   "Testing PostgreSQL ticket updates.",
		CustomerName:  "Update Test Customer",
		CustomerEmail: "update-test@example.com",
		Channel:       "web",
	}

	ticket, err := repo.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("failed to create test ticket: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM tickets WHERE id = $1",
			ticket.ID,
		)
		if err != nil {
			t.Errorf("failed to clean up test ticket: %v", err)
		}
	})

	return ticket.ID
}

func TestTicketRepositoryUpdateStatusAndPriority(t *testing.T) {
	repo, pool := setupTestRepository(t)
	ctx := context.Background()

	id := createTicketForUpdateTest(t, repo, pool)

	status := "in_progress"
	priority := "high"

	request := tickets.UpdateTicketRequest{
		Status:   tickets.OptionalString{Set: true, Value: &status},
		Priority: tickets.OptionalString{Set: true, Value: &priority},
	}

	result, err := repo.Update(ctx, id, request)
	if err != nil {
		t.Fatalf("failed to update ticket: %v", err)
	}

	if result.Status != status {
		t.Errorf("expected status %s, got %s", status, result.Status)
	}

	if result.Priority != priority {
		t.Errorf("expected priority %s, got %s", priority, result.Priority)
	}
}

func TestTicketRepositoryUpdateCategory(t *testing.T) {
	repo, pool := setupTestRepository(t)
	ctx := context.Background()

	id := createTicketForUpdateTest(t, repo, pool)

	category := "Payment"

	request := tickets.UpdateTicketRequest{
		Category: tickets.OptionalString{
			Set:   true,
			Value: &category,
		},
	}

	result, err := repo.Update(ctx, id, request)
	if err != nil {
		t.Fatalf("failed to update category: %v", err)
	}

	if result.Category == nil || *result.Category != category {
		t.Errorf("expected category %s", category)
	}
}

func TestTicketRepositoryClearCategory(t *testing.T) {
	repo, pool := setupTestRepository(t)
	ctx := context.Background()

	id := createTicketForUpdateTest(t, repo, pool)

	category := "Payment"

	_, err := repo.Update(ctx, id, tickets.UpdateTicketRequest{
		Category: tickets.OptionalString{
			Set:   true,
			Value: &category,
		},
	})
	if err != nil {
		t.Fatalf("failed to set category: %v", err)
	}

	result, err := repo.Update(ctx, id, tickets.UpdateTicketRequest{
		Category: tickets.OptionalString{
			Set:   true,
			Value: nil,
		},
	})
	if err != nil {
		t.Fatalf("failed to clear category: %v", err)
	}

	if result.Category != nil {
		t.Errorf("expected category to be nil, got %v", *result.Category)
	}
}

func TestTicketRepositoryUpdateResolutionStatus(t *testing.T) {
	repo, pool := setupTestRepository(t)
	ctx := context.Background()

	id := createTicketForUpdateTest(t, repo, pool)

	resolved := "resolved"

	result, err := repo.Update(ctx, id, tickets.UpdateTicketRequest{
		Status: tickets.OptionalString{
			Set:   true,
			Value: &resolved,
		},
	})
	if err != nil {
		t.Fatalf("failed to resolve ticket: %v", err)
	}

	if result.ResolvedAt == nil {
		t.Fatal("expected resolved_at to be set")
	}

	open := "open"

	result, err = repo.Update(ctx, id, tickets.UpdateTicketRequest{
		Status: tickets.OptionalString{
			Set:   true,
			Value: &open,
		},
	})
	if err != nil {
		t.Fatalf("failed to reopen ticket: %v", err)
	}

	if result.ResolvedAt != nil {
		t.Error("expected resolved_at to be cleared when reopening ticket")
	}
}

func TestTicketRepositoryUpdateNotFound(t *testing.T) {
	repo, _ := setupTestRepository(t)

	id := "00000000-0000-4000-8000-000000000000"
	status := "in_progress"

	_, err := repo.Update(
		context.Background(),
		id,
		tickets.UpdateTicketRequest{
			Status: tickets.OptionalString{
				Set:   true,
				Value: &status,
			},
		},
	)

	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}
}
