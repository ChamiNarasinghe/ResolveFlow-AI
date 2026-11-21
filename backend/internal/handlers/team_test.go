package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
	"github.com/chami2002/resolveflow-api/internal/repository"
)

// --- mock ---

type mockTeamRepository struct {
	createTeamResult    *models.Team
	createTeamErr       error
	listTeamsResult     []models.Team
	listTeamsErr        error
	getTeamResult       *models.Team
	getTeamErr          error
	updateTeamResult    *models.Team
	updateTeamErr       error
	addMemberResult     *models.TeamMember
	addMemberErr        error
	removeMemberErr     error
	listMembersResult   []models.TeamMember
	listMembersErr      error

	gotCreateName        string
	gotCreateDescription *string
	gotUpdateID          string
	gotUpdateReq         repository.UpdateTeamRequest
	gotAddTeamID         string
	gotAddUserID         string
	gotAddRole           string
	gotRemoveTeamID      string
	gotRemoveUserID      string
	gotListMembersTeamID string
}

func (m *mockTeamRepository) CreateTeam(ctx context.Context, name string, description *string) (*models.Team, error) {
	m.gotCreateName = name
	m.gotCreateDescription = description
	return m.createTeamResult, m.createTeamErr
}

func (m *mockTeamRepository) ListTeams(ctx context.Context) ([]models.Team, error) {
	return m.listTeamsResult, m.listTeamsErr
}

func (m *mockTeamRepository) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	return m.getTeamResult, m.getTeamErr
}

func (m *mockTeamRepository) UpdateTeam(ctx context.Context, id string, req repository.UpdateTeamRequest) (*models.Team, error) {
	m.gotUpdateID = id
	m.gotUpdateReq = req
	return m.updateTeamResult, m.updateTeamErr
}

func (m *mockTeamRepository) AddTeamMember(ctx context.Context, teamID, userID, teamRole string) (*models.TeamMember, error) {
	m.gotAddTeamID = teamID
	m.gotAddUserID = userID
	m.gotAddRole = teamRole
	return m.addMemberResult, m.addMemberErr
}

func (m *mockTeamRepository) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	m.gotRemoveTeamID = teamID
	m.gotRemoveUserID = userID
	return m.removeMemberErr
}

func (m *mockTeamRepository) ListTeamMembers(ctx context.Context, teamID string) ([]models.TeamMember, error) {
	m.gotListMembersTeamID = teamID
	return m.listMembersResult, m.listMembersErr
}

// --- helpers ---

const (
	testTeamID = "a2f66976-199f-4ec9-8774-21f937befe13"
	testUserID = "b3e77087-2aa0-5fda-9885-32f048cefe24"
)

func withAdminClaims(r *http.Request) *http.Request {
	return r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID: "admin-id",
		Email:  "admin@example.com",
		Role:   "admin",
	}))
}

func withAgentClaims(r *http.Request) *http.Request {
	return r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID: "agent-id",
		Email:  "agent@example.com",
		Role:   "agent",
	}))
}

func newTeam() *models.Team {
	return &models.Team{ID: testTeamID, Name: "Support", CreatedAt: time.Now()}
}

func newMember() *models.TeamMember {
	return &models.TeamMember{TeamID: testTeamID, UserID: testUserID, TeamRole: "member", JoinedAt: time.Now()}
}

// --- CreateTeam ---

func TestCreateTeamSuccess(t *testing.T) {
	repo := &mockTeamRepository{createTeamResult: newTeam()}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":"Support"}`))
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.gotCreateName != "Support" {
		t.Errorf("expected name Support, got %s", repo.gotCreateName)
	}
	if !strings.Contains(rec.Body.String(), testTeamID) {
		t.Error("expected team ID in response")
	}
}

func TestCreateTeamRequiresAdmin(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":"Support"}`))
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestCreateTeamUnauthenticated(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":"Support"}`))
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestCreateTeamMissingName(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":""}`))
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestCreateTeamDuplicateName(t *testing.T) {
	repo := &mockTeamRepository{createTeamErr: repository.ErrTeamNameTaken}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":"Support"}`))
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestCreateTeamDatabaseError(t *testing.T) {
	repo := &mockTeamRepository{createTeamErr: errors.New("db failure")}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams",
		strings.NewReader(`{"name":"Support"}`))
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.CreateTeam(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}

// --- ListTeams ---

func TestListTeamsSuccess(t *testing.T) {
	repo := &mockTeamRepository{listTeamsResult: []models.Team{*newTeam()}}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Error("expected data key in response")
	}
}

func TestListTeamsEmpty(t *testing.T) {
	repo := &mockTeamRepository{listTeamsResult: []models.Team{}}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeams(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Error("expected empty data array")
	}
}

func TestListTeamsDatabaseError(t *testing.T) {
	repo := &mockTeamRepository{listTeamsErr: errors.New("db failure")}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeams(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- GetTeam ---

func TestGetTeamSuccess(t *testing.T) {
	repo := &mockTeamRepository{getTeamResult: newTeam()}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+testTeamID, nil)
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.GetTeam(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), testTeamID) {
		t.Error("expected team ID in response")
	}
}

func TestGetTeamNotFound(t *testing.T) {
	repo := &mockTeamRepository{getTeamErr: repository.ErrTeamNotFound}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+testTeamID, nil)
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.GetTeam(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestGetTeamInvalidUUID(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/not-a-uuid", nil)
	req.SetPathValue("id", "not-a-uuid")
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.GetTeam(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- UpdateTeam ---

func TestUpdateTeamSuccess(t *testing.T) {
	updated := newTeam()
	updated.Name = "Billing"
	repo := &mockTeamRepository{updateTeamResult: updated}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+testTeamID,
		strings.NewReader(`{"name":"Billing"}`))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.UpdateTeam(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.gotUpdateID != testTeamID {
		t.Errorf("expected team ID %s, got %s", testTeamID, repo.gotUpdateID)
	}
}

func TestUpdateTeamRequiresAdmin(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+testTeamID,
		strings.NewReader(`{"name":"Billing"}`))
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.UpdateTeam(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestUpdateTeamNotFound(t *testing.T) {
	repo := &mockTeamRepository{updateTeamErr: repository.ErrTeamNotFound}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+testTeamID,
		strings.NewReader(`{"name":"Billing"}`))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.UpdateTeam(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateTeamDuplicateName(t *testing.T) {
	repo := &mockTeamRepository{updateTeamErr: repository.ErrTeamNameTaken}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+testTeamID,
		strings.NewReader(`{"name":"Existing"}`))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.UpdateTeam(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestUpdateTeamNoFields(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/teams/"+testTeamID,
		strings.NewReader(`{}`))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.UpdateTeam(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- AddTeamMember ---

func TestAddTeamMemberSuccess(t *testing.T) {
	repo := &mockTeamRepository{addMemberResult: newMember()}
	h := NewTeamHandler(repo)

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if repo.gotAddTeamID != testTeamID {
		t.Errorf("expected team ID %s, got %s", testTeamID, repo.gotAddTeamID)
	}
	if repo.gotAddUserID != testUserID {
		t.Errorf("expected user ID %s, got %s", testUserID, repo.gotAddUserID)
	}
	if repo.gotAddRole != "member" {
		t.Errorf("expected role member, got %s", repo.gotAddRole)
	}
}

func TestAddTeamMemberRequiresAdmin(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAddTeamMemberTeamNotFound(t *testing.T) {
	repo := &mockTeamRepository{addMemberErr: repository.ErrTeamNotFound}
	h := NewTeamHandler(repo)

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAddTeamMemberUserNotFound(t *testing.T) {
	repo := &mockTeamRepository{addMemberErr: repository.ErrUserNotFound}
	h := NewTeamHandler(repo)

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAddTeamMemberDuplicate(t *testing.T) {
	repo := &mockTeamRepository{addMemberErr: repository.ErrMembershipExists}
	h := NewTeamHandler(repo)

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rec.Code)
	}
}

func TestAddTeamMemberInvalidUserID(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	body := `{"user_id":"not-a-uuid","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAddTeamMemberInvalidRole(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	body := `{"user_id":"` + testUserID + `","team_role":"superuser"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAddTeamMemberDatabaseError(t *testing.T) {
	repo := &mockTeamRepository{addMemberErr: errors.New("db failure")}
	h := NewTeamHandler(repo)

	body := `{"user_id":"` + testUserID + `","team_role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teams/"+testTeamID+"/members",
		strings.NewReader(body))
	req.SetPathValue("id", testTeamID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.AddTeamMember(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}

// --- ListTeamMembers ---

func TestListTeamMembersSuccess(t *testing.T) {
	repo := &mockTeamRepository{listMembersResult: []models.TeamMember{*newMember()}}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/members", nil)
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeamMembers(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"data"`) {
		t.Error("expected data key in response")
	}
}

func TestListTeamMembersTeamNotFound(t *testing.T) {
	repo := &mockTeamRepository{listMembersErr: repository.ErrTeamNotFound}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/"+testTeamID+"/members", nil)
	req.SetPathValue("id", testTeamID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeamMembers(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListTeamMembersInvalidUUID(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teams/bad-id/members", nil)
	req.SetPathValue("id", "bad-id")
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.ListTeamMembers(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// --- RemoveTeamMember ---

func TestRemoveTeamMemberSuccess(t *testing.T) {
	repo := &mockTeamRepository{}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/teams/"+testTeamID+"/members/"+testUserID, nil)
	req.SetPathValue("id", testTeamID)
	req.SetPathValue("userId", testUserID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if repo.gotRemoveTeamID != testTeamID {
		t.Errorf("expected team ID %s, got %s", testTeamID, repo.gotRemoveTeamID)
	}
	if repo.gotRemoveUserID != testUserID {
		t.Errorf("expected user ID %s, got %s", testUserID, repo.gotRemoveUserID)
	}
}

func TestRemoveTeamMemberRequiresAdmin(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/teams/"+testTeamID+"/members/"+testUserID, nil)
	req.SetPathValue("id", testTeamID)
	req.SetPathValue("userId", testUserID)
	req = withAgentClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestRemoveTeamMemberNotFound(t *testing.T) {
	repo := &mockTeamRepository{removeMemberErr: repository.ErrMembershipNotFound}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/teams/"+testTeamID+"/members/"+testUserID, nil)
	req.SetPathValue("id", testTeamID)
	req.SetPathValue("userId", testUserID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestRemoveTeamMemberInvalidTeamUUID(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/teams/bad/members/"+testUserID, nil)
	req.SetPathValue("id", "bad")
	req.SetPathValue("userId", testUserID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRemoveTeamMemberInvalidUserUUID(t *testing.T) {
	h := NewTeamHandler(&mockTeamRepository{})

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/teams/"+testTeamID+"/members/bad", nil)
	req.SetPathValue("id", testTeamID)
	req.SetPathValue("userId", "bad")
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRemoveTeamMemberDatabaseError(t *testing.T) {
	repo := &mockTeamRepository{removeMemberErr: errors.New("db failure")}
	h := NewTeamHandler(repo)

	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/teams/"+testTeamID+"/members/"+testUserID, nil)
	req.SetPathValue("id", testTeamID)
	req.SetPathValue("userId", testUserID)
	req = withAdminClaims(req)
	rec := httptest.NewRecorder()
	h.RemoveTeamMember(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "db failure") {
		t.Error("response must not expose internal error details")
	}
}
