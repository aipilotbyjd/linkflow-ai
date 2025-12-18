package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware provides JWT authentication
type AuthMiddleware struct {
	jwtSecret []byte
	skipPaths []string
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtSecret []byte) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
		skipPaths: []string{
			"/health/",
			"/metrics",
			"/api/v1/users/register",
			"/api/v1/users/login",
			"/auth/login",
			"/auth/register",
		},
	}
}

// Middleware returns the middleware handler
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths
		for _, path := range m.skipPaths {
			if strings.Contains(r.URL.Path, path) {
				next.ServeHTTP(w, r)
				return
			}
		}

		// Extract token from header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.respondUnauthorized(w, "missing authorization header")
			return
		}

		// Parse Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			m.respondUnauthorized(w, "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			m.respondUnauthorized(w, "invalid token")
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			m.respondUnauthorized(w, "invalid token claims")
			return
		}

		// Add user info to context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userID", claims["user_id"])
		ctx = context.WithValue(ctx, "email", claims["email"])
		ctx = context.WithValue(ctx, "roles", claims["roles"])
		
		// Add to headers for downstream services
		if userID, ok := claims["user_id"].(string); ok {
			r.Header.Set("X-User-ID", userID)
		}
		if email, ok := claims["email"].(string); ok {
			r.Header.Set("X-User-Email", email)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole creates a middleware that requires a specific role
func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			roles, ok := r.Context().Value("roles").([]interface{})
			if !ok {
				m.respondForbidden(w, "no roles found")
				return
			}

			hasRole := false
			for _, r := range roles {
				if roleStr, ok := r.(string); ok && roleStr == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				m.respondForbidden(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole creates a middleware that requires any of the specified roles
func (m *AuthMiddleware) RequireAnyRole(roles []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value("roles").([]interface{})
			if !ok {
				m.respondForbidden(w, "no roles found")
				return
			}

			hasRole := false
			for _, userRole := range userRoles {
				if userRoleStr, ok := userRole.(string); ok {
					for _, requiredRole := range roles {
						if userRoleStr == requiredRole {
							hasRole = true
							break
						}
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				m.respondForbidden(w, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ExtractUserID extracts the user ID from the context
func ExtractUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("userID").(string)
	return userID, ok
}

// ExtractEmail extracts the email from the context
func ExtractEmail(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("email").(string)
	return email, ok
}

// ExtractRoles extracts roles from the context
func ExtractRoles(ctx context.Context) ([]string, bool) {
	rolesInterface, ok := ctx.Value("roles").([]interface{})
	if !ok {
		return nil, false
	}

	roles := make([]string, 0, len(rolesInterface))
	for _, r := range rolesInterface {
		if roleStr, ok := r.(string); ok {
			roles = append(roles, roleStr)
		}
	}
	return roles, true
}

func (m *AuthMiddleware) respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

func (m *AuthMiddleware) respondForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
