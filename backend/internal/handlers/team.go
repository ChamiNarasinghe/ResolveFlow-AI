package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/repository"
	"github.com/google/uuid"
)

// TeamRepository defines the database operations required by TeamHandler.
type TeamRepository interface {
	CreateTeam(ctx context.Context, name string, description *string) (*models.Team, error)
	ListTeams(ctx context.Context) ([]models.Team, error)
	GetTeamByID(ctx context.Context, id string) (*models.Team, error)
	UpdateTeam(ctx context.Context, id string, req repository.UpdateTeamRequest) (*models.Team, error)
	AddTeamMember(ctx context.Context, teamID, userID, teamRole string) (*models.TeamMember, error)
	RemoveTeamMember(ctx context.Context, teamID, userID string) error
	ListTeamMembers(ctx context.Context, teamID string) ([]models.TeamMember, error)
}

// TeamHandler handles team management endpoints.
type TeamHandler struct {
	repo TeamRepository
}

// NewTeamHandler creates a team handler.
func NewTeamHandler(repo TeamRepository) *TeamHandler {
	return &TeamHandler{repo: repo}
}

// requireAdmin returns false and writes 403 if the caller is not an admin.
func requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return false
	}
	if claims.Role != "admin" {
		http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
		return false
	}
	return true
}

// parseTeamID extracts and validates the {id} path value.
func parseTeamID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(r.PathValue("id"))
	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error":"invalid team ID"}`, http.StatusBadRequest)
		return "", false
	}
	return id, true
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// CreateTeam POST /api/v1/teams — admin only.
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if len(body.Name) > 100 {
		http.Error(w, `{"error":"name must not exceed 100 characters"}`, http.StatusBadRequest)
		return
	}

	team, err := h.repo.CreateTeam(r.Context(), body.Name, body.Description)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNameTaken) {
			http.Error(w, `{"error":"team name is already taken"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"failed to create team"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, team)
}

// ListTeams GET /api/v1/teams — any authenticated user.
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := h.repo.ListTeams(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to retrieve teams"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": teams})
}

// GetTeam GET /api/v1/teams/{id} — any authenticated user.
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	id, ok := parseTeamID(w, r)
	if !ok {
		return
	}

	team, err := h.repo.GetTeamByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNotFound) {
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to retrieve team"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, team)
}

// UpdateTeam PATCH /api/v1/teams/{id} — admin only.
func (h *TeamHandler) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	id, ok := parseTeamID(w, r)
	if !ok {
		return
	}

	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		// ClearDescription allows explicitly setting description to null.
		ClearDescription bool `json:"clear_description"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
		return
	}

	if body.Name != nil {
		*body.Name = strings.TrimSpace(*body.Name)
		if *body.Name == "" {
			http.Error(w, `{"error":"name must not be empty"}`, http.StatusBadRequest)
			return
		}
		if len(*body.Name) > 100 {
			http.Error(w, `{"error":"name must not exceed 100 characters"}`, http.StatusBadRequest)
			return
		}
	}

	if body.Name == nil && body.Description == nil && !body.ClearDescription {
		http.Error(w, `{"error":"no fields to update"}`, http.StatusBadRequest)
		return
	}

	setDesc := body.Description != nil || body.ClearDescription
	req := repository.UpdateTeamRequest{
		Name:           body.Name,
		Description:    body.Description,
		SetDescription: setDesc,
	}

	team, err := h.repo.UpdateTeam(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNotFound) {
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, repository.ErrTeamNameTaken) {
			http.Error(w, `{"error":"team name is already taken"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"failed to update team"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, team)
}

// AddTeamMember POST /api/v1/teams/{id}/members — admin only.
func (h *TeamHandler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	teamID, ok := parseTeamID(w, r)
	if !ok {
		return
	}

	var body struct {
		UserID   string `json:"user_id"`
		TeamRole string `json:"team_role"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid JSON request"}`, http.StatusBadRequest)
		return
	}

	body.UserID = strings.TrimSpace(body.UserID)
	if _, err := uuid.Parse(body.UserID); err != nil {
		http.Error(w, `{"error":"invalid user_id"}`, http.StatusBadRequest)
		return
	}

	body.TeamRole = strings.TrimSpace(body.TeamRole)
	if body.TeamRole == "" {
		body.TeamRole = "member"
	}
	if body.TeamRole != "lead" && body.TeamRole != "member" {
		http.Error(w, `{"error":"team_role must be lead or member"}`, http.StatusBadRequest)
		return
	}

	member, err := h.repo.AddTeamMember(r.Context(), teamID, body.UserID, body.TeamRole)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTeamNotFound):
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
		case errors.Is(err, repository.ErrUserNotFound):
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		case errors.Is(err, repository.ErrMembershipExists):
			http.Error(w, `{"error":"user is already a member of this team"}`, http.StatusConflict)
		default:
			http.Error(w, `{"error":"failed to add team member"}`, http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

// ListTeamMembers GET /api/v1/teams/{id}/members — any authenticated user.
func (h *TeamHandler) ListTeamMembers(w http.ResponseWriter, r *http.Request) {
	teamID, ok := parseTeamID(w, r)
	if !ok {
		return
	}

	members, err := h.repo.ListTeamMembers(r.Context(), teamID)
	if err != nil {
		if errors.Is(err, repository.ErrTeamNotFound) {
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to retrieve team members"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": members})
}

// RemoveTeamMember DELETE /api/v1/teams/{id}/members/{userId} — admin only.
func (h *TeamHandler) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}

	teamID, ok := parseTeamID(w, r)
	if !ok {
		return
	}

	userID := strings.TrimSpace(r.PathValue("userId"))
	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, `{"error":"invalid user ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.repo.RemoveTeamMember(r.Context(), teamID, userID); err != nil {
		if errors.Is(err, repository.ErrMembershipNotFound) {
			http.Error(w, `{"error":"membership not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"failed to remove team member"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
