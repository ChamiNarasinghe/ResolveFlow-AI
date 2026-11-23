package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds the allowed origins and methods for CORS.
type CORSConfig struct {
	AllowedOrigins []string
}

// CORS returns middleware that sets CORS headers for allowed origins.
// OPTIONS preflight requests are responded to immediately with 204 — no
// authentication or downstream handler is invoked.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := "GET, POST, PATCH, DELETE, OPTIONS"
	allowedHeaders := "Content-Type, Authorization"

	originSet := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		originSet[strings.TrimRight(o, "/")] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if _, ok := originSet[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
					w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
					w.Header().Set("Vary", "Origin")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
