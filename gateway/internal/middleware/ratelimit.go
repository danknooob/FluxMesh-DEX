package middleware

import (
	"fmt"
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter is per-client token-bucket rate limiting middleware.
// After JWT middleware runs, the client key is the authenticated user ID.
// For unauthenticated routes the key falls back to the caller's IP.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a limiter allowing rps requests/sec with a burst buffer.
func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limiters[key]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[key] = l
	}
	return l
}

// Handler returns chi-compatible middleware.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := clientKey(r)
		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"error":"rate limit exceeded, try again shortly"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func clientKey(r *http.Request) string {
	if uid, ok := r.Context().Value(CtxUserID).(string); ok && uid != "" {
		return fmt.Sprintf("user:%s", uid)
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	return fmt.Sprintf("ip:%s", ip)
}
