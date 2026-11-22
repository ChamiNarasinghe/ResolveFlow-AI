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
	"github.com/chami2002/resolveflow-api/internal/repository"
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
		pageSize int,
		status string,
		priority string,
	) ([]models.Ticket, int, error)

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

	GetHistoryByTicketID(
		ctx context.Context,
		ticketID string,
		page int,
		pageSize int,
	) ([]models.TicketHistory, error)

	CountHistoryByTicketID(
		ctx context.Context,
		ticketID string,
	) (int, error)

	AssignTeam(
		ctx context.Context,
		ticketID string,
		teamID *string,
		actorID string,
	) (*models.Ticket, error)

	SoftDelete(
		ctx context.Context,
		id string,
		actorID string,
	) error
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
				ModelVersion:      "tfidf-logreg-v1",
				Category:          &classification.Category,
				PredictedPriority: &classification.Priority,
				Sentiment:         &classification.Sentiment,
				Confidence:        &classification.Confidence,
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

const maxTicketPageSize = 100

func (h *TicketHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 20

	q := r.URL.Query()

	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			http.Error(w, "page must be a positive integer", http.StatusBadRequest)
			return
		}
		page = p
	}

	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > maxTicketPageSize {
			http.Error(w, "page_size must be between 1 and 100", http.StatusBadRequest)
			return
		}
		pageSize = ps
	}

	status := strings.ToLower(strings.TrimSpace(q.Get("status")))
	priority := strings.ToLower(strings.TrimSpace(q.Get("priority")))

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

	tickets, total, err := h.repository.List(r.Context(), page, pageSize, status, priority)
	if err != nil {
		http.Error(w, "Failed to retrieve tickets", http.StatusInternalServerError)
		return
	}

	if tickets == nil {
		tickets = []models.Ticket{}
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	response := map[string]any{
		"data": tickets,
		"pagination": map[string]int{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response) //nolint:errcheck
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

const maxHistoryPageSize = 100

func (h *TicketHandler) GetTicketHistory(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid ticket ID"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.repository.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"error":"ticket not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to retrieve ticket"}`, http.StatusInternalServerError)
		return
	}

	page, pageSize := 1, 20
	q := r.URL.Query()

	if v := q.Get("page"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 {
			http.Error(w, `{"error":"page must be a positive integer"}`, http.StatusBadRequest)
			return
		}
		page = p
	}

	if v := q.Get("page_size"); v != "" {
		ps, err := strconv.Atoi(v)
		if err != nil || ps < 1 || ps > maxHistoryPageSize {
			http.Error(w, `{"error":"page_size must be between 1 and 100"}`, http.StatusBadRequest)
			return
		}
		pageSize = ps
	}

	total, err := h.repository.CountHistoryByTicketID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve ticket history"}`, http.StatusInternalServerError)
		return
	}

	records, err := h.repository.GetHistoryByTicketID(r.Context(), id, page, pageSize)
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve ticket history"}`, http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []models.TicketHistory{}
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	response := map[string]any{
		"data": records,
		"pagination": map[string]int{
			"page":        page,
			"page_size":   pageSize,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response) //nolint:errcheck
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

// DeleteTicket handles DELETE /api/v1/tickets/{id}.
// Admin only. Soft-deletes the ticket; returns 204 on success.
func (h *TicketHandler) DeleteTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid ticket ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.repository.SoftDelete(r.Context(), id, claims.UserID); err != nil {
		if errors.Is(err, repository.ErrTicketNotFound) {
			http.Error(w, `{"error":"ticket not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to delete ticket"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// assignTicketBody is the request body for PATCH /api/v1/tickets/{id}/assignment.
// TeamID is a pointer so that JSON null explicitly unassigns the ticket.
type assignTicketBody struct {
	TeamID *string `json:"team_id"`
}

// AssignTicket handles PATCH /api/v1/tickets/{id}/assignment.
// Admin only. Assigns or unassigns a ticket to/from a team.
func (h *TicketHandler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid ticket ID"}`, http.StatusBadRequest)
		return
	}

	var body assignTicketBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
		return
	}

	// Validate team UUID when provided.
	if body.TeamID != nil {
		if _, err := uuid.Parse(*body.TeamID); err != nil {
			http.Error(w, `{"error":"invalid team_id"}`, http.StatusBadRequest)
			return
		}
	}

	ticket, err := h.repository.AssignTeam(r.Context(), id, body.TeamID, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTicketNotFound):
			http.Error(w, `{"error":"ticket not found"}`, http.StatusNotFound)
		case errors.Is(err, repository.ErrTeamNotFound):
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
		default:
			http.Error(w, `{"error":"failed to assign ticket"}`, http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}
