package repository

import (
	"context"
	"os"
	"testing"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupCommentTest(t *testing.T) (*CommentRepository, *pgxpool.Pool, string) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("failed to connect to database: %v", err)
	}

	var ticketID string

	err = pool.QueryRow(ctx, `
		INSERT INTO tickets (
			subject,
			description,
			customer_name,
			customer_email,
			status,
			priority,
			channel
		)
		VALUES (
			'Comment repository test',
			'Testing ticket comments',
			'Test Customer',
			'comment-test@example.com',
			'open',
			'medium',
			'web'
		)
		RETURNING id
	`).Scan(&ticketID)

	if err != nil {
		pool.Close()
		t.Fatalf("failed to create test ticket: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(ctx, `DELETE FROM tickets WHERE id = $1`, ticketID)
		if err != nil {
			t.Errorf("failed to clean up test ticket: %v", err)
		}
		pool.Close()
	})

	return NewCommentRepository(pool), pool, ticketID
}

func TestCommentRepositoryCreate(t *testing.T) {
	repo, _, ticketID := setupCommentTest(t)

	authorType := "agent"

	comment := &models.Comment{
		TicketID:   ticketID,
		AuthorType: &authorType,
		Body:       "I am investigating this issue.",
		IsInternal: false,
	}

	err := repo.Create(context.Background(), comment)
	if err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	if comment.ID == "" {
		t.Error("expected comment ID to be populated")
	}

	if comment.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be populated")
	}
}

func TestCommentRepositoryListByTicket(t *testing.T) {
	repo, _, ticketID := setupCommentTest(t)

	authorType := "agent"

	firstComment := &models.Comment{
		TicketID:   ticketID,
		AuthorType: &authorType,
		Body:       "First comment",
	}

	secondComment := &models.Comment{
		TicketID:   ticketID,
		AuthorType: &authorType,
		Body:       "Second comment",
		IsInternal: true,
	}

	for _, comment := range []*models.Comment{firstComment, secondComment} {
		if err := repo.Create(context.Background(), comment); err != nil {
			t.Fatalf("Create() returned an error: %v", err)
		}
	}

	comments, err := repo.ListByTicket(context.Background(), ticketID)
	if err != nil {
		t.Fatalf("ListByTicket() returned an error: %v", err)
	}

	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	if comments[0].Body != "First comment" {
		t.Errorf("expected first comment, got %q", comments[0].Body)
	}

	if comments[1].Body != "Second comment" {
		t.Errorf("expected second comment, got %q", comments[1].Body)
	}

	if !comments[1].IsInternal {
		t.Error("expected second comment to be internal")
	}
}