package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CommentRepository defines comment database operations.
type CommentRepository interface {
	Create(ctx context.Context, comment *models.Comment) error
	ListByTicket(ctx context.Context, ticketID string) ([]models.Comment, error)
}

// CommentTicketRepository checks whether a ticket exists.
type CommentTicketRepository interface {
	GetByID(ctx context.Context, id string) (*models.Ticket, error)
}

type CommentHandler struct {
	comments CommentRepository
	tickets  CommentTicketRepository
}

// NewCommentHandler creates a comment handler.
func NewCommentHandler(
	comments CommentRepository,
	tickets CommentTicketRepository,
) *CommentHandler {
	return &CommentHandler{
		comments: comments,
		tickets:  tickets,
	}
}

// CreateComment adds a comment to a ticket.
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	ticketID := strings.TrimSpace(r.PathValue("id"))

	if _, err := uuid.Parse(ticketID); err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	_, err := h.tickets.GetByID(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to retrieve ticket", http.StatusInternalServerError)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var request struct {
		Body       string `json:"body"`
		IsInternal bool   `json:"is_internal"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	request.Body = strings.TrimSpace(request.Body)

	if request.Body == "" {
		http.Error(w, "Comment body is required", http.StatusBadRequest)
		return
	}

	if len(request.Body) > 5000 {
		http.Error(w, "Comment cannot exceed 5000 characters", http.StatusBadRequest)
		return
	}

	authorType := "agent"

	comment := &models.Comment{
		TicketID:   ticketID,
		AuthorID:   &claims.UserID,
		AuthorType: &authorType,
		Body:       request.Body,
		IsInternal: request.IsInternal,
	}

	if err := h.comments.Create(r.Context(), comment); err != nil {
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(comment)
}

// ListComments retrieves comments belonging to a ticket.
func (h *CommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	ticketID := strings.TrimSpace(r.PathValue("id"))

	if _, err := uuid.Parse(ticketID); err != nil {
		http.Error(w, "Invalid ticket ID", http.StatusBadRequest)
		return
	}

	_, err := h.tickets.GetByID(r.Context(), ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Ticket not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to retrieve ticket", http.StatusInternalServerError)
		return
	}

	comments, err := h.comments.ListByTicket(r.Context(), ticketID)
	if err != nil {
		http.Error(w, "Failed to retrieve comments", http.StatusInternalServerError)
		return
	}

	if comments == nil {
		comments = []models.Comment{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(comments); err != nil {
		return
	}
}
