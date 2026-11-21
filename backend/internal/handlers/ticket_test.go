package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/repository"
	"github.com/chami2002/resolveflow-api/internal/services"
	"github.com/chami2002/resolveflow-api/internal/tickets"
)

// withTestClaims injects fake JWT claims into a request context.
func withTestClaims(r *http.Request) *http.Request {
	claims := &auth.Claims{
		UserID: "test-actor-id",
		Email:  "test@example.com",
		Role:   "agent",
	}
	return r.WithContext(middleware.SetUserClaims(r.Context(), claims))
}

// mockTicketRepository simulates the ticket repository for handler tests.
type mockTicketRepository struct {
	listResult []models.Ticket
	listTotal  int
	listError  error

	gotPage     int
	gotPageSize int
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

	historyResult []models.TicketHistory
	historyError  error
	historyCount  int
	historyCountError error
	gotHistoryPage     int
	gotHistoryPageSize int

	assignResult      *models.Ticket
	assignError       error
	assignCalled      bool
	gotAssignTicketID string
	gotAssignTeamID   *string

	softDeleteError  error
	softDeleteCalled bool
	gotSoftDeleteID  string
}

func (m *mockTicketRepository) Create(
	ctx context.Context,
	req tickets.CreateTicketRequest,
	creatorID string,
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
	actorID string,
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
	pageSize int,
	status string,
	priority string,
) ([]models.Ticket, int, error) {
	m.listCalled = true
	m.gotPage = page
	m.gotPageSize = pageSize
	m.gotStatus = status
	m.gotPriority = priority

	return m.listResult, m.listTotal, m.listError
}

func (m *mockTicketRepository) GetHistoryByTicketID(
	ctx context.Context,
	ticketID string,
	page int,
	pageSize int,
) ([]models.TicketHistory, error) {
	m.gotHistoryPage = page
	m.gotHistoryPageSize = pageSize
	return m.historyResult, m.historyError
}

func (m *mockTicketRepository) CountHistoryByTicketID(
	ctx context.Context,
	ticketID string,
) (int, error) {
	return m.historyCount, m.historyCountError
}

func (m *mockTicketRepository) AssignTeam(
	ctx context.Context,
	ticketID string,
	teamID *string,
	actorID string,
) (*models.Ticket, error) {
	m.assignCalled = true
	m.gotAssignTicketID = ticketID
	m.gotAssignTeamID = teamID
	return m.assignResult, m.assignError
}

func (m *mockTicketRepository) SoftDelete(
	ctx context.Context,
	id string,
	actorID string,
) error {
	m.softDeleteCalled = true
	m.gotSoftDeleteID = id
	return m.softDeleteError
}

// TestListTicketsDefaultPagination checks the default page and page_size.
func TestListTicketsDefaultPagination(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 3, listResult: []models.Ticket{{}, {}, {}}}
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
	if mockRepo.gotPageSize != 20 {
		t.Errorf("expected page_size 20, got %d", mockRepo.gotPageSize)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"pagination"`) {
		t.Error("expected pagination in response")
	}
	if !strings.Contains(body, `"total":3`) {
		t.Error("expected total in response")
	}
	if !strings.Contains(body, `"total_pages":1`) {
		t.Error("expected total_pages in response")
	}
}

// TestListTicketsFirstPage verifies first-page results and metadata.
func TestListTicketsFirstPage(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 25, listResult: make([]models.Ticket, 20)}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets?page=1&page_size=20", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mockRepo.gotPage != 1 {
		t.Errorf("expected page 1, got %d", mockRepo.gotPage)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"total":25`) {
		t.Error("expected total 25")
	}
	if !strings.Contains(body, `"total_pages":2`) {
		t.Error("expected total_pages 2")
	}
}

// TestListTicketsSecondPage verifies second-page offset is forwarded.
func TestListTicketsSecondPage(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 25, listResult: make([]models.Ticket, 5)}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets?page=2&page_size=20", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mockRepo.gotPage != 2 {
		t.Errorf("expected page 2, got %d", mockRepo.gotPage)
	}
}

// TestListTicketsEmptyResults returns empty array and zero total_pages.
func TestListTicketsEmptyResults(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 0, listResult: nil}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"data":[]`) {
		t.Error("expected empty data array, not null")
	}
	if !strings.Contains(body, `"total":0`) {
		t.Error("expected total 0")
	}
	if !strings.Contains(body, `"total_pages":0`) {
		t.Error("expected total_pages 0")
	}
}

// TestListTicketsTotalCount verifies total count is returned.
func TestListTicketsTotalCount(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 125, listResult: make([]models.Ticket, 20)}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil))

	if !strings.Contains(rec.Body.String(), `"total":125`) {
		t.Error("expected total 125")
	}
}

// TestListTicketsTotalPages verifies total_pages calculation.
func TestListTicketsTotalPages(t *testing.T) {
	cases := []struct {
		total    int
		pageSize int
		want     int
	}{
		{125, 20, 7},
		{20, 20, 1},
		{21, 20, 2},
		{0, 20, 0},
	}
	for _, tc := range cases {
		mockRepo := &mockTicketRepository{listTotal: tc.total}
		handler := NewTicketHandler(mockRepo)
		rec := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v1/tickets?page_size=%d", tc.pageSize)
		handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, url, nil))
		expected := fmt.Sprintf(`"total_pages":%d`, tc.want)
		if !strings.Contains(rec.Body.String(), expected) {
			t.Errorf("total=%d page_size=%d: expected %s in %s", tc.total, tc.pageSize, expected, rec.Body.String())
		}
	}
}

// TestListTicketsMaxPageSize verifies page_size=100 is accepted.
func TestListTicketsMaxPageSize(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets?page_size=100", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if mockRepo.gotPageSize != 100 {
		t.Errorf("expected page_size 100, got %d", mockRepo.gotPageSize)
	}
}

// TestListTicketsDatabaseError returns 500 on repository error.
func TestListTicketsDatabaseError(t *testing.T) {
	mockRepo := &mockTicketRepository{listError: errors.New("db failure")}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}

// TestListTicketsCustomPagination checks custom page_size parameter.
func TestListTicketsCustomPagination(t *testing.T) {
	mockRepo := &mockTicketRepository{listTotal: 10, listResult: make([]models.Ticket, 5)}
	handler := NewTicketHandler(mockRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets?page=2&page_size=5",
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
	if mockRepo.gotPageSize != 5 {
		t.Errorf("expected page_size 5, got %d", mockRepo.gotPageSize)
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

// TestListTicketsFilteredResults verifies filters are forwarded and count matches.
func TestListTicketsFilteredResults(t *testing.T) {
	mockRepo := &mockTicketRepository{
		listTotal:  3,
		listResult: make([]models.Ticket, 3),
	}
	handler := NewTicketHandler(mockRepo)

	rec := httptest.NewRecorder()
	handler.ListTickets(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tickets?status=open&priority=high", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mockRepo.gotStatus != "open" {
		t.Errorf("expected status open, got %s", mockRepo.gotStatus)
	}
	if mockRepo.gotPriority != "high" {
		t.Errorf("expected priority high, got %s", mockRepo.gotPriority)
	}
	if !strings.Contains(rec.Body.String(), `"total":3`) {
		t.Error("expected total 3 for filtered results")
	}
}

// TestListTicketsInvalidPagination rejects invalid page and page_size values.
func TestListTicketsInvalidPagination(t *testing.T) {
	testCases := []string{
		"/api/v1/tickets?page=0",
		"/api/v1/tickets?page=abc",
		"/api/v1/tickets?page_size=0",
		"/api/v1/tickets?page_size=101",
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
	request = withTestClaims(request)

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
	request = withTestClaims(request)

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
	request = withTestClaims(request)

	recorder := httptest.NewRecorder()
	handler.UpdateTicket(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", recorder.Code)
	}
}

// --- AI classification tests ---

type mockAIClassifier struct {
	response *services.ClassificationResponse
	err      error
	// block signals the mock to block until the context is cancelled.
	block bool
}

func (m *mockAIClassifier) ClassifyTicket(
	ctx context.Context,
	subject string,
	description string,
) (*services.ClassificationResponse, error) {
	if m.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return m.response, m.err
}

type mockPredictionRepository struct {
	err          error
	createCalled bool
	got          *models.AIPrediction
}

func (m *mockPredictionRepository) Create(
	ctx context.Context,
	prediction *models.AIPrediction,
) error {
	m.createCalled = true
	m.got = prediction
	return m.err
}

// mockTicketRepositoryWithResult wraps mockTicketRepository and returns a
// configured ticket from Create so AI classification tests have a real ticket.
type mockTicketRepositoryWithResult struct {
	mockTicketRepository
	createResult *models.Ticket
	createErr    error
	gotCreatorID string
}

func (m *mockTicketRepositoryWithResult) Create(
	ctx context.Context,
	req tickets.CreateTicketRequest,
	creatorID string,
) (*models.Ticket, error) {
	m.gotCreatorID = creatorID
	return m.createResult, m.createErr
}

func newTicketForAI() *models.Ticket {
	return &models.Ticket{
		ID:          "a2f66976-199f-4ec9-8774-21f937befe13",
		Subject:     "Payment failed",
		Description: "Transaction declined",
	}
}

func TestCreateTicketAIClassificationSuccess(t *testing.T) {
	repo := &mockTicketRepositoryWithResult{createResult: newTicketForAI()}
	aiClient := &mockAIClassifier{
		response: &services.ClassificationResponse{
			Category:   "billing",
			Priority:   "high",
			Sentiment:  "negative",
			Confidence: 0.95,
		},
	}
	predRepo := &mockPredictionRepository{}

	handler := NewTicketHandlerWithAI(repo, aiClient, predRepo)

	body := `{"subject":"Payment failed","description":"Transaction declined","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	request = withTestClaims(request)
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !predRepo.createCalled {
		t.Error("expected prediction to be saved")
	}
	if predRepo.got.TicketID != newTicketForAI().ID {
		t.Errorf("expected ticket ID %s, got %s", newTicketForAI().ID, predRepo.got.TicketID)
	}
}

func TestCreateTicketAIClassificationFailureDoesNotBlockCreation(t *testing.T) {
	repo := &mockTicketRepositoryWithResult{createResult: newTicketForAI()}
	aiClient := &mockAIClassifier{err: errors.New("AI service unavailable")}
	predRepo := &mockPredictionRepository{}

	handler := NewTicketHandlerWithAI(repo, aiClient, predRepo)

	body := `{"subject":"Payment failed","description":"Transaction declined","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	request = withTestClaims(request)
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201 even when AI fails, got %d", recorder.Code)
	}
	if predRepo.createCalled {
		t.Error("prediction must not be saved when classification failed")
	}
}

func TestCreateTicketAITimeoutDoesNotBlockCreation(t *testing.T) {
	repo := &mockTicketRepositoryWithResult{createResult: newTicketForAI()}
	aiClient := &mockAIClassifier{block: true}
	predRepo := &mockPredictionRepository{}

	handler := NewTicketHandlerWithAI(repo, aiClient, predRepo)

	body := `{"subject":"Payment failed","description":"Transaction declined","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	request = withTestClaims(request)
	// Pre-cancel the context to simulate an immediate timeout without sleeping.
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)

	recorder := httptest.NewRecorder()
	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201 even on AI timeout, got %d", recorder.Code)
	}
}

func TestCreateTicketAIPredictionPersistenceErrorIsHandledSafely(t *testing.T) {
	repo := &mockTicketRepositoryWithResult{createResult: newTicketForAI()}
	aiClient := &mockAIClassifier{
		response: &services.ClassificationResponse{
			Category:   "billing",
			Priority:   "medium",
			Sentiment:  "neutral",
			Confidence: 0.80,
		},
	}
	predRepo := &mockPredictionRepository{err: errors.New("db write failed")}

	handler := NewTicketHandlerWithAI(repo, aiClient, predRepo)

	body := `{"subject":"Payment failed","description":"Transaction declined","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	request = withTestClaims(request)
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201 even when prediction save fails, got %d", recorder.Code)
	}
}

// --- Creator tracking tests ---

func TestCreateTicketRequiresAuthentication(t *testing.T) {
	repo := &mockTicketRepositoryWithResult{createResult: newTicketForAI()}
	handler := NewTicketHandler(repo)

	body := `{"subject":"Test","description":"Desc","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", recorder.Code)
	}
}

func TestCreateTicketStoresCreatorID(t *testing.T) {
	creatorID := "user-uuid-1234"
	ticket := newTicketForAI()
	ticket.CreatedBy = &creatorID

	repo := &mockTicketRepositoryWithResult{createResult: ticket}
	handler := NewTicketHandler(repo)

	body := `{"subject":"Payment failed","description":"Transaction declined","customer_name":"Jane","customer_email":"jane@example.com","channel":"web"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	claims := &auth.Claims{UserID: creatorID, Email: "agent@example.com", Role: "agent"}
	request = request.WithContext(middleware.SetUserClaims(request.Context(), claims))
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if repo.gotCreatorID != creatorID {
		t.Errorf("expected creatorID %s, got %s", creatorID, repo.gotCreatorID)
	}
}

func TestCreateTicketClientCannotOverrideCreatorID(t *testing.T) {
	creatorID := "real-user-uuid"
	ticket := newTicketForAI()
	ticket.CreatedBy = &creatorID

	repo := &mockTicketRepositoryWithResult{createResult: ticket}
	handler := NewTicketHandler(repo)

	// Body includes a created_by field — DisallowUnknownFields must reject it.
	body := `{"subject":"Test","description":"Desc","customer_name":"Jane","customer_email":"jane@example.com","channel":"web","created_by":"attacker-uuid"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", strings.NewReader(body))
	claims := &auth.Claims{UserID: creatorID, Email: "agent@example.com", Role: "agent"}
	request = request.WithContext(middleware.SetUserClaims(request.Context(), claims))
	recorder := httptest.NewRecorder()

	handler.CreateTicket(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected 400 when created_by is in body, got %d", recorder.Code)
	}
}

// --- GetTicketHistory tests ---

const historyTicketID = "a2f66976-199f-4ec9-8774-21f937befe13"

func newHistoryRequest(ticketID, query string) *http.Request {
	url := "/api/v1/tickets/" + ticketID + "/history"
	if query != "" {
		url += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.SetPathValue("id", ticketID)
	return withTestClaims(req)
}

func TestGetTicketHistorySuccess(t *testing.T) {
	actorID := "actor-uuid"
	old := []byte(`{"status":"open"}`)
	nw := []byte(`{"status":"in_progress"}`)

	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyCount:  1,
		historyResult: []models.TicketHistory{
			{
				ID:        "hist-1",
				TicketID:  historyTicketID,
				ActorID:   &actorID,
				Action:    "ticket.updated",
				OldValue:  old,
				NewValue:  nw,
				CreatedAt: time.Now(),
			},
		},
	}

	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()
	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"data"`) {
		t.Error("expected data key in response")
	}
	if !strings.Contains(body, `"pagination"`) {
		t.Error("expected pagination key in response")
	}
	if !strings.Contains(body, "ticket.updated") {
		t.Error("expected history action in response")
	}
}

func TestGetTicketHistoryUnauthenticated(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+historyTicketID+"/history", nil)
	req.SetPathValue("id", historyTicketID)
	// No claims injected — handler checks GetByID first, but the route in
	// production is behind AuthMiddleware. Here we verify the handler itself
	// does not panic and the middleware would block it. We simulate by calling
	// the handler without claims; it proceeds to GetByID (no auth check in
	// handler body), so we just confirm the route is auth-gated via the
	// middleware test below.
	mockRepo.getByIDResult = &models.Ticket{ID: historyTicketID}
	rec := httptest.NewRecorder()
	handler.GetTicketHistory(rec, req)
	// Without claims the handler still works (auth is middleware's job).
	// Verify the middleware blocks unauthenticated requests.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+historyTicketID+"/history", nil)
	req2.SetPathValue("id", historyTicketID)
	rec2 := httptest.NewRecorder()
	jwtService, _ := auth.NewJWTService("this-is-a-test-secret-with-more-than-32-characters", 15)
	middleware.AuthMiddleware(jwtService)(http.HandlerFunc(handler.GetTicketHistory)).ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec2.Code)
	}
}

func TestGetTicketHistoryInvalidUUID(t *testing.T) {
	mockRepo := &mockTicketRepository{}
	handler := NewTicketHandler(mockRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/not-a-uuid/history", nil)
	req.SetPathValue("id", "not-a-uuid")
	req = withTestClaims(req)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetTicketHistoryTicketNotFound(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDError: pgx.ErrNoRows,
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetTicketHistoryEmpty(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyCount:  0,
		historyResult: nil,
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Error("expected empty data array, not null")
	}
}

func TestGetTicketHistoryMultipleRecords(t *testing.T) {
	actorID := "actor-uuid"
	records := []models.TicketHistory{
		{ID: "h1", TicketID: historyTicketID, ActorID: &actorID, Action: "ticket.updated", CreatedAt: time.Now()},
		{ID: "h2", TicketID: historyTicketID, ActorID: &actorID, Action: "ticket.updated", CreatedAt: time.Now().Add(-time.Minute)},
	}
	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyCount:  2,
		historyResult: records,
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "h1") || !strings.Contains(rec.Body.String(), "h2") {
		t.Error("expected both history records in response")
	}
}

func TestGetTicketHistoryPagination(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyCount:  50,
		historyResult: []models.TicketHistory{},
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, "page=3&page_size=10"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if mockRepo.gotHistoryPage != 3 {
		t.Errorf("expected page 3, got %d", mockRepo.gotHistoryPage)
	}
	if mockRepo.gotHistoryPageSize != 10 {
		t.Errorf("expected page_size 10, got %d", mockRepo.gotHistoryPageSize)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"total":50`) {
		t.Error("expected total 50 in pagination")
	}
	if !strings.Contains(body, `"total_pages":5`) {
		t.Error("expected total_pages 5 in pagination")
	}
}

func TestGetTicketHistoryInvalidPagination(t *testing.T) {
	cases := []string{"page=0", "page=abc", "page_size=0", "page_size=101"}
	for _, q := range cases {
		t.Run(q, func(t *testing.T) {
			mockRepo := &mockTicketRepository{
				getByIDResult: &models.Ticket{ID: historyTicketID},
			}
			handler := NewTicketHandler(mockRepo)
			rec := httptest.NewRecorder()
			handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, q))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", rec.Code)
			}
		})
	}
}

func TestGetTicketHistoryRepositoryError(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyError:  errors.New("db failure"),
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}

func TestGetTicketHistoryCountError(t *testing.T) {
	mockRepo := &mockTicketRepository{
		getByIDResult:     &models.Ticket{ID: historyTicketID},
		historyCountError: errors.New("count query failed"),
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestGetTicketHistoryNewestFirst(t *testing.T) {
	now := time.Now()
	older := now.Add(-time.Hour)
	actorID := "actor-uuid"

	// Repository returns newest first (as the SQL ORDER BY guarantees).
	// The handler must preserve that order.
	records := []models.TicketHistory{
		{ID: "newer", TicketID: historyTicketID, ActorID: &actorID, Action: "ticket.updated", CreatedAt: now},
		{ID: "older", TicketID: historyTicketID, ActorID: &actorID, Action: "ticket.updated", CreatedAt: older},
	}
	mockRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: historyTicketID},
		historyCount:  2,
		historyResult: records,
	}
	handler := NewTicketHandler(mockRepo)
	rec := httptest.NewRecorder()

	handler.GetTicketHistory(rec, newHistoryRequest(historyTicketID, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	newerPos := strings.Index(body, "newer")
	olderPos := strings.Index(body, "older")
	if newerPos == -1 || olderPos == -1 {
		t.Fatal("expected both records in response")
	}
	if newerPos > olderPos {
		t.Error("expected newest record to appear before older record")
	}
}

// --- AssignTicket handler tests ---

const assignTicketID = "a2f66976-199f-4ec9-8774-21f937befe13"
const assignTeamID = "c4d88198-3bb1-6eeb-aa96-43f159dffe35"

func newAssignRequest(ticketID, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPatch,
		"/api/v1/tickets/"+ticketID+"/assignment",
		strings.NewReader(body))
	req.SetPathValue("id", ticketID)
	return req
}

func TestAssignTicketSuccess(t *testing.T) {
	teamID := assignTeamID
	ticket := &models.Ticket{ID: assignTicketID, AssignedTeamID: &teamID}
	repo := &mockTicketRepository{assignResult: ticket}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":"`+teamID+`"}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !repo.assignCalled {
		t.Fatal("expected AssignTeam to be called")
	}
	if repo.gotAssignTicketID != assignTicketID {
		t.Errorf("expected ticket ID %s, got %s", assignTicketID, repo.gotAssignTicketID)
	}
	if repo.gotAssignTeamID == nil || *repo.gotAssignTeamID != teamID {
		t.Errorf("expected team ID %s", teamID)
	}
}

func TestAssignTicketUnassign(t *testing.T) {
	ticket := &models.Ticket{ID: assignTicketID}
	repo := &mockTicketRepository{assignResult: ticket}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":null}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.gotAssignTeamID != nil {
		t.Error("expected nil team ID for unassign")
	}
}

func TestAssignTicketUnauthenticated(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := newAssignRequest(assignTicketID, `{"team_id":null}`)
	// no claims
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAssignTicketNonAdmin(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := newAssignRequest(assignTicketID, `{"team_id":null}`)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAssignTicketInvalidTicketUUID(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tickets/bad-id/assignment",
		strings.NewReader(`{"team_id":null}`))
	req.SetPathValue("id", "bad-id")
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAssignTicketInvalidTeamUUID(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := newAssignRequest(assignTicketID, `{"team_id":"not-a-uuid"}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAssignTicketNotFound(t *testing.T) {
	repo := &mockTicketRepository{assignError: repository.ErrTicketNotFound}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":null}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAssignTicketTeamNotFound(t *testing.T) {
	// Use a valid UUID that the handler accepts but the mock repo returns not-found for.
	teamID := "d5e99209-4cc2-7ffc-bb07-54f26ae0ff46"
	repo := &mockTicketRepository{assignError: repository.ErrTeamNotFound}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":"`+teamID+`"}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAssignTicketRepositoryFailure(t *testing.T) {
	repo := &mockTicketRepository{assignError: errors.New("db failure")}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":null}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}

func TestAssignTicketDuplicateAssignment(t *testing.T) {
	// Same team already assigned — repository returns the ticket unchanged (no error).
	teamID := assignTeamID
	ticket := &models.Ticket{ID: assignTicketID, AssignedTeamID: &teamID}
	repo := &mockTicketRepository{assignResult: ticket}
	h := NewTicketHandler(repo)

	req := newAssignRequest(assignTicketID, `{"team_id":"`+teamID+`"}`)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AssignTicket(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for duplicate assignment, got %d", rec.Code)
	}
}

// --- DeleteTicket handler tests ---

const deleteTicketID = "a2f66976-199f-4ec9-8774-21f937befe13"

func newDeleteRequest(ticketID string) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tickets/"+ticketID, nil)
	req.SetPathValue("id", ticketID)
	return req
}

func TestDeleteTicketSuccess(t *testing.T) {
	repo := &mockTicketRepository{}
	h := NewTicketHandler(repo)

	req := newDeleteRequest(deleteTicketID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if !repo.softDeleteCalled {
		t.Error("expected SoftDelete to be called")
	}
	if repo.gotSoftDeleteID != deleteTicketID {
		t.Errorf("expected ticket ID %s, got %s", deleteTicketID, repo.gotSoftDeleteID)
	}
}

func TestDeleteTicketUnauthenticated(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := newDeleteRequest(deleteTicketID)
	// no claims
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteTicketNonAdmin(t *testing.T) {
	h := NewTicketHandler(&mockTicketRepository{})

	req := newDeleteRequest(deleteTicketID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestDeleteTicketInvalidUUID(t *testing.T) {
	repo := &mockTicketRepository{}
	h := NewTicketHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tickets/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	if repo.softDeleteCalled {
		t.Error("SoftDelete must not be called for invalid UUID")
	}
}

func TestDeleteTicketNotFound(t *testing.T) {
	repo := &mockTicketRepository{softDeleteError: repository.ErrTicketNotFound}
	h := NewTicketHandler(repo)

	req := newDeleteRequest(deleteTicketID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteTicketAlreadyDeleted(t *testing.T) {
	// SoftDelete returns ErrTicketNotFound for already-deleted tickets.
	repo := &mockTicketRepository{softDeleteError: repository.ErrTicketNotFound}
	h := NewTicketHandler(repo)

	req := newDeleteRequest(deleteTicketID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for already-deleted ticket, got %d", rec.Code)
	}
}

func TestDeleteTicketRepositoryFailure(t *testing.T) {
	repo := &mockTicketRepository{softDeleteError: errors.New("db failure")}
	h := NewTicketHandler(repo)

	req := newDeleteRequest(deleteTicketID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.DeleteTicket(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}
