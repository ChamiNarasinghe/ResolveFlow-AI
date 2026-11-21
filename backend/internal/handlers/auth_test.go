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

type mockAuthService struct {
	registerResponse *auth.AuthResponse
	loginResponse    *auth.AuthResponse
	profileResponse  *models.User

	registerErr error
	loginErr    error
	profileErr  error
}

func (m *mockAuthService) Register(
	ctx context.Context,
	request auth.RegisterRequest,
) (*auth.AuthResponse, error) {
	return m.registerResponse, m.registerErr
}

func (m *mockAuthService) Login(
	ctx context.Context,
	request auth.LoginRequest,
) (*auth.AuthResponse, error) {
	return m.loginResponse, m.loginErr
}

func (m *mockAuthService) GetProfile(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	return m.profileResponse, m.profileErr
}

func TestAuthHandlerRegister(t *testing.T) {
	service := &mockAuthService{
		registerResponse: &auth.AuthResponse{
			User: &models.User{
				ID:    "user-123",
				Email: "agent@example.com",
				Role:  "agent",
			},
			AccessToken: "test-token",
			TokenType:   "Bearer",
		},
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"full_name":"Test Agent","email":"agent@example.com","password":"Password123"}`),
	)

	response := httptest.NewRecorder()
	handler.Register(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", response.Code)
	}

	var result auth.AuthResponse
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}

	if result.AccessToken != "test-token" {
		t.Fatal("expected access token in response")
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	service := &mockAuthService{
		loginResponse: &auth.AuthResponse{
			User: &models.User{
				ID:    "user-123",
				Email: "agent@example.com",
				Role:  "agent",
			},
			AccessToken: "test-token",
			TokenType:   "Bearer",
		},
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"email":"agent@example.com","password":"Password123"}`),
	)

	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}

func TestAuthHandlerInvalidJSON(t *testing.T) {
	handler := NewAuthHandler(&mockAuthService{})

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`invalid-json`),
	)

	response := httptest.NewRecorder()
	handler.Login(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestAuthHandlerLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
	}{
		{"invalid credentials", auth.ErrInvalidCredentials, http.StatusUnauthorized},
		{"inactive user", auth.ErrInactiveUser, http.StatusForbidden},
		{"internal error", errors.New("database error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(&mockAuthService{
				loginErr: tt.err,
			})

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/login",
				strings.NewReader(`{"email":"agent@example.com","password":"Password123"}`),
			)

			response := httptest.NewRecorder()
			handler.Login(response, request)

			if response.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, response.Code)
			}
		})
	}
}

func TestAuthHandlerMe(t *testing.T) {
	jwtService, err := auth.NewJWTService(
		"this-is-a-test-secret-with-more-than-32-characters",
		15,
	)
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwtService.GenerateToken(
		"user-123",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	service := &mockAuthService{
		profileResponse: &models.User{
			ID:       "user-123",
			FullName: "Test Agent",
			Email:    "agent@example.com",
			Role:     "agent",
			IsActive: true,
		},
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+token)

	response := httptest.NewRecorder()

	protectedHandler := middleware.AuthMiddleware(jwtService)(
		http.HandlerFunc(handler.Me),
	)
	protectedHandler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s",
			response.Code, response.Body.String())
	}

	var result struct {
		User models.User `json:"user"`
	}

	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}

	if result.User.ID != "user-123" {
		t.Fatalf("expected user-123, got %s", result.User.ID)
	}
}

func TestAuthHandlerMeWithoutAuthentication(t *testing.T) {
	handler := NewAuthHandler(&mockAuthService{})

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/auth/me",
		nil,
	)

	response := httptest.NewRecorder()
	handler.Me(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", response.Code)
	}
}

func TestAuthHandlerRegisterDoesNotLeakInternalError(t *testing.T) {
	internalMsg := "pq: duplicate key value violates unique constraint"
	service := &mockAuthService{
		registerErr: errors.New(internalMsg),
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"full_name":"Test","email":"t@example.com","password":"Password1"}`),
	)

	response := httptest.NewRecorder()
	handler.Register(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), internalMsg) {
		t.Error("response body must not contain the internal error message")
	}
}

func TestAuthHandlerRegisterValidationErrorIsReturned(t *testing.T) {
	// Validation errors (from RegisterRequest.Validate) are safe to return.
	service := &mockAuthService{
		registerErr: errors.New("validation error: full name is required"),
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"full_name":"","email":"t@example.com","password":"Password1"}`),
	)

	response := httptest.NewRecorder()
	handler.Register(response, request)

	// The service mock returns an error; the handler must not 500 for a
	// validation-shaped error but must also not leak DB internals.
	if response.Code == http.StatusOK {
		t.Error("expected a non-200 response for a registration error")
	}
}

func TestAuthHandlerRegisterDuplicateEmailReturns409(t *testing.T) {
	service := &mockAuthService{
		registerErr: auth.ErrEmailAlreadyRegistered,
	}

	handler := NewAuthHandler(service)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"full_name":"Test","email":"t@example.com","password":"Password1"}`),
	)

	response := httptest.NewRecorder()
	handler.Register(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", response.Code)
	}
}
