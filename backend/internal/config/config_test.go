package config

import "testing"

func TestLoadDefaultConfig(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")

	cfg := Load()

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

	cfg := Load()

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

	cfg := Load()

	expected := "postgres://test:test@localhost:5433/testdb"

	if cfg.DatabaseURL != expected {
		t.Errorf("Expected database URL %s, got %s", expected, cfg.DatabaseURL)
	}
}
