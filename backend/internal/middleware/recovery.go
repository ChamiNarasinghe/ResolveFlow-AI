package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that catches panics in downstream handlers,
// logs the panic type and a stack trace (server-side only), and writes
// HTTP 500 to the client without exposing any internal details.
//
// The raw panic value is never logged to prevent accidental secret leakage.
// Only its type is recorded so operators can identify the source.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()

					log.Error("panic recovered",
						slog.String("request_id", GetRequestID(r.Context())),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
						// Log only the dynamic type, never the value, to prevent
						// accidental leakage of passwords, tokens, or PII.
						slog.String("panic_type", fmt.Sprintf("%T", rec)),
						slog.String("stack", string(stack)),
					)

					// Return a generic 500 — no internal details to the client.
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
