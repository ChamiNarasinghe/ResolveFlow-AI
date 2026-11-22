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

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/jackc/pgx/v5"
)

// mockAIPredictionReader simulates AI prediction database operations.
type mockAIPredictionReader struct {
	result *models.AIPrediction
	err    error
	called bool
}

func (m *mockAIPredictionReader) GetLatestByTicketID(
	ctx context.Context,
	ticketID string,
) (*models.AIPrediction, error) {
	m.called = true
	return m.result, m.err
}

// TestGetLatestPredictionSuccess tests successful prediction retrieval.
func TestGetLatestPredictionSuccess(t *testing.T) {
	ticketID := "8d8110dd-6840-4fda-833a-1df922221cf2"

	category := "billing"
	priority := "urgent"
	sentiment := "negative"
	confidence := 0.8

	expectedPrediction := &models.AIPrediction{
		ID:                "0a554678-8e7d-413b-a781-e1276aa58833",
		TicketID:          ticketID,
		ModelVersion:      "tfidf-logreg-v1",
		Category:          &category,
		PredictedPriority: &priority,
		Sentiment:         &sentiment,
		Confidence:        &confidence,
		CreatedAt:         time.Now(),
	}

	ticketRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: ticketID},
	}

	predictionRepo := &mockAIPredictionReader{
		result: expectedPrediction,
	}

	handler := NewAIPredictionHandler(ticketRepo, predictionRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/ai-prediction",
		nil,
	)
	request.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()
	handler.GetLatestPrediction(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if !strings.Contains(recorder.Body.String(), `"predicted_priority":"urgent"`) {
		t.Error("expected urgent prediction in response")
	}

	if !strings.Contains(recorder.Body.String(), `"category":"billing"`) {
		t.Error("expected billing category in response")
	}

	if !predictionRepo.called {
		t.Error("expected prediction repository to be called")
	}
}

// TestGetLatestPredictionInvalidUUID tests invalid ticket IDs.
func TestGetLatestPredictionInvalidUUID(t *testing.T) {
	ticketRepo := &mockTicketRepository{}
	predictionRepo := &mockAIPredictionReader{}

	handler := NewAIPredictionHandler(ticketRepo, predictionRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/invalid-id/ai-prediction",
		nil,
	)
	request.SetPathValue("id", "invalid-id")

	recorder := httptest.NewRecorder()
	handler.GetLatestPrediction(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}

	if predictionRepo.called {
		t.Error("prediction repository should not be called for invalid UUID")
	}
}

// TestGetLatestPredictionTicketNotFound tests missing tickets.
func TestGetLatestPredictionTicketNotFound(t *testing.T) {
	ticketID := "8d8110dd-6840-4fda-833a-1df922221cf2"

	ticketRepo := &mockTicketRepository{
		getByIDError: pgx.ErrNoRows,
	}

	predictionRepo := &mockAIPredictionReader{}

	handler := NewAIPredictionHandler(ticketRepo, predictionRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/ai-prediction",
		nil,
	)
	request.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()
	handler.GetLatestPrediction(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", recorder.Code)
	}

	if predictionRepo.called {
		t.Error("prediction repository should not be called when ticket does not exist")
	}
}

// TestGetLatestPredictionNotFound tests tickets without AI predictions.
func TestGetLatestPredictionNotFound(t *testing.T) {
	ticketID := "8d8110dd-6840-4fda-833a-1df922221cf2"

	ticketRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: ticketID},
	}

	predictionRepo := &mockAIPredictionReader{
		err: pgx.ErrNoRows,
	}

	handler := NewAIPredictionHandler(ticketRepo, predictionRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/ai-prediction",
		nil,
	)
	request.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()
	handler.GetLatestPrediction(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", recorder.Code)
	}
}

// TestGetLatestPredictionDatabaseError tests unexpected database errors.
func TestGetLatestPredictionDatabaseError(t *testing.T) {
	ticketID := "8d8110dd-6840-4fda-833a-1df922221cf2"

	ticketRepo := &mockTicketRepository{
		getByIDResult: &models.Ticket{ID: ticketID},
	}

	predictionRepo := &mockAIPredictionReader{
		err: errors.New("database connection failed"),
	}

	handler := NewAIPredictionHandler(ticketRepo, predictionRepo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/tickets/"+ticketID+"/ai-prediction",
		nil,
	)
	request.SetPathValue("id", ticketID)

	recorder := httptest.NewRecorder()
	handler.GetLatestPrediction(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", recorder.Code)
	}
}

// TestGetLatestPredictionErrorResponsesAreJSON verifies every error path
// returns Content-Type: application/json with a valid {"error":"..."} body.
func TestGetLatestPredictionErrorResponsesAreJSON(t *testing.T) {
	ticketID := "8d8110dd-6840-4fda-833a-1df922221cf2"

	cases := []struct {
		name       string
		ticketRepo *mockTicketRepository
		predRepo   *mockAIPredictionReader
		pathValue  string
		wantStatus int
	}{
		{
			name:       "invalid UUID",
			ticketRepo: &mockTicketRepository{},
			predRepo:   &mockAIPredictionReader{},
			pathValue:  "not-a-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ticket not found",
			ticketRepo: &mockTicketRepository{getByIDError: pgx.ErrNoRows},
			predRepo:   &mockAIPredictionReader{},
			pathValue:  ticketID,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "prediction not found",
			ticketRepo: &mockTicketRepository{getByIDResult: &models.Ticket{ID: ticketID}},
			predRepo:   &mockAIPredictionReader{err: pgx.ErrNoRows},
			pathValue:  ticketID,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			ticketRepo: &mockTicketRepository{getByIDResult: &models.Ticket{ID: ticketID}},
			predRepo:   &mockAIPredictionReader{err: errors.New("db error")},
			pathValue:  ticketID,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewAIPredictionHandler(tc.ticketRepo, tc.predRepo)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets/"+tc.pathValue+"/ai-prediction", nil)
			req.SetPathValue("id", tc.pathValue)
			rec := httptest.NewRecorder()
			handler.GetLatestPrediction(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected %d, got %d", tc.wantStatus, rec.Code)
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "application/json") {
				t.Errorf("expected application/json Content-Type, got %q", ct)
			}
			var body map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("response body is not valid JSON: %v", err)
			}
			if body["error"] == "" {
				t.Error("expected non-empty error field in JSON body")
			}
		})
	}
}
