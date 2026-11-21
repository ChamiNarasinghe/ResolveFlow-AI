package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DatabasePinger defines the database health-check requirement.
type DatabasePinger interface {
	Ping(ctx context.Context) error
}

// ReadinessHandler checks whether the API can access PostgreSQL.
func ReadinessHandler(db DatabasePinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)

			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":   "not_ready",
				"database": "unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "ready",
			"database": "connected",
		})
	}
}
