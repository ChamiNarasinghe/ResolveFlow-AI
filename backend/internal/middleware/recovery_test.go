package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryHandlesPanic(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went very wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(Recovery(log)(panicHandler)).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestRecoveryLogsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("deliberate panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(Recovery(log)(panicHandler)).ServeHTTP(rec, req)

	responseID := rec.Header().Get("X-Request-ID")
	if responseID == "" {
		t.Fatal("expected X-Request-ID header")
	}
	if !strings.Contains(buf.String(), responseID) {
		t.Errorf("expected request ID %s in log output", responseID)
	}
}

func TestRecoveryDoesNotExposeInternalDetails(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("db password is hunter2")
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(Recovery(log)(panicHandler)).ServeHTTP(rec, req)

	// The panic value must appear in the server log but NOT in the response body.
	body := rec.Body.String()
	if strings.Contains(body, "hunter2") {
		t.Error("response body must not expose panic details")
	}
	if strings.Contains(body, "goroutine") {
		t.Error("response body must not expose stack trace")
	}
}

func TestRecoveryServerRemainsOperational(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			panic("first request panics")
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestID(Recovery(log)(handler))

	// First request — panics.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:5678"
	rec1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusInternalServerError {
		t.Errorf("first request: expected 500, got %d", rec1.Code)
	}

	// Second request — must succeed normally.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4:5678"
	rec2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("second request: expected 200, got %d", rec2.Code)
	}
}

func TestRecoveryNoPanicPassesThrough(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf)

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	req.RemoteAddr = "1.2.3.4:5678"
	rec := httptest.NewRecorder()

	RequestID(Recovery(log)(okHandler)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// No panic logged.
	if strings.Contains(buf.String(), "panic recovered") {
		t.Error("unexpected panic log entry for non-panicking handler")
	}
}
