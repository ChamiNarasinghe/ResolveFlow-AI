package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/google/uuid"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5"
)

// Mock comment repository for handler tests.
type mockCommentRepository struct {
	comments  []models.Comment
	createErr error
	listErr   error
}

func (m *mockCommentRepository) Create(
	ctx context.Context,
	comment *models.Comment,
) error {
	if m.createErr != nil {
		return m.createErr
	}

	comment.ID = "test-comment-id"
	comment.CreatedAt = time.Now()
	m.comments = append(m.comments, *comment)

	return nil
}

func (m *mockCommentRepository) ListByTicket(
	ctx context.Context,
	ticketID string,
) ([]models.Comment, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}

	return m.comments, nil
}

// Mock ticket repository for handler tests.
type mockCommentTicketRepository struct {
	ticket *models.Ticket
	err    error
}

func (m *mockCommentTicketRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Ticket, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.ticket == nil {
		return nil, pgx.ErrNoRows
	}

	return m.ticket, nil
}

var errMockCommentRepository = errors.New("mock comment repository error")

func TestCreateCommentSuccess(t *testing.T) {
	commentRepo := &mockCommentRepository{}

	ticketID := uuid.NewString()

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{
			ID: ticketID,
		},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	requestBody := `{"body":"I am investigating this issue.","is_internal":false}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(requestBody),
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	authMiddleware := middleware.AuthMiddleware(jwtService)
	authMiddleware(http.HandlerFunc(handler.CreateComment)).
		ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"Expected status %d, got %d. Response: %s",
			http.StatusCreated,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if len(commentRepo.comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(commentRepo.comments))
	}

	savedComment := commentRepo.comments[0]

	if savedComment.Body != "I am investigating this issue." {
		t.Errorf("Unexpected comment body: %s", savedComment.Body)
	}

	if savedComment.AuthorType == nil || *savedComment.AuthorType != "agent" {
		t.Error("Expected author type to be agent")
	}

	if savedComment.IsInternal {
		t.Error("Expected a public comment, but got an internal comment")
	}
}

func TestCreateInternalCommentSuccess(t *testing.T) {
	commentRepo := &mockCommentRepository{}
	ticketID := uuid.NewString()

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	requestBody := `{"body":"Internal investigation note.","is_internal":true}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(requestBody),
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.CreateComment),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s",
			recorder.Code, recorder.Body.String())
	}

	if len(commentRepo.comments) != 1 {
		t.Fatalf("Expected 1 comment, got %d", len(commentRepo.comments))
	}

	if !commentRepo.comments[0].IsInternal {
		t.Error("Expected comment to be marked as internal")
	}
}

func TestCreateCommentWithoutAuthentication(t *testing.T) {
	commentRepo := &mockCommentRepository{}
	ticketID := uuid.NewString()

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(`{"body":"Unauthorized comment"}`),
	)

	req.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()

	// No JWT middleware token is supplied.
	// The handler is called directly, so it should reject missing claims.
	handler.CreateComment(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Comment should not be saved without authentication")
	}
}

func TestCreateCommentInvalidTicketID(t *testing.T) {
	commentRepo := &mockCommentRepository{}

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/invalid-id/comments",
		strings.NewReader(`{"body":"Test comment"}`),
	)

	req.SetPathValue("id", "invalid-id")

	recorder := httptest.NewRecorder()

	handler.CreateComment(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Comment should not be saved with an invalid ticket ID")
	}
}

func TestCreateCommentTicketNotFound(t *testing.T) {
	commentRepo := &mockCommentRepository{}

	ticketRepo := &mockCommentTicketRepository{
		ticket: nil,
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	ticketID := uuid.NewString()

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(`{"body":"Test comment"}`),
	)

	req.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()

	handler.CreateComment(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Comment should not be saved for a missing ticket")
	}
}

func TestCreateCommentEmptyBody(t *testing.T) {
	commentRepo := &mockCommentRepository{}
	ticketID := uuid.NewString()

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(`{"body":"   ","is_internal":false}`),
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.CreateComment),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Empty comment should not be saved")
	}
}

func TestCreateCommentTooLong(t *testing.T) {
	commentRepo := &mockCommentRepository{}
	ticketID := uuid.NewString()

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	longBody := strings.Repeat("a", 5001)
	requestBody := `{"body":"` + longBody + `","is_internal":false}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(requestBody),
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.CreateComment),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Overlength comment should not be saved")
	}
}

func TestListCommentsSuccess(t *testing.T) {
	ticketID := uuid.NewString()
	authorID := "test-user-id"
	authorType := "agent"

	commentRepo := &mockCommentRepository{
		comments: []models.Comment{
			{
				ID:         uuid.NewString(),
				TicketID:   ticketID,
				AuthorID:   &authorID,
				AuthorType: &authorType,
				Body:       "First comment",
				IsInternal: false,
				CreatedAt:  time.Now(),
			},
			{
				ID:         uuid.NewString(),
				TicketID:   ticketID,
				AuthorID:   &authorID,
				AuthorType: &authorType,
				Body:       "Internal investigation note",
				IsInternal: true,
				CreatedAt:  time.Now(),
			},
		},
	}

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/comments",
		nil,
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.ListComments),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s",
			recorder.Code, recorder.Body.String())
	}

	var comments []models.Comment

	if err := json.Unmarshal(recorder.Body.Bytes(), &comments); err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	if len(comments) != 2 {
		t.Fatalf("Expected 2 comments, got %d", len(comments))
	}

	if comments[0].Body != "First comment" {
		t.Errorf("Unexpected first comment: %s", comments[0].Body)
	}

	if !comments[1].IsInternal {
		t.Error("Expected second comment to be internal")
	}
}

func TestListCommentsEmpty(t *testing.T) {
	ticketID := uuid.NewString()

	commentRepo := &mockCommentRepository{
		comments: []models.Comment{},
	}

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/comments",
		nil,
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.ListComments),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", recorder.Code)
	}

	var comments []models.Comment

	if err := json.Unmarshal(recorder.Body.Bytes(), &comments); err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	if comments == nil {
		t.Error("Expected an empty JSON array, got null")
	}

	if len(comments) != 0 {
		t.Errorf("Expected 0 comments, got %d", len(comments))
	}
}

func TestListCommentsInvalidTicketID(t *testing.T) {
	commentRepo := &mockCommentRepository{}

	ticketRepo := &mockCommentTicketRepository{}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/invalid-id/comments",
		nil,
	)

	req.SetPathValue("id", "invalid-id")

	recorder := httptest.NewRecorder()

	handler.ListComments(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", recorder.Code)
	}
}

func TestListCommentsTicketNotFound(t *testing.T) {
	ticketID := uuid.NewString()

	commentRepo := &mockCommentRepository{}

	ticketRepo := &mockCommentTicketRepository{
		ticket: nil,
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/comments",
		nil,
	)

	req.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()

	handler.ListComments(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", recorder.Code)
	}
}

func TestCreateCommentRepositoryError(t *testing.T) {
	ticketID := uuid.NewString()

	commentRepo := &mockCommentRepository{
		createErr: errMockCommentRepository,
	}

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"test-user-id",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tickets/"+ticketID+"/comments",
		strings.NewReader(`{"body":"Test database error","is_internal":false}`),
	)

	req.SetPathValue("id", ticketID)
	req.Header.Set("Authorization", "Bearer "+token)

	recorder := httptest.NewRecorder()

	middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.CreateComment),
	).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}

	if len(commentRepo.comments) != 0 {
		t.Error("Comment should not be saved when repository returns an error")
	}
}

func TestListCommentsRepositoryError(t *testing.T) {
	ticketID := uuid.NewString()

	commentRepo := &mockCommentRepository{
		listErr: errMockCommentRepository,
	}

	ticketRepo := &mockCommentTicketRepository{
		ticket: &models.Ticket{ID: ticketID},
	}

	handler := NewCommentHandler(commentRepo, ticketRepo)

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/comments",
		nil,
	)

	req.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()

	handler.ListComments(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", recorder.Code)
	}
}
