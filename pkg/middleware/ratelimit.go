package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	BurstSize         int
	KeyFunc           func(r *http.Request) string
	SkipPaths         []string
	ExcludedIPs       []string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         200,
		KeyFunc: func(r *http.Request) string {
			return getClientIP(r)
		},
	}
}

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Remaining returns remaining tokens
func (tb *TokenBucket) Remaining() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return int(tb.tokens)
}

// RateLimiter manages rate limiting for multiple keys
type RateLimiter struct {
	buckets map[string]*TokenBucket
	config  *RateLimitConfig
	mu      sync.RWMutex
	cleanup time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		config:  config,
		cleanup: time.Now(),
	}
}

// getBucket gets or creates a token bucket for a key
func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Cleanup old buckets periodically
	if time.Since(rl.cleanup) > 10*time.Minute {
		for k := range rl.buckets {
			delete(rl.buckets, k)
		}
		rl.cleanup = time.Now()
	}

	bucket, ok := rl.buckets[key]
	if !ok {
		bucket = NewTokenBucket(
			float64(rl.config.BurstSize),
			float64(rl.config.RequestsPerMinute)/60.0,
		)
		rl.buckets[key] = bucket
	}
	return bucket
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	bucket := rl.getBucket(key)
	return bucket.Allow()
}

// Remaining returns remaining requests for a key
func (rl *RateLimiter) Remaining(key string) int {
	bucket := rl.getBucket(key)
	return bucket.Remaining()
}

// RateLimit creates rate limiting middleware
func RateLimit(config *RateLimitConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	limiter := NewRateLimiter(config)

	excludedIPs := make(map[string]bool)
	for _, ip := range config.ExcludedIPs {
		excludedIPs[ip] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for certain paths
			for _, path := range config.SkipPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get the key for rate limiting
			key := config.KeyFunc(r)

			// Skip excluded IPs
			if excludedIPs[key] {
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error": map[string]string{
						"code":    "RATE_LIMIT_EXCEEDED",
						"message": "Too many requests. Please try again later.",
					},
				})
				return
			}

			// Set rate limit headers
			remaining := limiter.Remaining(key)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}


