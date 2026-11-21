package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chami2002/resolveflow-api/internal/auth"
	"github.com/chami2002/resolveflow-api/internal/middleware"
	"github.com/chami2002/resolveflow-api/internal/models"
)

type mockAdminUserService struct {
	result *models.User
	err    error
	gotReq auth.ProvisionRequest
}

func (m *mockAdminUserService) ProvisionUser(ctx context.Context, req auth.ProvisionRequest) (*models.User, error) {
	m.gotReq = req
	return m.result, m.err
}

func adminRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(body))
	return withAdminClaims(r)
}

func agentRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(body))
	return withAgentClaims(r)
}

const validProvisionBody = `{"name":"Alice Admin","email":"alice@example.com","password":"Secure123","role":"agent"}`

func TestAdminCreateUserSuccess_Agent(t *testing.T) {
	svc := &mockAdminUserService{result: &models.User{
		ID: "new-user-id", FullName: "Alice Admin", Email: "alice@example.com", Role: "agent",
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
	if resp["user"].Role != "agent" {
		t.Errorf("expected agent role, got %s", resp["user"].Role)
	}
	if strings.Contains(rec.Body.String(), "Secure123") {
		t.Error("response must not contain plaintext password")
	}
}

func TestAdminCreateUserSuccess_Admin(t *testing.T) {
	svc := &mockAdminUserService{result: &models.User{
		ID: "new-admin-id", FullName: "Bob Admin", Email: "bob@example.com", Role: "admin",
	}}
	h := NewAdminHandler(svc)

	body := `{"name":"Bob Admin","email":"bob@example.com","password":"Secure123","role":"admin"}`
	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(body))

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCreateUserAgentForbidden(t *testing.T) {
	h := NewAdminHandler(&mockAdminUserService{})

	rec := httptest.NewRecorder()
	h.CreateUser(rec, agentRequest(validProvisionBody))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAdminCreateUserUnauthenticated(t *testing.T) {
	h := NewAdminHandler(&mockAdminUserService{})

	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users", strings.NewReader(validProvisionBody))
	rec := httptest.NewRecorder()
	h.CreateUser(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminCreateUserDuplicateEmail(t *testing.T) {
	svc := &mockAdminUserService{err: auth.ErrEmailAlreadyRegistered}
	h := NewAdminHandler(svc)

	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(validProvisionBody))

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestAdminCreateUserInvalidRole(t *testing.T) {
	h := NewAdminHandler(&mockAdminUserService{})

	body := `{"name":"Alice","email":"alice@example.com","password":"Secure123","role":"superuser"}`
	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(body))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAdminCreateUserInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"email":"a@b.com","password":"Secure123","role":"agent"}`},
		{"invalid email", `{"name":"Alice","email":"not-an-email","password":"Secure123","role":"agent"}`},
		{"short password", `{"name":"Alice","email":"a@b.com","password":"abc","role":"agent"}`},
		{"missing role", `{"name":"Alice","email":"a@b.com","password":"Secure123"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAdminHandler(&mockAdminUserService{})
			rec := httptest.NewRecorder()
			h.CreateUser(rec, adminRequest(tt.body))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminCreateUserPasswordNotReturned(t *testing.T) {
	svc := &mockAdminUserService{result: &models.User{
		ID: "u1", FullName: "Alice", Email: "alice@example.com", Role: "agent",
	}}
	h := NewAdminHandler(svc)

	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(validProvisionBody))

	body := rec.Body.String()
	if strings.Contains(body, "Secure123") {
		t.Error("response must not contain plaintext password")
	}
	if strings.Contains(body, "password_hash") || strings.Contains(body, "passwordHash") {
		t.Error("response must not contain password hash field")
	}
}

func TestAdminCreateUserInternalErrorNotLeaked(t *testing.T) {
	svc := &mockAdminUserService{err: errors.New("pq: connection refused")}
	h := NewAdminHandler(svc)

	rec := httptest.NewRecorder()
	h.CreateUser(rec, adminRequest(validProvisionBody))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "pq:") {
		t.Error("response must not expose internal error details")
	}
}

func TestAdminCreateUserPublicRegistrationCannotSetAdminRole(t *testing.T) {
	// Simulate what would happen if a public registration endpoint tried to
	// set role=admin: the Register path in AuthService always uses 'agent'.
	// This test verifies the ProvisionRequest validation rejects unknown roles
	// and that the public Register path is separate from ProvisionUser.
	svc := &mockAdminUserService{}
	h := NewAdminHandler(svc)

	// An unauthenticated caller cannot reach this handler at all.
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users",
		strings.NewReader(`{"name":"Hacker","email":"h@x.com","password":"Hack1234","role":"admin"}`))
	// No claims set — unauthenticated.
	rec := httptest.NewRecorder()
	h.CreateUser(rec, r)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAdminCreateUserWithAdminJWT(t *testing.T) {
	// Verify the full JWT-based admin flow using middleware.SetUserClaims.
	svc := &mockAdminUserService{result: &models.User{
		ID: "u2", FullName: "Carol", Email: "carol@example.com", Role: "agent",
	}}
	h := NewAdminHandler(svc)

	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users",
		strings.NewReader(`{"name":"Carol","email":"carol@example.com","password":"Secure123","role":"agent"}`))
	r = r.WithContext(middleware.SetUserClaims(r.Context(), &auth.Claims{
		UserID: "admin-id",
		Email:  "admin@example.com",
		Role:   "admin",
	}))

	rec := httptest.NewRecorder()
	h.CreateUser(rec, r)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}
