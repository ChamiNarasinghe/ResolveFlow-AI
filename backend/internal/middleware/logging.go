package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code written
// by the downstream handler. The default status is 200 (Go's implicit default
// when WriteHeader is never called).
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

// Write ensures the status is recorded even when a handler calls Write
// without calling WriteHeader first (Go implicitly sends 200 in that case).
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestLogger returns middleware that logs one structured line per request.
//
// Logged fields: request_id, method, path, status, duration_ms, client_ip.
//
// Deliberately excluded to prevent sensitive data leakage:
//   - Authorization / token headers
//   - Request bodies
//   - Query parameters (may contain tokens or PII)
//   - Database connection details
func RequestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			log.Info("request",
				slog.String("request_id", GetRequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int64("duration_ms", time.Since(start).Milliseconds()),
				slog.String("client_ip", logClientIP(r)),
			)
		})
	}
}

// logClientIP extracts the host portion of RemoteAddr for logging purposes.
// It does not consult forwarded headers to keep logging independent of proxy
// configuration.
func logClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
