package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AIPredictionReader defines the operation for retrieving AI predictions.
type AIPredictionReader interface {
	GetLatestByTicketID(
		ctx context.Context,
		ticketID string,
	) (*models.AIPrediction, error)
}

// AIPredictionHandler handles AI prediction requests.
type AIPredictionHandler struct {
	ticketRepo     TicketRepository
	predictionRepo AIPredictionReader
}

// NewAIPredictionHandler creates an AI prediction handler.
func NewAIPredictionHandler(
	ticketRepo TicketRepository,
	predictionRepo AIPredictionReader,
) *AIPredictionHandler {
	return &AIPredictionHandler{
		ticketRepo:     ticketRepo,
		predictionRepo: predictionRepo,
	}
}

// jsonError writes a JSON error response with the correct Content-Type.
func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// GetLatestPrediction retrieves the latest AI prediction for a ticket.
func (h *AIPredictionHandler) GetLatestPrediction(
	w http.ResponseWriter,
	r *http.Request,
) {
	ticketID := strings.TrimSpace(r.PathValue("id"))

	if _, err := uuid.Parse(ticketID); err != nil {
		jsonError(w, "invalid ticket ID", http.StatusBadRequest)
		return
	}

	// Check whether the ticket exists.
	_, err := h.ticketRepo.GetByID(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			jsonError(w, "ticket not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to retrieve ticket", http.StatusInternalServerError)
		return
	}

	// Retrieve the latest AI prediction.
	prediction, err := h.predictionRepo.GetLatestByTicketID(
		r.Context(),
		ticketID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			jsonError(w, "AI prediction not found", http.StatusNotFound)
			return
		}
		jsonError(w, "failed to retrieve AI prediction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(prediction); err != nil {
		return
	}
}
