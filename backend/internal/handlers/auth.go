package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
)

type AuthService interface {
	Register(
		ctx context.Context,
		request auth.RegisterRequest,
	) (*auth.AuthResponse, error)

	Login(
		ctx context.Context,
		request auth.LoginRequest,
	) (*auth.AuthResponse, error)

	GetProfile(
		ctx context.Context,
		userID string,
	) (*models.User, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var request auth.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.Register(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request auth.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.service.Login(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		case errors.Is(err, auth.ErrInactiveUser):
			http.Error(w, "User account is inactive", http.StatusForbidden)
		default:
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// Me returns the authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	user, err := h.service.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, auth.ErrInactiveUser) {
			http.Error(w, "User account is inactive", http.StatusForbidden)
			return
		}

		if errors.Is(err, auth.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Failed to retrieve user profile", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user": user,
	})
}
