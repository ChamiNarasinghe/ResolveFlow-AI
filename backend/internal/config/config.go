package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config holds the application configuration.
type Config struct {
	Port               string
	Environment        string
	DatabaseURL        string
	JWTSecret          string
	JWTExpiration      int
	AIServiceURL       string
	LogLevel           string
	BootstrapAdminName  string
	BootstrapAdminEmail string
	BootstrapAdminPass  string
	AIServiceAPIKey     string
	AllowedOrigins      []string
}

// ErrMissingJWTSecret is returned by Load when JWT_SECRET is absent or too short.
var ErrMissingJWTSecret = errors.New("JWT_SECRET must be at least 32 characters")

// Load reads configuration from environment variables.
// It returns an error when required security-critical values are invalid.
func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}

	aiServiceURL := os.Getenv("AI_SERVICE_URL")
	if aiServiceURL == "" {
		aiServiceURL = "http://127.0.0.1:8001"
	}

	jwtExpiration := 15
	if value := os.Getenv("JWT_EXPIRATION_MINUTES"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			jwtExpiration = parsed
		}
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, ErrMissingJWTSecret
	}

	allowedOrigins := parseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:3000"}
	}

	return Config{
		JWTSecret:           jwtSecret,
		JWTExpiration:       jwtExpiration,
		Port:                port,
		Environment:         environment,
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		AIServiceURL:        aiServiceURL,
		LogLevel:            logLevel,
		BootstrapAdminName:  os.Getenv("BOOTSTRAP_ADMIN_NAME"),
		BootstrapAdminEmail: os.Getenv("BOOTSTRAP_ADMIN_EMAIL"),
		BootstrapAdminPass:  os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
		AIServiceAPIKey:     os.Getenv("AI_SERVICE_API_KEY"),
		AllowedOrigins:      allowedOrigins,
	}, nil
}

// parseOrigins splits a comma-separated list of origins, trimming whitespace.
func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if o := strings.TrimSpace(p); o != "" {
			out = append(out, o)
		}
	}
	return out
}
