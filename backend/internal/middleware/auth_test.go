package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chami2002/resolveflow-api/internal/auth"
)

func TestAuthMiddleware(t *testing.T) {
	jwtService, err := auth.NewJWTService(strings.Repeat("a", 32), 15)
	if err != nil {
		t.Fatal(err)
	}

	validToken, err := jwtService.GenerateToken(
		"user-123",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "missing token",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid authorization format",
			authHeader:     "Invalid " + validToken,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + validToken,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := AuthMiddleware(jwtService)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					claims, ok := GetUserClaims(r.Context())
					if !ok {
						t.Error("expected user claims in request context")
						http.Error(w, "Missing claims", http.StatusInternalServerError)
						return
					}

					if claims.UserID != "user-123" {
						t.Errorf("unexpected user ID: %s", claims.UserID)
					}

					w.WriteHeader(http.StatusOK)
				}),
			)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}

func TestRequireRoles(t *testing.T) {
	jwtService, err := auth.NewJWTService(strings.Repeat("a", 32), 15)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		role           string
		allowedRoles   []string
		expectedStatus int
	}{
		{
			name:           "admin allowed",
			role:           "admin",
			allowedRoles:   []string{"admin", "manager"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "manager allowed",
			role:           "manager",
			allowedRoles:   []string{"admin", "manager"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "agent forbidden",
			role:           "agent",
			allowedRoles:   []string{"admin", "manager"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := jwtService.GenerateToken(
				"user-123",
				"user@example.com",
				tt.role,
			)
			if err != nil {
				t.Fatal(err)
			}

			protectedHandler := AuthMiddleware(jwtService)(
				RequireRoles(tt.allowedRoles...)(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					}),
				),
			)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+token)

			recorder := httptest.NewRecorder()
			protectedHandler.ServeHTTP(recorder, req)

			if recorder.Code != tt.expectedStatus {
				t.Errorf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}
