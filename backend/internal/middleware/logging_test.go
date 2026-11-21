package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// newTestLogger returns a JSON slog.Logger writing to buf.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

func TestRequestLoggerLogsMethod(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tickets", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	chain := RequestID(RequestLogger(log)(okHandler))
	chain.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	if entry["method"] != "POST" {
		t.Errorf("expected method POST, got %v", entry["method"])
	}
}

func TestRequestLoggerLogsPath(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tickets", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["path"] != "/api/v1/tickets" {
		t.Errorf("expected path /api/v1/tickets, got %v", entry["path"])
	}
}

func TestRequestLoggerCapturesStatus200(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// JSON numbers decode as float64.
	if entry["status"] != float64(200) {
		t.Errorf("expected status 200, got %v", entry["status"])
	}
}

func TestRequestLoggerCapturesStatus404(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	notFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(notFound)).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["status"] != float64(404) {
		t.Errorf("expected status 404, got %v", entry["status"])
	}
}

func TestRequestLoggerCapturesStatus500(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	errHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(errHandler)).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["status"] != float64(500) {
		t.Errorf("expected status 500, got %v", entry["status"])
	}
}

func TestRequestLoggerIncludesDurationMS(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := entry["duration_ms"]; !ok {
		t.Error("expected duration_ms field in log entry")
	}
}

func TestRequestLoggerIncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	responseID := rec.Header().Get("X-Request-ID")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["request_id"] != responseID {
		t.Errorf("log request_id %v does not match response header %s",
			entry["request_id"], responseID)
	}
}

func TestRequestLoggerExcludesAuthorizationHeader(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer super-secret-jwt-token")
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "super-secret-jwt-token") {
		t.Error("log output must not contain JWT token value")
	}
	if strings.Contains(buf.String(), "Authorization") {
		t.Error("log output must not contain Authorization header name")
	}
}

func TestRequestLoggerExcludesQueryParameters(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/search?token=secret123&password=hunter2", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(RequestLogger(log)(okHandler)).ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "secret123") {
		t.Error("log output must not contain query parameter values")
	}
	if strings.Contains(buf.String(), "hunter2") {
		t.Error("log output must not contain password query parameter")
	}
}

func TestRequestLoggerConcurrentRequests(t *testing.T) {
	var buf bytes.Buffer
	var mu sync.Mutex

	// Thread-safe writer for the logger.
	log := slog.New(slog.NewJSONHandler(&safeWriter{buf: &buf, mu: &mu},
		&slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := RequestID(RequestLogger(log)(okHandler))

	var wg sync.WaitGroup
	const n = 50
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "1.2.3.4:5678"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()
}

func TestRequestLoggerCapturesPanicRecovery500(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("deliberate panic")
	})

	// Chain: RequestID → RequestLogger → Recovery → handler
	// This mirrors the production middleware order fixed in routes.go.
	chain := RequestID(RequestLogger(log)(Recovery(log)(panicHandler)))

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()
	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	// RequestLogger must have emitted exactly one log line with status=500.
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var requestLogLine string
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["msg"] == "request" {
			requestLogLine = line
			break
		}
	}
	if requestLogLine == "" {
		t.Fatal("RequestLogger did not emit a log line for the panicking request")
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(requestLogLine), &entry); err != nil {
		t.Fatalf("request log line is not valid JSON: %v", err)
	}
	if entry["status"] != float64(500) {
		t.Errorf("expected status 500 in request log, got %v", entry["status"])
	}
}

// safeWriter is a mutex-protected io.Writer used in concurrent log tests.
type safeWriter struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (sw *safeWriter) Write(p []byte) (int, error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.buf.Write(p)
}
