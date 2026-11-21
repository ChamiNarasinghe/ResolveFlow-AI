package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/tickets"
)

// mockTicketRepository simulates the ticket repository for handler tests.
type mockTicketRepository struct {
	listResult []models.Ticket

	gotPage     int
	gotLimit    int
	gotStatus   string
	gotPriority string
	listCalled  bool

	getByIDResult *models.Ticket
	getByIDError  error

	updateResult *models.Ticket
	updateError  error
	updateCalled bool
	gotUpdateID  string
	gotUpdateReq tickets.UpdateTicketRequest
}

func (m *mockTicketRepository) Create(
	ctx context.Context,
	req tickets.CreateTicketRequest,
) (*models.Ticket, error) {
	return nil, nil
}

func (m *mockTicketRepository) GetByID(
	ctx context.Context,
	id string,
) (*models.Ticket, error) {
	return m.getByIDResult, m.getByIDError
}

func (m *mockTicketRepository) Update(
	ctx context.Context,
	id string,
	req tickets.UpdateTicketRequest,
) (*models.Ticket, error) {
	m.updateCalled = true
	m.gotUpdateID = id
	m.gotUpdateReq = req

	return m.updateResult, m.updateError
}

func (m *mockTicketRepository) List(
	ctx context.Context,
	page int,
	limit int,
	status string,
	priority string,
) ([]models.Ticket, error) {
	m.listCalled = true
	m.gotPage = page
	m.gotLimit = limit
	m.gotStatus = status
	m.gotPriority = priority

	return m.listResult, nil
}

// TestListTicketsDefaultPagination checks the default page and limit.
func TestListTicketsDefaultPagination(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	recorder := httptest.NewRecorder()

	handler.ListTickets(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if mockRepo.gotPage != 1 {
		t.Errorf("expected page 1, got %d", mockRepo.gotPage)
	}

	if mockRepo.gotLimit != 10 {
		t.Errorf("expected limit 10, got %d", mockRepo.gotLimit)
	}

	if !strings.Contains(recorder.Body.String(), `"pagination"`) {
		t.Error("expected pagination information in response")
	}
}

// TestListTicketsCustomPagination checks custom pagination parameters.
func TestListTicketsCustomPagination(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets?page=2&limit=5",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ListTickets(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if mockRepo.gotPage != 2 {
		t.Errorf("expected page 2, got %d", mockRepo.gotPage)
	}

	if mockRepo.gotLimit != 5 {
		t.Errorf("expected limit 5, got %d", mockRepo.gotLimit)
	}
}

// TestListTicketsFilters checks status and priority filters.
func TestListTicketsFilters(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets?status=OPEN&priority=HIGH",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ListTickets(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if mockRepo.gotStatus != "open" {
		t.Errorf("expected status open, got %s", mockRepo.gotStatus)
	}

	if mockRepo.gotPriority != "high" {
		t.Errorf("expected priority high, got %s", mockRepo.gotPriority)
	}
}

// TestListTicketsInvalidStatus rejects unsupported statuses.
func TestListTicketsInvalidStatus(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets?status=pending",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ListTickets(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	if mockRepo.listCalled {
		t.Error("repository should not be called for an invalid status")
	}
}

// TestListTicketsInvalidPriority rejects unsupported priorities.
func TestListTicketsInvalidPriority(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets?priority=critical",
		nil,
	)
	recorder := httptest.NewRecorder()

	handler.ListTickets(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	if mockRepo.listCalled {
		t.Error("repository should not be called for an invalid priority")
	}
}

// TestListTicketsInvalidPagination rejects invalid page and limit values.
func TestListTicketsInvalidPagination(t *testing.T) {
	testCases := []string{
		"/api/v1/tickets?page=0",
		"/api/v1/tickets?page=abc",
		"/api/v1/tickets?limit=0",
		"/api/v1/tickets?limit=101",
	}

	for _, testURL := range testCases {
		t.Run(testURL, func(t *testing.T) {
			mockRepo := &mockTicketRepository{}
			handler := NewTicketHandler(mockRepo)

			request := httptest.NewRequest(http.MethodGet, testURL, nil)
			recorder := httptest.NewRecorder()

			handler.ListTickets(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", recorder.Code)
			}

			if mockRepo.listCalled {
				t.Error("repository should not be called for invalid pagination")
			}
		})
	}
}

func TestGetTicketSuccess(t *testing.T) {
	expectedTicket := &models.Ticket{
		ID:            "a2f66976-199f-4ec9-8774-21f937befe13",
		TicketNumber:  5,
		Subject:       "Payment failed",
		Description:   "Payment transaction failed",
		CustomerName:  "John Smith",
		CustomerEmail: "john@example.com",
		Status:        "open",
		Priority:      "medium",
		Channel:       "web",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mockRepo := &mockTicketRepository{
		getByIDResult: expectedTicket,
	}

	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+expectedTicket.ID,
		nil,
	)
	request.SetPathValue("id", expectedTicket.ID)

	recorder := httptest.NewRecorder()
	handler.GetTicket(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	if !strings.Contains(recorder.Body.String(), "Payment failed") {
		t.Error("expected ticket details in response")
	}
}

func TestGetTicketInvalidUUID(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/invalid-id",
		nil,
	)
	request.SetPathValue("id", "invalid-id")

	recorder := httptest.NewRecorder()
	handler.GetTicket(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDError: pgx.ErrNoRows,
	}

	handler := NewTicketHandler(mockRepo)

	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+id,
		nil,
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.GetTicket(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", recorder.Code)
	}
}

func TestGetTicketDatabaseError(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDError: errors.New("database connection failed"),
	}

	handler := NewTicketHandler(mockRepo)

	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+id,
		nil,
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.GetTicket(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", recorder.Code)
	}
}

func TestUpdateTicketSuccess(t *testing.T) {
	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	expectedTicket := &models.Ticket{
		ID:           id,
		TicketNumber: 5,
		Subject:      "Payment failed",
		Status:       "in_progress",
		Priority:     "high",
		Channel:      "web",
	}

	mockRepo := &mockTicketRepository{
		updateResult: expectedTicket,
	}

	handler := NewTicketHandler(mockRepo)

	body := `{"status":"in_progress","priority":"high"}`

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+id,
		strings.NewReader(body),
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	if !mockRepo.updateCalled {
		t.Fatal("expected repository Update to be called")
	}

	if mockRepo.gotUpdateID != id {
		t.Errorf("expected ticket ID %s, got %s", id, mockRepo.gotUpdateID)
	}

	if mockRepo.gotUpdateReq.Status.Value == nil ||
		*mockRepo.gotUpdateReq.Status.Value != "in_progress" {
		t.Error("expected status to be in_progress")
	}

	if !strings.Contains(recorder.Body.String(), `"status":"in_progress"`) {
		t.Error("expected updated ticket in response")
	}
}

func TestUpdateTicketInvalidUUID(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/invalid-id",
		strings.NewReader(`{"status":"resolved"}`),
	)
	request.SetPathValue("id", "invalid-id")

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	if mockRepo.updateCalled {
		t.Error("repository should not be called for an invalid UUID")
	}
}

func TestUpdateTicketInvalidJSON(t *testing.T) {
	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+id,
		strings.NewReader(`{"status":`),
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	if mockRepo.updateCalled {
		t.Error("repository should not be called for invalid JSON")
	}
}

func TestUpdateTicketInvalidFields(t *testing.T) {
	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	testCases := []struct {
		name string
		body string
	}{
		{name: "invalid status", body: `{"status":"pending"}`},
		{name: "invalid priority", body: `{"priority":"critical"}`},
		{name: "empty update", body: `{}`},
		{name: "unknown field", body: `{"unknown":"value"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockTicketRepository{}
			handler := NewTicketHandler(mockRepo)

			request := httptest.NewRequest(
				http.MethodPatch,
				"/api/v1/tickets/"+id,
				strings.NewReader(tc.body),
			)
			request.SetPathValue("id", id)

			recorder := httptest.NewRecorder()
			handler.UpdateTicket(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", recorder.Code)
			}

			if mockRepo.updateCalled {
				t.Error("repository should not be called for invalid fields")
			}
		})
	}
}

func TestUpdateTicketNotFound(t *testing.T) {
	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	mockRepo := &mockTicketRepository{
		updateError: pgx.ErrNoRows,
	}

	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+id,
		strings.NewReader(`{"status":"resolved"}`),
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", recorder.Code)
	}
}

func TestUpdateTicketDatabaseError(t *testing.T) {
	id := "a2f66976-199f-4ec9-8774-21f937befe13"

	mockRepo := &mockTicketRepository{
		updateError: errors.New("database connection failed"),
	}

	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/tickets/"+id,
		strings.NewReader(`{"status":"resolved"}`),
	)
	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", recorder.Code)
	}
}
