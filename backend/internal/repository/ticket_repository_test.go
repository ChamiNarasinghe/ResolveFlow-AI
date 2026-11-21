package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func createTestUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	email := fmt.Sprintf("ticket-test-%d@example.com", time.Now().UnixNano())

	var userID string

	err := pool.QueryRow(
		context.Background(),
		`INSERT INTO users (full_name, email, password_hash, role)
         VALUES ($1, $2, $3, $4)
         RETURNING id`,
		"Ticket Test Agent",
		email,
		"test-password-hash",
		"agent",
	).Scan(&userID)

	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			context.Background(),
			"DELETE FROM users WHERE id = $1",
			userID,
		)
		if err != nil {
			t.Errorf("failed to clean up test user: %v", err)
		}
	})

	return userID
}

func TestTicketRepositoryCreate(t *testing.T) {
	repo, pool := setupTestRepository(t)

	ctx := context.Background()
	actorID := createTestUser(t, pool)

	request := tickets.CreateTicketRequest{
		Subject:       "Integration test payment issue",
		Description:   "Testing ticket creation in PostgreSQL.",
		CustomerName:  "Test Customer",
		CustomerEmail: "integration-test@example.com",
		Channel:       "web",
	}

	result, err := repo.Create(ctx, request, actorID)
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

	if result.CreatedBy == nil || *result.CreatedBy != actorID {
		t.Errorf("expected created_by %s, got %v", actorID, result.CreatedBy)
	}

	// Remove the test record after verification.
	_, err = pool.Exec(ctx, "DELETE FROM tickets WHERE id = $1", result.ID)
	if err != nil {
		t.Errorf("failed to clean up test ticket: %v", err)
	}
}

func TestTicketRepositoryCreateDatabaseError(t *testing.T) {
	repo, pool := setupTestRepository(t)
	actorID := createTestUser(t, pool)

	request := tickets.CreateTicketRequest{
		Subject:       "Invalid channel test",
		Description:   "This should fail at the database.",
		CustomerName:  "Test Customer",
		CustomerEmail: "integration-test@example.com",
		Channel:       "fax",
	}

	result, err := repo.Create(context.Background(), request, actorID)

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

	actorID := createTestUser(t, pool)

	request := tickets.CreateTicketRequest{
		Subject:       "Ticket update integration test",
		Description:   "Testing PostgreSQL ticket updates.",
		CustomerName:  "Update Test Customer",
		CustomerEmail: "update-test@example.com",
		Channel:       "web",
	}

	ticket, err := repo.Create(context.Background(), request, actorID)
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

	actorID := createTestUser(t, pool)
	id := createTicketForUpdateTest(t, repo, pool)

	status := "in_progress"
	priority := "high"

	request := tickets.UpdateTicketRequest{
		Status:   tickets.OptionalString{Set: true, Value: &status},
		Priority: tickets.OptionalString{Set: true, Value: &priority},
	}

	result, err := repo.Update(ctx, id, actorID, request)
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

	actorID := createTestUser(t, pool)
	id := createTicketForUpdateTest(t, repo, pool)

	category := "Payment"

	request := tickets.UpdateTicketRequest{
		Category: tickets.OptionalString{
			Set:   true,
			Value: &category,
		},
	}

	result, err := repo.Update(ctx, id, actorID, request)
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

	actorID := createTestUser(t, pool)
	id := createTicketForUpdateTest(t, repo, pool)

	category := "Payment"

	_, err := repo.Update(ctx, id, actorID, tickets.UpdateTicketRequest{
		Category: tickets.OptionalString{
			Set:   true,
			Value: &category,
		},
	})
	if err != nil {
		t.Fatalf("failed to set category: %v", err)
	}

	result, err := repo.Update(ctx, id, actorID, tickets.UpdateTicketRequest{
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

	actorID := createTestUser(t, pool)
	id := createTicketForUpdateTest(t, repo, pool)

	resolved := "resolved"

	result, err := repo.Update(ctx, id, actorID, tickets.UpdateTicketRequest{
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

	result, err = repo.Update(ctx, id, actorID, tickets.UpdateTicketRequest{
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
	repo, pool := setupTestRepository(t)

	actorID := createTestUser(t, pool)
	id := "00000000-0000-4000-8000-000000000000"
	status := "in_progress"

	_, err := repo.Update(
		context.Background(),
		id,
		actorID,
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

func TestTicketRepositoryUpdateCreatesAuditHistory(t *testing.T) {
	repo, pool := setupTestRepository(t)
	ctx := context.Background()

	actorID := createTestUser(t, pool)
	ticketID := createTicketForUpdateTest(t, repo, pool)

	status := "in_progress"
	priority := "high"

	_, err := repo.Update(ctx, ticketID, actorID, tickets.UpdateTicketRequest{
		Status:   tickets.OptionalString{Set: true, Value: &status},
		Priority: tickets.OptionalString{Set: true, Value: &priority},
	})
	if err != nil {
		t.Fatalf("failed to update ticket: %v", err)
	}

	rows, err := pool.Query(ctx,
		`SELECT actor_id, action, old_value, new_value
		   FROM ticket_history
		  WHERE ticket_id = $1::uuid`, ticketID)
	if err != nil {
		t.Fatalf("failed to query ticket_history: %v", err)
	}
	defer rows.Close()

	type historyRow struct {
		actorID  string
		action   string
		oldValue []byte
		newValue []byte
	}

	var records []historyRow
	for rows.Next() {
		var r historyRow
		if err := rows.Scan(&r.actorID, &r.action, &r.oldValue, &r.newValue); err != nil {
			t.Fatalf("failed to scan history row: %v", err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("row iteration error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}

	rec := records[0]

	if rec.actorID != actorID {
		t.Errorf("expected actor_id %s, got %s", actorID, rec.actorID)
	}

	if rec.action != "ticket.updated" {
		t.Errorf("expected action ticket.updated, got %s", rec.action)
	}

	var oldMap, newMap map[string]any
	if err := json.Unmarshal(rec.oldValue, &oldMap); err != nil {
		t.Fatalf("failed to decode old_value: %v", err)
	}
	if err := json.Unmarshal(rec.newValue, &newMap); err != nil {
		t.Fatalf("failed to decode new_value: %v", err)
	}

	if oldMap["status"] != "open" {
		t.Errorf("expected old status open, got %v", oldMap["status"])
	}
	if oldMap["priority"] != "medium" {
		t.Errorf("expected old priority medium, got %v", oldMap["priority"])
	}
	if newMap["status"] != "in_progress" {
		t.Errorf("expected new status in_progress, got %v", newMap["status"])
	}
	if newMap["priority"] != "high" {
		t.Errorf("expected new priority high, got %v", newMap["priority"])
	}
}
