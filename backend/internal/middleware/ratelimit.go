package middleware

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimitConfig defines the token-bucket parameters for one limiter.
//
// Token-bucket behaviour: each client IP starts with a full bucket of
// RequestsPerMinute tokens. Tokens refill continuously at
// RequestsPerMinute/60 per second, so short bursts up to the full
// capacity are permitted before the sustained rate is enforced.
//
// Limits are tracked in memory and are therefore per application instance.
// In a horizontally-scaled deployment each instance enforces its own limit
// independently; a shared store (e.g. Redis) would be required for a
// global limit across all instances.
type RateLimitConfig struct {
	// RequestsPerMinute is the sustained request rate allowed per client IP.
	RequestsPerMinute int

	// TrustedProxies is the explicit list of reverse-proxy IPs or CIDRs
	// (e.g. "127.0.0.1", "10.0.0.0/8") whose X-Forwarded-For header is
	// trusted to carry the real client IP.
	//
	// Only requests whose RemoteAddr matches one of these entries will have
	// their X-Forwarded-For header consulted. All other requests use
	// RemoteAddr directly, preventing header-spoofing attacks.
	//
	// Leave nil or empty to always use RemoteAddr (safe default).
	TrustedProxies []string
}

// bucket holds the token-bucket state for a single client IP.
type bucket struct {
	tokens    float64
	lastRefil time.Time
}

// RateLimiter is a concurrency-safe, per-IP token-bucket rate limiter.
type RateLimiter struct {
	mu             sync.Mutex
	clients        map[string]*bucket
	rate           float64 // tokens added per second
	capacity       float64 // maximum tokens (burst = capacity)
	trustedProxies []*net.IPNet
}

// NewRateLimiter creates a RateLimiter and starts a background cleanup goroutine
// that removes stale entries. The goroutine stops when ctx is cancelled.
func NewRateLimiter(ctx context.Context, cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		clients:  make(map[string]*bucket),
		rate:     float64(cfg.RequestsPerMinute) / 60.0,
		capacity: float64(cfg.RequestsPerMinute),
	}

	for _, entry := range cfg.TrustedProxies {
		// Accept both bare IPs ("127.0.0.1") and CIDR notation ("10.0.0.0/8").
		if _, network, err := net.ParseCIDR(entry); err == nil {
			rl.trustedProxies = append(rl.trustedProxies, network)
		} else if ip := net.ParseIP(entry); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			rl.trustedProxies = append(rl.trustedProxies, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
		}
	}

	go rl.cleanupLoop(ctx)

	return rl
}

// allow returns true when the request from ip is within the rate limit.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	b, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &bucket{
			tokens:    rl.capacity - 1,
			lastRefil: now,
		}
		return true
	}

	// Refill tokens proportional to elapsed time.
	elapsed := now.Sub(b.lastRefil).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastRefil = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// cleanupLoop removes entries that have been idle for more than two window
// periods, preventing unbounded memory growth.
func (rl *RateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-2 * time.Minute)
			for ip, b := range rl.clients {
				if b.lastRefil.Before(cutoff) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Limit returns middleware that enforces this rate limiter.
// On rejection it writes HTTP 429 with a JSON body and a Retry-After header.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.clientIP(r)

		if !rl.allow(ip) {
			retryAfter := int(60.0 / rl.rate)

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			w.WriteHeader(http.StatusTooManyRequests)

			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "rate limit exceeded, please try again later",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the real client IP from the request.
//
// Strategy:
//  1. Always parse RemoteAddr as the authoritative peer address.
//  2. Consult X-Forwarded-For only when RemoteAddr matches one of the
//     explicitly configured TrustedProxies. Only the leftmost (original
//     client) value is used; subsequent hops may be attacker-controlled.
//  3. If no trusted proxies are configured, RemoteAddr is always used.
//
// This prevents IP spoofing: an external client cannot inject an arbitrary
// X-Forwarded-For header unless they are connecting through a proxy that
// the operator has explicitly trusted.
func (rl *RateLimiter) clientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if len(rl.trustedProxies) > 0 && rl.isTrustedProxy(remoteHost) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take only the leftmost address — the original client IP.
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					xff = xff[:i]
					break
				}
			}
			if candidate := trimSpace(xff); candidate != "" {
				return candidate
			}
		}
	}

	return remoteHost
}

// isTrustedProxy reports whether the given IP string matches any entry in
// the configured trusted proxy list.
func (rl *RateLimiter) isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range rl.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// trimSpace removes leading and trailing ASCII spaces and tabs.
func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
