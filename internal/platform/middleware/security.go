package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

// SecurityHeaders adds security headers to responses
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")
			
			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")
			
			// Enable XSS filter
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			
			// HTTP Strict Transport Security
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			
			// Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'")
			
			// Referrer Policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			
			// Permissions Policy
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			
			// Cache control for sensitive endpoints
			if strings.HasPrefix(r.URL.Path, "/api/v1/auth") {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFProtection provides CSRF token validation
type CSRFConfig struct {
	TokenHeader string
	CookieName  string
	CookiePath  string
	Secure      bool
	SameSite    http.SameSite
	MaxAge      int
	ExemptPaths []string
}

func DefaultCSRFConfig() CSRFConfig {
	return CSRFConfig{
		TokenHeader: "X-CSRF-Token",
		CookieName:  "_csrf",
		CookiePath:  "/",
		Secure:      true,
		SameSite:    http.SameSiteStrictMode,
		MaxAge:      86400, // 24 hours
		ExemptPaths: []string{"/api/v1/auth/login", "/api/v1/auth/register", "/health"},
	}
}

func CSRFProtection(config CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF check for safe methods
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			
			// Check if path is exempt
			for _, path := range config.ExemptPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Get token from cookie
			cookie, err := r.Cookie(config.CookieName)
			if err != nil {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}
			
			// Get token from header
			headerToken := r.Header.Get(config.TokenHeader)
			if headerToken == "" {
				http.Error(w, "CSRF token missing from header", http.StatusForbidden)
				return
			}
			
			// Compare tokens using constant-time comparison
			if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
				http.Error(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimit limits the size of request bodies
func RequestSizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// SlowDown adds artificial delay after failed attempts (for timing attack prevention)
type SlowDownConfig struct {
	Threshold     int
	DelayPerRetry time.Duration
	MaxDelay      time.Duration
}

func SlowDown(config SlowDownConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This would typically use a counter from Redis or similar
			// For demonstration, we just pass through
			next.ServeHTTP(w, r)
		})
	}
}

// IPWhitelist restricts access to specific IPs
func IPWhitelist(allowedIPs []string) func(http.Handler) http.Handler {
	ipSet := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipSet[ip] = true
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			
			if !ipSet[clientIP] {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (when behind proxy/load balancer)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.LastIndex(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	
	return ip
}

// APIKeyAuth validates API key authentication
type APIKeyConfig struct {
	Header    string
	QueryParam string
	Validator func(key string) (bool, error)
}

func APIKeyAuth(config APIKeyConfig) func(http.Handler) http.Handler {
	if config.Header == "" {
		config.Header = "X-API-Key"
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var apiKey string
			
			// Check header first
			apiKey = r.Header.Get(config.Header)
			
			// Fall back to query parameter
			if apiKey == "" && config.QueryParam != "" {
				apiKey = r.URL.Query().Get(config.QueryParam)
			}
			
			if apiKey == "" {
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}
			
			if config.Validator != nil {
				valid, err := config.Validator(apiKey)
				if err != nil || !valid {
					http.Error(w, "Invalid API key", http.StatusUnauthorized)
					return
				}
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// SanitizeInputMiddleware sanitizes input to prevent XSS
func SanitizeInput() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Query parameters would be sanitized here
			// This is a placeholder for actual sanitization logic
			next.ServeHTTP(w, r)
		})
	}
}

// AuditLog logs all requests for audit trail
type AuditLogger interface {
	Log(entry AuditEntry) error
}

type AuditEntry struct {
	Timestamp  time.Time
	UserID     string
	Action     string
	Resource   string
	ResourceID string
	IPAddress  string
	UserAgent  string
	StatusCode int
	Duration   time.Duration
	Details    map[string]interface{}
}

func AuditLogging(logger AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			next.ServeHTTP(wrapped, r)
			
			entry := AuditEntry{
				Timestamp:  start,
				Action:     r.Method,
				Resource:   r.URL.Path,
				IPAddress:  getClientIP(r),
				UserAgent:  r.UserAgent(),
				StatusCode: wrapped.statusCode,
				Duration:   time.Since(start),
			}
			
			// Extract user ID from context if available
			if userID := r.Context().Value("userID"); userID != nil {
				entry.UserID = userID.(string)
			}
			
			if logger != nil {
				logger.Log(entry)
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
