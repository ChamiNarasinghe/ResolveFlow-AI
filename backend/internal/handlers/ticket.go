package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/services"
	"github.com/chami2002/resolveflow-api/internal/tickets"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TicketRepository defines the database operations required by TicketHandler.
type TicketRepository interface {
	Create(
		ctx context.Context,
		req tickets.CreateTicketRequest,
		creatorID string,
	) (*models.Ticket, error)

	List(
		ctx context.Context,
		page int,
		limit int,
		status string,
		priority string,
	) ([]models.Ticket, error)

	GetByID(
		ctx context.Context,
		id string,
	) (*models.Ticket, error)

	Update(
		ctx context.Context,
		id string,
		actorID string,
		req tickets.UpdateTicketRequest,
	) (*models.Ticket, error)
}

// AIClassifier defines the AI classification operation.
type AIClassifier interface {
	ClassifyTicket(
		ctx context.Context,
		subject string,
		description string,
	) (*services.ClassificationResponse, error)
}

// AIPredictionRepository defines AI prediction database operations.
type AIPredictionRepository interface {
	Create(
		ctx context.Context,
		prediction *models.AIPrediction,
	) error
}

type TicketHandler struct {
	repository     TicketRepository
	aiClient       AIClassifier
	predictionRepo AIPredictionRepository
}

// NewTicketHandler creates a ticket handler without AI dependencies.
func NewTicketHandler(repo TicketRepository) *TicketHandler {
	return &TicketHandler{
		repository: repo,
	}
}

// NewTicketHandlerWithAI creates a ticket handler with AI dependencies.
func NewTicketHandlerWithAI(
	repo TicketRepository,
	aiClient AIClassifier,
	predictionRepo AIPredictionRepository,
) *TicketHandler {
	return &TicketHandler{
		repository:     repo,
		aiClient:       aiClient,
		predictionRepo: predictionRepo,
	}
}

func (h *TicketHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	var request tickets.CreateTicketRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.repository.Create(r.Context(), request, claims.UserID)
	if err != nil {
		http.Error(w, "Failed to create ticket", http.StatusInternalServerError)
		return
	}

	// Run AI classification when the AI dependencies are configured.
	if h.aiClient != nil && h.predictionRepo != nil {
		aiCtx, cancel := context.WithTimeout(
			r.Context(),
			5*time.Second,
		)
		defer cancel()

		classification, aiErr := h.aiClient.ClassifyTicket(
			aiCtx,
			result.Subject,
			result.Description,
		)

		rid := middleware.GetRequestID(r.Context())
		if aiErr != nil {
			slog.Warn("AI classification failed",
				slog.String("request_id", rid),
				slog.String("ticket_id", result.ID),
				slog.String("error", aiErr.Error()),
			)
		} else {
			prediction := &models.AIPrediction{
				TicketID:          result.ID,
				ModelVersion:      "rules-v1",
				Category:          classification.Category,
				PredictedPriority: classification.Priority,
				Sentiment:         classification.Sentiment,
				Confidence:        classification.Confidence,
			}

			if err := h.predictionRepo.Create(
				r.Context(),
				prediction,
			); err != nil {
				slog.Error("Failed to save AI prediction",
					slog.String("request_id", rid),
					slog.String("ticket_id", result.ID),
					slog.String("error", err.Error()),
				)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		return
	}
}

func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	// Read pagination parameters.
	page := 1
	limit := 10

	query := r.URL.Query()

	if pageValue := query.Get("page"); pageValue != "" {
		parsedPage, err := strconv.Atoi(pageValue)
		if err != nil || parsedPage < 1 {
			http.Error(w, "Page must be a positive integer", http.StatusBadRequest)
			return
		}
		page = parsedPage
	}

	if limitValue := query.Get("limit"); limitValue != "" {
		parsedLimit, err := strconv.Atoi(limitValue)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			http.Error(w, "Limit must be between 1 and 100", http.StatusBadRequest)
			return
		}
		limit = parsedLimit
	}

	status := strings.ToLower(strings.TrimSpace(query.Get("status")))
	priority := strings.ToLower(strings.TrimSpace(query.Get("priority")))

	validStatuses := map[string]bool{
		"open":                 true,
		"in_progress":          true,
		"waiting_for_customer": true,
		"resolved":             true,
		"closed":               true,
	}

	validPriorities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"urgent": true,
	}

	if status != "" && !validStatuses[status] {
		http.Error(w, "Invalid ticket status", http.StatusBadRequest)
		return
	}

	if priority != "" && !validPriorities[priority] {
		http.Error(w, "Invalid ticket priority", http.StatusBadRequest)
		return
	}

	result, err := h.repository.List(
		r.Context(),
		page,
		limit,
		status,
		priority,
	)
	if err != nil {
		http.Error(w, "Failed to retrieve tickets", http.StatusInternalServerError)
		return
	}

	// Return tickets and pagination information.
	response := map[string]interface{}{
		"data": result,
		"pagination": map[string]int{
			"page":  page,
			"limit": limit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *TicketHandler) GetTicket(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))

	// Validate the ticket UUID.
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid ticket ID"}`, http.StatusBadRequest)
		return
	}

	ticket, err := h.repository.GetByID(r.Context(), id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"ticket not found"}`, http.StatusNotFound)
			return
		}

		http.Error(w, `{"error":"failed to retrieve ticket"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(ticket); err != nil {
		return
	}
}

func (h *TicketHandler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid ticket ID"}`, http.StatusBadRequest)
		return
	}

	var request tickets.UpdateTicketRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
		return
	}

	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	result, err := h.repository.Update(
		r.Context(),
		id,
		claims.UserID,
		request,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"ticket not found"}`, http.StatusNotFound)
			return
		}

		http.Error(w, `{"error":"failed to update ticket"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		return
	}
}
