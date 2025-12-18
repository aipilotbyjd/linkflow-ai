package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// EndpointRateLimit defines rate limits for specific endpoints
type EndpointRateLimit struct {
	Path           string
	Method         string
	RequestsPerMin int
	BurstSize      int
}

// AdvancedRateLimiter provides per-endpoint rate limiting
type AdvancedRateLimiter struct {
	endpoints     map[string]*EndpointRateLimit
	defaultLimit  int
	defaultBurst  int
	clients       map[string]*clientLimits
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
}

type clientLimits struct {
	endpoints map[string]*tokenBucket
	mu        sync.Mutex
}

type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64
	lastUpdate time.Time
}

// NewAdvancedRateLimiter creates a new advanced rate limiter
func NewAdvancedRateLimiter(defaultLimit, defaultBurst int) *AdvancedRateLimiter {
	rl := &AdvancedRateLimiter{
		endpoints:     make(map[string]*EndpointRateLimit),
		defaultLimit:  defaultLimit,
		defaultBurst:  defaultBurst,
		clients:       make(map[string]*clientLimits),
		cleanupTicker: time.NewTicker(5 * time.Minute),
	}
	
	// Start cleanup goroutine
	go rl.cleanupLoop()
	
	return rl
}

// AddEndpointLimit adds a rate limit for a specific endpoint
func (rl *AdvancedRateLimiter) AddEndpointLimit(path, method string, requestsPerMin, burstSize int) {
	key := fmt.Sprintf("%s:%s", method, path)
	rl.endpoints[key] = &EndpointRateLimit{
		Path:           path,
		Method:         method,
		RequestsPerMin: requestsPerMin,
		BurstSize:      burstSize,
	}
}

// ConfigureDefaults sets up default endpoint limits
func (rl *AdvancedRateLimiter) ConfigureDefaults() {
	// Authentication endpoints (stricter limits)
	rl.AddEndpointLimit("/api/v1/auth/login", "POST", 10, 5)
	rl.AddEndpointLimit("/api/v1/auth/register", "POST", 5, 3)
	rl.AddEndpointLimit("/api/v1/auth/password/reset", "POST", 5, 3)
	
	// Execution endpoints (moderate limits)
	rl.AddEndpointLimit("/api/v1/workflows/*/execute", "POST", 30, 10)
	rl.AddEndpointLimit("/api/v1/executions", "POST", 60, 20)
	
	// Search endpoints (moderate limits)
	rl.AddEndpointLimit("/api/v1/search", "GET", 60, 30)
	rl.AddEndpointLimit("/api/v1/search", "POST", 60, 30)
	
	// Webhook endpoints (higher limits)
	rl.AddEndpointLimit("/api/v1/webhooks/*/trigger", "POST", 300, 100)
	
	// Admin endpoints (stricter limits)
	rl.AddEndpointLimit("/api/v1/admin/*", "GET", 30, 15)
	rl.AddEndpointLimit("/api/v1/admin/*", "POST", 20, 10)
	rl.AddEndpointLimit("/api/v1/admin/*", "DELETE", 10, 5)
	
	// File upload endpoints
	rl.AddEndpointLimit("/api/v1/storage/upload", "POST", 20, 10)
	
	// Analytics endpoints
	rl.AddEndpointLimit("/api/v1/analytics/events", "POST", 100, 50)
}

// Middleware returns the rate limiting middleware
func (rl *AdvancedRateLimiter) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			
			// Get limit for this endpoint
			limit := rl.getLimitForEndpoint(r.Method, r.URL.Path)
			
			// Check rate limit
			if !rl.allowRequest(clientIP, r.Method, r.URL.Path, limit) {
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit.RequestsPerMin))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			// Add rate limit headers
			remaining := rl.getRemainingTokens(clientIP, r.Method, r.URL.Path, limit)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit.RequestsPerMin))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", int(remaining)))
			
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *AdvancedRateLimiter) getLimitForEndpoint(method, path string) *EndpointRateLimit {
	// Try exact match first
	key := fmt.Sprintf("%s:%s", method, path)
	if limit, ok := rl.endpoints[key]; ok {
		return limit
	}
	
	// Try wildcard matches
	for k, limit := range rl.endpoints {
		if matchPath(k, fmt.Sprintf("%s:%s", method, path)) {
			return limit
		}
	}
	
	// Return default limit
	return &EndpointRateLimit{
		Path:           path,
		Method:         method,
		RequestsPerMin: rl.defaultLimit,
		BurstSize:      rl.defaultBurst,
	}
}

func (rl *AdvancedRateLimiter) allowRequest(clientIP, method, path string, limit *EndpointRateLimit) bool {
	rl.mu.Lock()
	client, ok := rl.clients[clientIP]
	if !ok {
		client = &clientLimits{
			endpoints: make(map[string]*tokenBucket),
		}
		rl.clients[clientIP] = client
	}
	rl.mu.Unlock()
	
	client.mu.Lock()
	defer client.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", method, path)
	bucket, ok := client.endpoints[key]
	if !ok {
		bucket = &tokenBucket{
			tokens:     float64(limit.BurstSize),
			maxTokens:  float64(limit.BurstSize),
			refillRate: float64(limit.RequestsPerMin) / 60.0, // tokens per second
			lastUpdate: time.Now(),
		}
		client.endpoints[key] = bucket
	}
	
	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens = min(bucket.maxTokens, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastUpdate = now
	
	// Check if we have tokens available
	if bucket.tokens < 1 {
		return false
	}
	
	// Consume a token
	bucket.tokens--
	return true
}

func (rl *AdvancedRateLimiter) getRemainingTokens(clientIP, method, path string, limit *EndpointRateLimit) float64 {
	rl.mu.RLock()
	client, ok := rl.clients[clientIP]
	rl.mu.RUnlock()
	
	if !ok {
		return float64(limit.BurstSize)
	}
	
	client.mu.Lock()
	defer client.mu.Unlock()
	
	key := fmt.Sprintf("%s:%s", method, path)
	bucket, ok := client.endpoints[key]
	if !ok {
		return float64(limit.BurstSize)
	}
	
	return bucket.tokens
}

func (rl *AdvancedRateLimiter) cleanupLoop() {
	for range rl.cleanupTicker.C {
		rl.cleanup()
	}
}

func (rl *AdvancedRateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, client := range rl.clients {
		client.mu.Lock()
		allExpired := true
		for _, bucket := range client.endpoints {
			if bucket.lastUpdate.After(cutoff) {
				allExpired = false
				break
			}
		}
		if allExpired {
			delete(rl.clients, ip)
		}
		client.mu.Unlock()
	}
}

func (rl *AdvancedRateLimiter) Stop() {
	rl.cleanupTicker.Stop()
}

func matchPath(pattern, path string) bool {
	// Simple wildcard matching
	if pattern == path {
		return true
	}
	
	// Handle * wildcards
	patternParts := splitPath(pattern)
	pathParts := splitPath(path)
	
	if len(patternParts) != len(pathParts) {
		return false
	}
	
	for i, pp := range patternParts {
		if pp != "*" && pp != pathParts[i] {
			return false
		}
	}
	
	return true
}

func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' || c == ':' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			for i, c := range xff {
				if c == ',' {
					return xff[:i]
				}
			}
			return xff
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	for i := len(ip) - 1; i >= 0; i-- {
		if ip[i] == ':' {
			return ip[:i]
		}
	}
	return ip
}

// UserRateLimiter provides per-user rate limiting
type UserRateLimiter struct {
	limits       map[string]int // tier -> requests per minute
	userTiers    map[string]string
	userLimiters map[string]*tokenBucket
	mu           sync.RWMutex
}

// NewUserRateLimiter creates a new user-based rate limiter
func NewUserRateLimiter() *UserRateLimiter {
	return &UserRateLimiter{
		limits: map[string]int{
			"free":       60,
			"starter":    300,
			"pro":        1000,
			"enterprise": 5000,
		},
		userTiers:    make(map[string]string),
		userLimiters: make(map[string]*tokenBucket),
	}
}

// SetUserTier sets the rate limit tier for a user
func (rl *UserRateLimiter) SetUserTier(userID, tier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.userTiers[userID] = tier
}

// AllowRequest checks if a user's request should be allowed
func (rl *UserRateLimiter) AllowRequest(ctx context.Context, userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	tier := rl.userTiers[userID]
	if tier == "" {
		tier = "free"
	}
	
	limit := rl.limits[tier]
	
	bucket, ok := rl.userLimiters[userID]
	if !ok {
		bucket = &tokenBucket{
			tokens:     float64(limit),
			maxTokens:  float64(limit),
			refillRate: float64(limit) / 60.0,
			lastUpdate: time.Now(),
		}
		rl.userLimiters[userID] = bucket
	}
	
	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens = min(bucket.maxTokens, bucket.tokens+elapsed*bucket.refillRate)
	bucket.lastUpdate = now
	
	if bucket.tokens < 1 {
		return false
	}
	
	bucket.tokens--
	return true
}
