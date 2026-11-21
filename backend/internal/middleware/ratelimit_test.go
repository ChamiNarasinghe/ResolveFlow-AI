package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// okHandler is a trivial handler that always returns 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func newTestLimiter(t *testing.T, requestsPerMinute int) *RateLimiter {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewRateLimiter(ctx, RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
	})
}

// newTestLimiterWithProxies creates a limiter with explicit trusted proxies.
func newTestLimiterWithProxies(t *testing.T, requestsPerMinute int, proxies ...string) *RateLimiter {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return NewRateLimiter(ctx, RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		TrustedProxies:    proxies,
	})
}

func makeRequest(handler http.Handler, ip string) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ip + ":12345"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code
}

// ---------------------------------------------------------------------------
// Basic allow / deny
// ---------------------------------------------------------------------------

func TestRateLimiterAllowsRequestsWithinLimit(t *testing.T) {
	rl := newTestLimiter(t, 5)
	handler := rl.Limit(okHandler)

	for i := 0; i < 5; i++ {
		if code := makeRequest(handler, "1.2.3.4"); code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, code)
		}
	}
}

func TestRateLimiterRejectsRequestsOverLimit(t *testing.T) {
	rl := newTestLimiter(t, 3)
	handler := rl.Limit(okHandler)

	for i := 0; i < 3; i++ {
		makeRequest(handler, "1.2.3.4")
	}

	if code := makeRequest(handler, "1.2.3.4"); code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// HTTP 429 response shape
// ---------------------------------------------------------------------------

func TestRateLimiterResponseHeaders(t *testing.T) {
	rl := newTestLimiter(t, 1)
	handler := rl.Limit(okHandler)

	makeRequest(handler, "1.2.3.4") // consume the single token

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "1.2.3.4:9999"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json content-type, got %s", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	if body["error"] == "" {
		t.Error("expected non-empty error field in JSON body")
	}
}

// ---------------------------------------------------------------------------
// Independent limits per IP
// ---------------------------------------------------------------------------

func TestRateLimiterIndependentPerIP(t *testing.T) {
	rl := newTestLimiter(t, 2)
	handler := rl.Limit(okHandler)

	// Exhaust IP A.
	makeRequest(handler, "10.0.0.1")
	makeRequest(handler, "10.0.0.1")

	// IP A should now be rate-limited.
	if code := makeRequest(handler, "10.0.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("IP A: expected 429, got %d", code)
	}

	// IP B should still be allowed.
	if code := makeRequest(handler, "10.0.0.2"); code != http.StatusOK {
		t.Fatalf("IP B: expected 200, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// Login limit (5 rpm)
// ---------------------------------------------------------------------------

func TestLoginRateLimit(t *testing.T) {
	rl := newTestLimiter(t, 5)
	handler := rl.Limit(okHandler)

	for i := 0; i < 5; i++ {
		if code := makeRequest(handler, "2.2.2.2"); code != http.StatusOK {
			t.Fatalf("login request %d: expected 200, got %d", i+1, code)
		}
	}

	if code := makeRequest(handler, "2.2.2.2"); code != http.StatusTooManyRequests {
		t.Fatalf("6th login request: expected 429, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// Registration limit (3 rpm)
// ---------------------------------------------------------------------------

func TestRegistrationRateLimit(t *testing.T) {
	rl := newTestLimiter(t, 3)
	handler := rl.Limit(okHandler)

	for i := 0; i < 3; i++ {
		if code := makeRequest(handler, "3.3.3.3"); code != http.StatusOK {
			t.Fatalf("register request %d: expected 200, got %d", i+1, code)
		}
	}

	if code := makeRequest(handler, "3.3.3.3"); code != http.StatusTooManyRequests {
		t.Fatalf("4th register request: expected 429, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// General API limit (100 rpm)
// ---------------------------------------------------------------------------

func TestGeneralAPIRateLimit(t *testing.T) {
	rl := newTestLimiter(t, 100)
	handler := rl.Limit(okHandler)

	for i := 0; i < 100; i++ {
		if code := makeRequest(handler, "4.4.4.4"); code != http.StatusOK {
			t.Fatalf("general request %d: expected 200, got %d", i+1, code)
		}
	}

	if code := makeRequest(handler, "4.4.4.4"); code != http.StatusTooManyRequests {
		t.Fatalf("101st general request: expected 429, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// Health endpoints bypass (tested via clientIP + allow logic)
// Health routes are registered without the limiter in routes.go;
// this test confirms the limiter itself does not special-case any path,
// so health endpoints are safe as long as they are wired without it.
// ---------------------------------------------------------------------------

func TestRateLimiterDoesNotSpecialCasePaths(t *testing.T) {
	rl := newTestLimiter(t, 1)
	handler := rl.Limit(okHandler)

	// First request on /health succeeds.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "5.5.5.5:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Second request on /health from same IP is limited (limiter is applied).
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when limiter is applied to /health, got %d", rec2.Code)
	}
}

// ---------------------------------------------------------------------------
// clientIP extraction — method on RateLimiter
// ---------------------------------------------------------------------------

func TestClientIPFromRemoteAddr(t *testing.T) {
	rl := newTestLimiter(t, 10)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:4321"

	if ip := rl.clientIP(req); ip != "203.0.113.5" {
		t.Errorf("expected 203.0.113.5, got %s", ip)
	}
}

// No trusted proxies configured — XFF must always be ignored.
func TestClientIPIgnoresXFFWhenNoProxiesConfigured(t *testing.T) {
	rl := newTestLimiter(t, 10) // no TrustedProxies
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:4321"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")

	// Even though RemoteAddr is loopback, no proxies are trusted.
	if ip := rl.clientIP(req); ip != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1 (RemoteAddr), got %s", ip)
	}
}

// Explicit trusted proxy — XFF is honoured.
func TestClientIPTrustsXFFFromConfiguredProxy(t *testing.T) {
	rl := newTestLimiterWithProxies(t, 10, "127.0.0.1")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:4321"
	req.Header.Set("X-Forwarded-For", "203.0.113.99, 10.0.0.1")

	if ip := rl.clientIP(req); ip != "203.0.113.99" {
		t.Errorf("expected 203.0.113.99, got %s", ip)
	}
}

// Trusted proxy configured as CIDR — XFF is honoured for any IP in range.
func TestClientIPTrustsXFFFromProxyCIDR(t *testing.T) {
	rl := newTestLimiterWithProxies(t, 10, "10.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.1.5:4321"
	req.Header.Set("X-Forwarded-For", "203.0.113.42")

	if ip := rl.clientIP(req); ip != "203.0.113.42" {
		t.Errorf("expected 203.0.113.42, got %s", ip)
	}
}

// Private IP that is NOT in the trusted proxy list — XFF must be ignored.
func TestClientIPIgnoresXFFFromUntrustedPrivateIP(t *testing.T) {
	rl := newTestLimiterWithProxies(t, 10, "127.0.0.1") // only loopback trusted
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.10:4321" // private but NOT trusted
	req.Header.Set("X-Forwarded-For", "203.0.113.42")

	if ip := rl.clientIP(req); ip != "192.168.1.10" {
		t.Errorf("expected 192.168.1.10 (RemoteAddr), got %s", ip)
	}
}

// Public peer with spoofed XFF — must always use RemoteAddr.
func TestClientIPIgnoresXFFFromPublicPeer(t *testing.T) {
	rl := newTestLimiterWithProxies(t, 10, "127.0.0.1", "10.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:4321"
	req.Header.Set("X-Forwarded-For", "1.1.1.1")

	if ip := rl.clientIP(req); ip != "203.0.113.5" {
		t.Errorf("expected 203.0.113.5 (RemoteAddr), got %s", ip)
	}
}

// Direct request with no XFF — RemoteAddr used regardless of proxy config.
func TestClientIPDirectRequestNoXFF(t *testing.T) {
	rl := newTestLimiterWithProxies(t, 10, "127.0.0.1")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:4321"
	// No X-Forwarded-For header set.

	if ip := rl.clientIP(req); ip != "127.0.0.1" {
		t.Errorf("expected 127.0.0.1, got %s", ip)
	}
}

// ---------------------------------------------------------------------------
// Concurrency / race detection
// ---------------------------------------------------------------------------

func TestRateLimiterConcurrentRequestsNoRace(t *testing.T) {
	rl := newTestLimiter(t, 100)
	handler := rl.Limit(okHandler)

	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			makeRequest(handler, "6.6.6.6")
		}()
	}

	wg.Wait()
}

func TestRateLimiterConcurrentDifferentIPs(t *testing.T) {
	rl := newTestLimiter(t, 10)
	handler := rl.Limit(okHandler)

	ips := []string{
		"7.0.0.1", "7.0.0.2", "7.0.0.3", "7.0.0.4", "7.0.0.5",
	}

	var wg sync.WaitGroup

	for _, ip := range ips {
		for j := 0; j < 5; j++ {
			wg.Add(1)
			ip := ip
			go func() {
				defer wg.Done()
				makeRequest(handler, ip)
			}()
		}
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Cleanup goroutine stops on context cancellation
// ---------------------------------------------------------------------------

func TestRateLimiterCleanupStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	rl := NewRateLimiter(ctx, RateLimitConfig{RequestsPerMinute: 10})

	// Seed one entry so the map is non-empty.
	makeRequest(rl.Limit(okHandler), "8.8.8.8")

	// Cancel the context — the cleanup goroutine must exit.
	cancel()

	// Allow the goroutine scheduler time to process the cancellation.
	// We cannot observe the goroutine directly, but we can confirm the
	// limiter still functions correctly after cancellation (no panic, no
	// deadlock).
	makeRequest(rl.Limit(okHandler), "8.8.8.9")
}
