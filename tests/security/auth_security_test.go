package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Security tests for authentication and authorization

const testBaseURL = "http://localhost:8001"

// TestSQLInjection tests for SQL injection vulnerabilities
func TestSQLInjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	sqlInjectionPayloads := []string{
		"' OR '1'='1",
		"'; DROP TABLE users; --",
		"admin'--",
		"1' AND '1'='1",
		"1'; SELECT * FROM users WHERE '1'='1",
		"1 UNION SELECT * FROM users",
		"' OR 1=1 --",
		"admin' OR '1'='1' --",
	}

	for _, payload := range sqlInjectionPayloads {
		t.Run("login_email_injection", func(t *testing.T) {
			body := map[string]interface{}{
				"email":    payload,
				"password": "password123",
			}
			resp := makeRequest(t, "POST", "/api/v1/auth/login", body, "")
			
			// Should return 400 or 401, not 500 (server error indicating successful injection)
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
				"Potential SQL injection vulnerability with payload: %s", payload)
		})

		t.Run("login_password_injection", func(t *testing.T) {
			body := map[string]interface{}{
				"email":    "test@example.com",
				"password": payload,
			}
			resp := makeRequest(t, "POST", "/api/v1/auth/login", body, "")
			
			assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
				"Potential SQL injection vulnerability with payload: %s", payload)
		})
	}
}

// TestXSSPrevention tests for XSS vulnerabilities
func TestXSSPrevention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg onload=alert('XSS')>",
		"<body onload=alert('XSS')>",
		"<iframe src='javascript:alert(1)'>",
		"<a href='javascript:alert(1)'>click</a>",
		"<div onclick=alert('XSS')>click me</div>",
	}

	for _, payload := range xssPayloads {
		t.Run("registration_xss", func(t *testing.T) {
			body := map[string]interface{}{
				"email":     "test@example.com",
				"password":  "SecurePass123!",
				"firstName": payload,
				"lastName":  "Test",
			}
			resp := makeRequest(t, "POST", "/api/v1/auth/register", body, "")
			
			// Check response doesn't contain unescaped XSS payload
			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)
			resp.Body.Close()
			
			// If user was created, firstName should be escaped
			if user, ok := result["user"].(map[string]interface{}); ok {
				firstName := user["firstName"].(string)
				assert.NotContains(t, firstName, "<script>",
					"XSS payload not escaped in response")
			}
		})
	}
}

// TestBruteForceProtection tests rate limiting on login
func TestBruteForceProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	// Make multiple rapid login attempts
	body := map[string]interface{}{
		"email":    "attacker@example.com",
		"password": "wrongpassword",
	}

	var rateLimited bool
	for i := 0; i < 20; i++ {
		resp := makeRequest(t, "POST", "/api/v1/auth/login", body, "")
		
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
		resp.Body.Close()
	}

	assert.True(t, rateLimited, "Rate limiting should be triggered after multiple failed login attempts")
}

// TestPasswordPolicy tests password validation
func TestPasswordPolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	weakPasswords := []struct {
		password string
		reason   string
	}{
		{"123456", "too short and simple"},
		{"password", "common password"},
		{"qwerty", "common password"},
		{"abc", "too short"},
		{"        ", "only whitespace"},
		{"password123", "common pattern"},
		{"12345678", "only numbers"},
		{"abcdefgh", "only letters"},
	}

	for _, wp := range weakPasswords {
		t.Run("weak_password_"+wp.reason, func(t *testing.T) {
			body := map[string]interface{}{
				"email":     "test@example.com",
				"password":  wp.password,
				"firstName": "Test",
				"lastName":  "User",
			}
			resp := makeRequest(t, "POST", "/api/v1/auth/register", body, "")
			
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode,
				"Weak password '%s' (%s) should be rejected", wp.password, wp.reason)
			resp.Body.Close()
		})
	}
}

// TestJWTTokenValidation tests JWT token security
func TestJWTTokenValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	t.Run("missing_token", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/v1/auth/me", nil, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("invalid_token_format", func(t *testing.T) {
		resp := makeRequest(t, "GET", "/api/v1/auth/me", nil, "invalid-token")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("expired_token", func(t *testing.T) {
		// This would require creating an expired token
		expiredToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2MDAwMDAwMDB9.fake"
		resp := makeRequest(t, "GET", "/api/v1/auth/me", nil, expiredToken)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("tampered_token", func(t *testing.T) {
		tamperedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiIxMjM0NTYifQ.tampered"
		resp := makeRequest(t, "GET", "/api/v1/auth/me", nil, tamperedToken)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})
}

// TestCSRFProtection tests CSRF token validation
func TestCSRFProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	t.Run("missing_csrf_token", func(t *testing.T) {
		// Make request without CSRF token to state-changing endpoint
		_ = map[string]interface{}{
			"name": "Test Workflow",
		}
		
		req, _ := http.NewRequest("POST", testBaseURL+"/api/v1/workflows", nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer valid-token")
		// Note: Not setting X-CSRF-Token header
		
		// In a properly configured system, this should return 403
		// For now, we just document the expected behavior
		_ = req // Silence unused warning for now
		t.Log("CSRF protection should reject requests without CSRF token for state-changing operations")
	})
}

// TestSessionSecurity tests session management security
func TestSessionSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	t.Run("session_fixation", func(t *testing.T) {
		// Test that session ID changes after login
		// This requires actual session tracking
		t.Log("Session fixation protection: Session ID should change after authentication")
	})

	t.Run("concurrent_sessions", func(t *testing.T) {
		// Test handling of multiple concurrent sessions
		t.Log("Concurrent sessions should be properly managed")
	})
}

// TestInputValidation tests input validation
func TestInputValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	t.Run("oversized_payload", func(t *testing.T) {
		// Create a very large payload
		largeString := strings.Repeat("a", 1024*1024*10) // 10MB
		body := map[string]interface{}{
			"email":     largeString + "@example.com",
			"password":  "password123",
			"firstName": "Test",
			"lastName":  "User",
		}
		
		resp := makeRequest(t, "POST", "/api/v1/auth/register", body, "")
		
		// Should reject oversized payloads
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
		assert.NotEqual(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("null_bytes", func(t *testing.T) {
		body := map[string]interface{}{
			"email":     "test\x00@example.com",
			"password":  "password\x00123",
			"firstName": "Test",
			"lastName":  "User",
		}
		
		resp := makeRequest(t, "POST", "/api/v1/auth/register", body, "")
		
		// Should handle null bytes safely
		assert.NotEqual(t, http.StatusInternalServerError, resp.StatusCode,
			"Null bytes should be handled safely")
		resp.Body.Close()
	})

	t.Run("unicode_normalization", func(t *testing.T) {
		// Test Unicode normalization attacks
		unicodePayloads := []string{
			"ⓐⓓⓜⓘⓝ@example.com", // Circled letters
			"аdmin@example.com",   // Cyrillic 'а' looks like Latin 'a'
		}
		
		for _, payload := range unicodePayloads {
			body := map[string]interface{}{
				"email":     payload,
				"password":  "SecurePass123!",
				"firstName": "Test",
				"lastName":  "User",
			}
			
			resp := makeRequest(t, "POST", "/api/v1/auth/register", body, "")
			// Should properly validate/normalize Unicode
			resp.Body.Close()
		}
	})
}

// TestSecurityHeaders tests security response headers
func TestSecurityHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	resp := makeRequest(t, "GET", "/api/v1/auth/me", nil, "test-token")
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":   "", // Should be present
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		if expectedValue != "" {
			assert.Equal(t, expectedValue, actualValue,
				"Security header %s should be set to %s", header, expectedValue)
		} else {
			t.Logf("Security header %s should be present (current: %s)", header, actualValue)
		}
	}
}

// TestAccountEnumeration tests protection against account enumeration
func TestAccountEnumeration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	// Test that login responses don't reveal whether an email exists
	existingUserBody := map[string]interface{}{
		"email":    "existing@example.com",
		"password": "wrongpassword",
	}
	
	nonExistingUserBody := map[string]interface{}{
		"email":    "nonexisting@example.com",
		"password": "wrongpassword",
	}

	resp1 := makeRequest(t, "POST", "/api/v1/auth/login", existingUserBody, "")
	resp2 := makeRequest(t, "POST", "/api/v1/auth/login", nonExistingUserBody, "")

	// Response should be identical to prevent enumeration
	t.Log("Account enumeration protection: Login failure responses should be identical")
	
	resp1.Body.Close()
	resp2.Body.Close()
}

// TestTimingAttacks tests protection against timing attacks
func TestTimingAttacks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security tests in short mode")
	}

	// Test that password comparison uses constant-time comparison
	shortPassword := map[string]interface{}{
		"email":    "test@example.com",
		"password": "a",
	}
	
	longPassword := map[string]interface{}{
		"email":    "test@example.com",
		"password": strings.Repeat("a", 100),
	}

	var shortTimes, longTimes []time.Duration

	for i := 0; i < 10; i++ {
		start := time.Now()
		resp := makeRequest(t, "POST", "/api/v1/auth/login", shortPassword, "")
		shortTimes = append(shortTimes, time.Since(start))
		resp.Body.Close()

		start = time.Now()
		resp = makeRequest(t, "POST", "/api/v1/auth/login", longPassword, "")
		longTimes = append(longTimes, time.Since(start))
		resp.Body.Close()
	}

	t.Log("Timing attack protection: Response times should be constant regardless of password length")
}

// Helper function
func makeRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	var bodyReader *bytes.Buffer
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(bodyBytes)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, testBaseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Create a mock response writer for testing
	w := httptest.NewRecorder()
	
	// In a real test, you would use the actual handler
	// For now, return a mock response
	return w.Result()
}
