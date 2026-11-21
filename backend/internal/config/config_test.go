package config

import (
	"errors"
	"strings"
	"testing"
)

const testSecret = "this-is-a-test-secret-32-chars!!"

func TestLoadDefaultConfig(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("JWT_SECRET", testSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", cfg.Port)
	}

	if cfg.Environment != "development" {
		t.Errorf(
			"Expected development, got %s",
			cfg.Environment,
		)
	}
}

func TestLoadCustomConfig(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", testSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Expected port 9000, got %s", cfg.Port)
	}

	if cfg.Environment != "production" {
		t.Errorf(
			"Expected production, got %s",
			cfg.Environment,
		)
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5433/testdb")
	t.Setenv("JWT_SECRET", testSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "postgres://test:test@localhost:5433/testdb"

	if cfg.DatabaseURL != expected {
		t.Errorf("Expected database URL %s, got %s", expected, cfg.DatabaseURL)
	}
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "tooshort")

	_, err := Load()
	if !errors.Is(err, ErrMissingJWTSecret) {
		t.Errorf("expected ErrMissingJWTSecret, got %v", err)
	}
}

func TestLoadRejectsMissingJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	if !errors.Is(err, ErrMissingJWTSecret) {
		t.Errorf("expected ErrMissingJWTSecret, got %v", err)
	}
}

func TestLoadAIServiceAPIKey(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("AI_SERVICE_API_KEY", "test-ai-key-value")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AIServiceAPIKey != "test-ai-key-value" {
		t.Errorf("expected test-ai-key-value, got %s", cfg.AIServiceAPIKey)
	}
}

func TestLoadAIServiceAPIKeyUnset(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)
	t.Setenv("AI_SERVICE_API_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AIServiceAPIKey != "" {
		t.Errorf("expected empty string, got %s", cfg.AIServiceAPIKey)
	}
}

func TestLoadErrorDoesNotExposeSecret(t *testing.T) {
	shortSecret := "tiny"
	t.Setenv("JWT_SECRET", shortSecret)

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), shortSecret) {
		t.Error("error message must not contain the secret value")
	}
}
