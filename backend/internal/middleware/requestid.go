package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type requestIDKey struct{}

// RequestID returns middleware that attaches a unique request ID to every
// incoming request.
//
// The ID is generated from 16 cryptographically random bytes encoded as a
// 32-character lowercase hex string. A client-supplied X-Request-ID header
// is never trusted — the server always generates its own ID.
//
// The ID is:
//   - stored in the request context (retrieve with GetRequestID)
//   - written to the X-Request-ID response header before the handler runs
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := generateRequestID()
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID stored by RequestID middleware.
// Returns an empty string when no ID is present.
func GetRequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// generateRequestID returns a 32-character lowercase hex string built from
// 16 cryptographically random bytes. Falls back to a fixed sentinel on the
// (practically impossible) event that rand.Read fails.
func generateRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b)
}
