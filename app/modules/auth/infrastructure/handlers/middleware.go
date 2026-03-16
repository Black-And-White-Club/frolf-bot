package authhandlers

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// cleanupThreshold is the minimum map size before a cleanup pass runs.
	cleanupThreshold = 500
	// maxIdleAge is the duration after which an idle IP entry is eligible for cleanup.
	maxIdleAge = 10 * time.Minute
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter is an IP-based rate limiter that prunes stale entries inline.
type IPRateLimiter struct {
	ips map[string]*ipEntry
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter creates a new IPRateLimiter.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*ipEntry),
		r:   r,
		b:   b,
	}
}

// GetLimiter returns a rate.Limiter for the given IP, pruning stale entries when the
// map exceeds cleanupThreshold.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	if len(i.ips) > cleanupThreshold {
		cutoff := time.Now().Add(-maxIdleAge)
		for k, e := range i.ips {
			if e.lastSeen.Before(cutoff) {
				delete(i.ips, k)
			}
		}
	}

	e, exists := i.ips[ip]
	if !exists {
		e = &ipEntry{limiter: rate.NewLimiter(i.r, i.b)}
		i.ips[ip] = e
	}
	e.lastSeen = time.Now()

	return e.limiter
}

// RateLimitMiddleware returns a middleware that rate limits requests based on IP.
func RateLimitMiddleware(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			if !limiter.GetLimiter(ip).Allow() {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddleware returns a middleware that sets CORS headers for the configured origins.
// When allowedOrigins is empty, no CORS headers are added and the middleware is a no-op.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	origins := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		origins[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				if _, ok := origins[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				}
			}

			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Max-Age", "3600")
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidTokenFormat checks that a refresh token has the expected format:
// exactly 64 lowercase hex characters (hex-encoded 32 random bytes).
func isValidTokenFormat(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// AuthMiddleware ensures a valid session is present.
// It accepts either a refresh_token cookie (PWA flow) or an
// Authorization: Bearer header (Discord Activity flow).
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try cookie first (PWA flow).
		if cookie, err := r.Cookie("refresh_token"); err == nil && cookie.Value != "" {
			if isValidTokenFormat(cookie.Value) {
				next.ServeHTTP(w, r)
				return
			}
		}
		// Fallback: Authorization header (Activity flow).
		if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			token := auth[7:]
			if isValidTokenFormat(token) {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
