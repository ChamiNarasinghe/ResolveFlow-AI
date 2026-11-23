package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
)

// ── mock auth service for change-password tests ───────────────────────────────

type mockChangePasswordService struct {
	// embed the fields used by existing mockAuthService tests
	registerResponse *auth.AuthResponse
	loginResponse    *auth.AuthResponse
	profileResponse  *models.User

	registerErr error
	loginErr    error
	profileErr  error

	// change-password
	changePasswordResponse *auth.AuthResponse
	changePasswordErr      error
	gotUserID              string
	gotReq                 auth.ChangePasswordRequest
}

func (m *mockChangePasswordService) Register(ctx context.Context, req auth.RegisterRequest) (*auth.AuthResponse, error) {
	return m.registerResponse, m.registerErr
}

func (m *mockChangePasswordService) Login(ctx context.Context, req auth.LoginRequest) (*auth.AuthResponse, error) {
	return m.loginResponse, m.loginErr
}

func (m *mockChangePasswordService) GetProfile(ctx context.Context, userID string) (*models.User, error) {
	return m.profileResponse, m.profileErr
}

func (m *mockChangePasswordService) ChangePassword(ctx context.Context, userID string, req auth.ChangePasswordRequest) (*auth.AuthResponse, error) {
	m.gotUserID = userID
	m.gotReq = req
	return m.changePasswordResponse, m.changePasswordErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

func changePasswordRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(body))
	return r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID: "user-123",
		Email:  "agent@example.com",
		Role:   "agent",
	}))
}

const validChangeBody = `{"current_password":"OldPass1","new_password":"NewPass2","confirm_password":"NewPass2"}`

// ── handler tests ─────────────────────────────────────────────────────────────

func TestChangePasswordSuccess(t *testing.T) {
	svc := &mockChangePasswordService{
		changePasswordResponse: &auth.AuthResponse{
			User:        &models.User{ID: "user-123", Email: "agent@example.com", Role: "agent"},
			AccessToken: "new-token",
			TokenType:   "Bearer",
		},
	}
	h := NewAuthHandler(svc)

	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(validChangeBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp auth.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.AccessToken != "new-token" {
		t.Errorf("expected new-token, got %s", resp.AccessToken)
	}
	if strings.Contains(rec.Body.String(), "OldPass1") || strings.Contains(rec.Body.String(), "NewPass2") {
		t.Error("response must not contain plaintext passwords")
	}
	if svc.gotUserID != "user-123" {
		t.Errorf("expected userID user-123, got %s", svc.gotUserID)
	}
}

func TestChangePasswordUnauthenticated(t *testing.T) {
	h := NewAuthHandler(&mockChangePasswordService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(validChangeBody))
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestChangePasswordIncorrectCurrentPassword(t *testing.T) {
	svc := &mockChangePasswordService{changePasswordErr: auth.ErrIncorrectPassword}
	h := NewAuthHandler(svc)

	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(validChangeBody))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "OldPass1") {
		t.Error("response must not contain plaintext password")
	}
}

func TestChangePasswordWeakNewPassword(t *testing.T) {
	svc := &mockChangePasswordService{
		changePasswordErr: errors.New("validation error: password must contain at least 8 characters"),
	}
	// Simulate validation error via IsValidationError — use the real sentinel.
	svc.changePasswordErr = fmt.Errorf("%w: password must contain at least 8 characters", auth.ErrValidation)
	h := NewAuthHandler(svc)

	body := `{"current_password":"OldPass1","new_password":"abc","confirm_password":"abc"}`
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordConfirmationMismatch(t *testing.T) {
	svc := &mockChangePasswordService{
		changePasswordErr: fmt.Errorf("%w: passwords do not match", auth.ErrValidation),
	}
	h := NewAuthHandler(svc)

	body := `{"current_password":"OldPass1","new_password":"NewPass2","confirm_password":"Different3"}`
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordSameAsCurrentPassword(t *testing.T) {
	svc := &mockChangePasswordService{
		changePasswordErr: fmt.Errorf("%w: new password must differ from current password", auth.ErrValidation),
	}
	h := NewAuthHandler(svc)

	body := `{"current_password":"OldPass1","new_password":"OldPass1","confirm_password":"OldPass1"}`
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordInvalidJSON(t *testing.T) {
	h := NewAuthHandler(&mockChangePasswordService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", strings.NewReader(`not-json`))
	r = r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{UserID: "u1", Role: "agent"}))
	rec := httptest.NewRecorder()
	h.ChangePassword(rec, r)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChangePasswordInternalErrorNotLeaked(t *testing.T) {
	svc := &mockChangePasswordService{changePasswordErr: errors.New("pq: connection refused")}
	h := NewAuthHandler(svc)

	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(validChangeBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "pq:") {
		t.Error("response must not expose internal error details")
	}
}

func TestChangePasswordResponseDoesNotContainHash(t *testing.T) {
	svc := &mockChangePasswordService{
		changePasswordResponse: &auth.AuthResponse{
			User:        &models.User{ID: "user-123", Role: "agent"},
			AccessToken: "tok",
			TokenType:   "Bearer",
		},
	}
	h := NewAuthHandler(svc)

	rec := httptest.NewRecorder()
	h.ChangePassword(rec, changePasswordRequest(validChangeBody))

	body := rec.Body.String()
	if strings.Contains(body, "password_hash") || strings.Contains(body, "passwordHash") {
		t.Error("response must not contain password hash field")
	}
}

// ── RequirePasswordChanged middleware tests ───────────────────────────────────

func TestRequirePasswordChangedBlocks(t *testing.T) {
	// A token with MustChangePassword = true must be blocked on normal routes.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.RequirePasswordChanged(inner)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	r = r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID:             "u1",
		Role:               "agent",
		MustChangePassword: true,
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "password_change_required") {
		t.Error("expected machine-readable error key in response")
	}
}

func TestRequirePasswordChangedAllows(t *testing.T) {
	// A token with MustChangePassword = false must pass through.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.RequirePasswordChanged(inner)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	r = r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID:             "u1",
		Role:               "agent",
		MustChangePassword: false,
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequirePasswordChangedFailsClosedWithoutClaims(t *testing.T) {
	// No claims in context — must fail closed with 401.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := middleware.RequirePasswordChanged(inner)

	r := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// ── Provisioning flag tests ───────────────────────────────────────────────────

func TestProvisionedUserReceivesMustChangePasswordFlag(t *testing.T) {
	// ProvisionUser must return a user with MustChangePassword = true.
	svc := &mockAdminUserService{result: &models.User{
		ID:                 "new-user-id",
		FullName:           "Alice",
		Email:              "alice@example.com",
		Role:               "agent",
		MustChangePassword: true,
	}}
	h := NewAdminHandler(svc)

	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(validProvisionBody))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp["user"].MustChangePassword {
		t.Error("expected must_change_password = true for provisioned user")
	}
}

func TestBootstrapAndExistingUsersDoNotReceiveMustChangePasswordFlag(t *testing.T) {
	// Users created via Register or Bootstrap have MustChangePassword = false.
	svc := &mockAuthService{
		loginResponse: &auth.AuthResponse{
			User:        &models.User{ID: "u1", Role: "admin", MustChangePassword: false},
			AccessToken: "tok",
			TokenType:   "Bearer",
		},
	}
	h := NewAuthHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"email":"admin@example.com","password":"Admin123"}`))
	rec := httptest.NewRecorder()
	h.Login(rec, r)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp auth.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.User.MustChangePassword {
		t.Error("expected must_change_password = false for bootstrap/existing user")
	}
}
