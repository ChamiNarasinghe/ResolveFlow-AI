package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestRequestIDSetsResponseHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	RequestID(okHandler).ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}
	if len(id) != 32 {
		t.Errorf("expected 32-char hex ID, got %q (len %d)", id, len(id))
	}
}

func TestRequestIDStoredInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	var capturedID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if capturedID == "" {
		t.Fatal("expected request ID in context, got empty string")
	}
	if capturedID != rec.Header().Get("X-Request-ID") {
		t.Errorf("context ID %q does not match response header %q",
			capturedID, rec.Header().Get("X-Request-ID"))
	}
}

func TestRequestIDIgnoresClientSuppliedHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "attacker-controlled-value")
	rec := httptest.NewRecorder()

	var capturedID string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rec, req)

	if capturedID == "attacker-controlled-value" {
		t.Error("request ID must not be taken from client-supplied header")
	}
	if rec.Header().Get("X-Request-ID") == "attacker-controlled-value" {
		t.Error("response X-Request-ID must not echo client-supplied value")
	}
}

func TestRequestIDUniquePerRequest(t *testing.T) {
	ids := make(map[string]struct{})

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		id := rec.Header().Get("X-Request-ID")
		if _, dup := ids[id]; dup {
			t.Fatalf("duplicate request ID generated: %s", id)
		}
		ids[id] = struct{}{}
	}
}

func TestGetRequestIDEmptyWhenNotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if id := GetRequestID(req.Context()); id != "" {
		t.Errorf("expected empty string, got %q", id)
	}
}

func TestRequestIDConcurrentRequests(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	var (
		mu  sync.Mutex
		ids = make(map[string]struct{})
		wg  sync.WaitGroup
	)

	const n = 200
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			id := rec.Header().Get("X-Request-ID")

			mu.Lock()
			defer mu.Unlock()
			if _, dup := ids[id]; dup {
				t.Errorf("duplicate request ID under concurrency: %s", id)
			}
			ids[id] = struct{}{}
		}()
	}

	wg.Wait()
}
