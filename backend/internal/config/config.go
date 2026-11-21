package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	Port          string
	Environment   string
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration int
	AIServiceURL  string
}

// Load reads configuration from environment variables.
func Load() Config {
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

	return Config{
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTExpiration: jwtExpiration,
		Port:          port,
		Environment:   environment,
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		AIServiceURL:  aiServiceURL,
	}
}
