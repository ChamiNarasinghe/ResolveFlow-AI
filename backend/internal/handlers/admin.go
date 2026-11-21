package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
)

// AdminUserService is the subset of auth.AuthService used by AdminHandler.
type AdminUserService interface {
	ProvisionUser(ctx context.Context, request auth.ProvisionRequest) (*models.User, error)
}

// AdminHandler handles administrator-only user management endpoints.
type AdminHandler struct {
	service AdminUserService
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(service AdminUserService) *AdminHandler {
	return &AdminHandler{service: service}
}

// CreateUser POST /api/v1/admin/users — admin only.
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	if claims.Role != "admin" {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return
	}

	var req auth.ProvisionRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.service.ProvisionUser(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyRegistered):
			http.Error(w, `{"error":"email address is already registered"}`, http.StatusConflict)
		case auth.IsValidationError(err):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		default:
			slog.Error("admin user creation failed",
				slog.String("request_id", middleware.GetRequestID(r.Context())),
				slog.String("error", err.Error()),
			)
			http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}
