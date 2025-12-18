// Package middleware provides HTTP middleware components
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ContextKey is used for context values
type ContextKey string

const (
	ContextUserID    ContextKey = "userID"
	ContextUserEmail ContextKey = "userEmail"
	ContextUserRoles ContextKey = "userRoles"
	ContextTenantID  ContextKey = "tenantID"
	ContextRequestID ContextKey = "requestID"
)

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"userId"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	TenantID string   `json:"tenantId"`
	jwt.RegisteredClaims
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret     []byte
	JWTIssuer     string
	SkipPaths     []string
	TokenHeader   string
	TokenPrefix   string
}

// DefaultAuthConfig returns default auth configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		TokenHeader: "Authorization",
		TokenPrefix: "Bearer ",
		SkipPaths:   []string{"/health", "/metrics", "/api/v1/auth/login", "/api/v1/auth/register"},
	}
}

// Auth creates JWT authentication middleware
func Auth(config *AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for certain paths
			for _, path := range config.SkipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from header
			authHeader := r.Header.Get(config.TokenHeader)
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, config.TokenPrefix) {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, config.TokenPrefix)

			// Parse and validate token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return config.JWTSecret, nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Add claims to context
			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextUserID, claims.UserID)
			ctx = context.WithValue(ctx, ContextUserEmail, claims.Email)
			ctx = context.WithValue(ctx, ContextUserRoles, claims.Roles)
			ctx = context.WithValue(ctx, ContextTenantID, claims.TenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates middleware that requires specific roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value(ContextUserRoles).([]string)
			if !ok {
				http.Error(w, `{"error":"no roles found"}`, http.StatusForbidden)
				return
			}

			hasRole := false
			for _, required := range roles {
				for _, userRole := range userRoles {
					if userRole == required || userRole == "admin" {
						hasRole = true
						break
					}
				}
			}

			if !hasRole {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// APIKey creates API key authentication middleware
func APIKey(validKeys map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				apiKey = r.URL.Query().Get("api_key")
			}

			if apiKey == "" {
				http.Error(w, `{"error":"missing API key"}`, http.StatusUnauthorized)
				return
			}

			userID, valid := validKeys[apiKey]
			if !valid {
				http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GenerateToken generates a JWT token
func GenerateToken(secret []byte, userID, email string, roles []string, tenantID string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Email:    email,
		Roles:    roles,
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(ContextUserID).(string); ok {
		return userID
	}
	return ""
}

// GetTenantID extracts tenant ID from context
func GetTenantID(ctx context.Context) string {
	if tenantID, ok := ctx.Value(ContextTenantID).(string); ok {
		return tenantID
	}
	return ""
}
