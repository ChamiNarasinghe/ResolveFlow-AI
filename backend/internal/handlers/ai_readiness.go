package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// AIPinger defines the AI service health-check requirement.
type AIPinger interface {
	Ping(ctx context.Context) error
}

// AIReadinessHandler checks whether the API can reach the AI service.
func AIReadinessHandler(ai AIPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := ai.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":     "not_ready",
				"ai_service": "unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":     "ready",
			"ai_service": "connected",
		})
	}
}
