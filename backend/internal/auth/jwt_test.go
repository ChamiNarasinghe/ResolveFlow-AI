package auth

import (
	"strings"
	"testing"
)

func TestNewJWTService(t *testing.T) {
	_, err := NewJWTService("short", 15)
	if err == nil {
		t.Error("expected error for short JWT secret")
	}

	_, err = NewJWTService(strings.Repeat("a", 32), 0)
	if err == nil {
		t.Error("expected error for invalid expiration")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	service, err := NewJWTService(strings.Repeat("a", 32), 15)
	if err != nil {
		t.Fatalf("failed to create JWT service: %v", err)
	}

	token, err := service.GenerateToken(
		"user-123",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected user ID user-123, got %s", claims.UserID)
	}

	if claims.Email != "agent@example.com" {
		t.Errorf("unexpected email: %s", claims.Email)
	}

	if claims.Role != "agent" {
		t.Errorf("expected role agent, got %s", claims.Role)
	}
}

func TestValidateTokenWithWrongSecret(t *testing.T) {
	service1, _ := NewJWTService(strings.Repeat("a", 32), 15)
	service2, _ := NewJWTService(strings.Repeat("b", 32), 15)

	token, err := service1.GenerateToken(
		"user-123",
		"agent@example.com",
		"agent",
	)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = service2.ValidateToken(token)
	if err == nil {
		t.Error("expected token validation to fail with wrong secret")
	}
}

func TestValidateMalformedToken(t *testing.T) {
	service, err := NewJWTService(strings.Repeat("a", 32), 15)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ValidateToken("invalid.token.value")
	if err == nil {
		t.Error("expected malformed token to be rejected")
	}
}
