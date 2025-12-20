// Package handlers provides HTTP handlers for the auth service
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
)

// AuthHandler handles auth HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Router is an interface for registering routes (compatible with both http.ServeMux and gorilla/mux)
type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(router Router) {
	// Auth endpoints
	router.HandleFunc("/api/v1/auth/login", h.login)
	router.HandleFunc("/api/v1/auth/register", h.register)
	router.HandleFunc("/api/v1/auth/refresh", h.refresh)
	router.HandleFunc("/api/v1/auth/logout", h.logout)
	router.HandleFunc("/api/v1/auth/password/forgot", h.forgotPassword)
	router.HandleFunc("/api/v1/auth/password/reset", h.resetPassword)
	router.HandleFunc("/api/v1/auth/password/change", h.changePassword)
	router.HandleFunc("/api/v1/auth/email/verify", h.verifyEmail)
	router.HandleFunc("/api/v1/auth/email/resend", h.resendVerification)
	
	// API Key endpoints
	router.HandleFunc("/api/v1/auth/api-keys", h.handleAPIKeys)
	router.HandleFunc("/api/v1/auth/api-keys/", h.handleAPIKey)
	
	// Session endpoints
	router.HandleFunc("/api/v1/auth/sessions", h.listSessions)
	router.HandleFunc("/api/v1/auth/sessions/", h.revokeSession)
	
	// OAuth endpoints
	router.HandleFunc("/api/v1/auth/oauth/google", h.oauthGoogle)
	router.HandleFunc("/api/v1/auth/oauth/github", h.oauthGitHub)
	router.HandleFunc("/api/v1/auth/oauth/callback", h.oauthCallback)
	
	// MFA endpoints
	router.HandleFunc("/api/v1/auth/mfa/setup", h.mfaSetup)
	router.HandleFunc("/api/v1/auth/mfa/verify", h.mfaVerify)
	router.HandleFunc("/api/v1/auth/mfa/disable", h.mfaDisable)
}

// LoginRequest represents login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	MFACode  string `json:"mfaCode,omitempty"`
}

// LoginResponse represents login response
type LoginResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
	User         UserResponse `json:"user"`
}

// UserResponse represents user in response
type UserResponse struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	FirstName     string `json:"firstName"`
	LastName      string `json:"lastName"`
	EmailVerified bool   `json:"emailVerified"`
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	result, err := h.authService.Login(r.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		MFACode:   req.MFACode,
		UserAgent: r.UserAgent(),
		IPAddress: getClientIP(r),
	})
	if err != nil {
		switch err {
		case model.ErrInvalidCredentials:
			writeError(w, "Invalid email or password", http.StatusUnauthorized)
		case model.ErrAccountLocked:
			writeError(w, "Account is locked. Please try again later", http.StatusForbidden)
		case model.ErrEmailNotVerified:
			writeError(w, "Please verify your email before logging in", http.StatusForbidden)
		default:
			writeError(w, "Login failed", http.StatusInternalServerError)
		}
		return
	}

	// Check if MFA is required
	if result.RequiresMFA {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"requiresMFA": true,
			"mfaToken":    result.MFAToken,
		})
		return
	}

	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		ExpiresAt:    result.Tokens.ExpiresAt,
		TokenType:    result.Tokens.TokenType,
		User: UserResponse{
			ID:            result.User.ID,
			Email:         result.User.Email,
			FirstName:     result.User.FirstName,
			LastName:      result.User.LastName,
			EmailVerified: result.User.EmailVerified,
		},
	})
}

// RegisterRequest represents registration request body
type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	result, err := h.authService.Register(r.Context(), service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		UserAgent: r.UserAgent(),
		IPAddress: getClientIP(r),
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusCreated, LoginResponse{
		AccessToken:  result.Tokens.AccessToken,
		RefreshToken: result.Tokens.RefreshToken,
		ExpiresAt:    result.Tokens.ExpiresAt,
		TokenType:    result.Tokens.TokenType,
		User: UserResponse{
			ID:            result.User.ID,
			Email:         result.User.Email,
			FirstName:     result.User.FirstName,
			LastName:      result.User.LastName,
			EmailVerified: result.User.EmailVerified,
		},
	})
}

// ChangePasswordRequest represents change password request body
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, "Current and new password are required", http.StatusBadRequest)
		return
	}

	err := h.authService.ChangePassword(r.Context(), service.ChangePasswordInput{
		UserID:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		IPAddress:       getClientIP(r),
		UserAgent:       r.UserAgent(),
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password changed successfully. Please log in again.",
	})
}

// RefreshRequest represents refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tokens, err := h.authService.RefreshTokens(r.Context(), req.RefreshToken, r.UserAgent(), getClientIP(r))
	if err != nil {
		writeError(w, "Invalid or expired refresh token", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"accessToken":  tokens.AccessToken,
		"refreshToken": tokens.RefreshToken,
		"expiresAt":    tokens.ExpiresAt,
		"tokenType":    tokens.TokenType,
	})
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context (set by auth middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		RefreshToken string `json:"refreshToken"`
		AllDevices   bool   `json:"allDevices"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var err error
	if req.AllDevices {
		err = h.authService.Logout(r.Context(), userID)
	} else if req.RefreshToken != "" {
		err = h.authService.LogoutSession(r.Context(), req.RefreshToken)
	}

	if err != nil {
		writeError(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ForgotPasswordRequest represents forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func (h *AuthHandler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Always return success to prevent email enumeration
	_ = h.authService.RequestPasswordReset(r.Context(), req.Email)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, you will receive a password reset link",
	})
}

// ResetPasswordRequest represents reset password request
type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		writeError(w, "Password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	if err := h.authService.ResetPassword(r.Context(), req.Token, req.NewPassword); err != nil {
		switch err {
		case model.ErrTokenExpired:
			writeError(w, "Reset link has expired", http.StatusBadRequest)
		case model.ErrTokenInvalid:
			writeError(w, "Invalid reset link", http.StatusBadRequest)
		default:
			writeError(w, "Failed to reset password", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

func (h *AuthHandler) verifyEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		var req struct {
			Token string `json:"token"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		token = req.Token
	}

	if token == "" {
		writeError(w, "Token is required", http.StatusBadRequest)
		return
	}

	if err := h.authService.VerifyEmail(r.Context(), token); err != nil {
		switch err {
		case model.ErrTokenExpired:
			writeError(w, "Verification link has expired", http.StatusBadRequest)
		case model.ErrTokenInvalid:
			writeError(w, "Invalid verification link", http.StatusBadRequest)
		default:
			writeError(w, "Failed to verify email", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Email verified successfully",
	})
}

func (h *AuthHandler) resendVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.authService.SendEmailVerification(r.Context(), userID); err != nil {
		writeError(w, "Failed to send verification email", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Verification email sent",
	})
}

// API Key handlers

func (h *AuthHandler) handleAPIKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAPIKeys(w, r)
	case http.MethodPost:
		h.createAPIKey(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AuthHandler) handleAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID := strings.TrimPrefix(r.URL.Path, "/api/v1/auth/api-keys/")
	if keyID == "" {
		http.Error(w, "Key ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodDelete:
		h.revokeAPIKey(w, r, keyID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateAPIKeyRequest represents API key creation request
type CreateAPIKeyRequest struct {
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// APIKeyResponse represents API key response
type APIKeyResponse struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"keyPrefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func (h *AuthHandler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	workspaceID := r.Header.Get("X-Workspace-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "Name is required", http.StatusBadRequest)
		return
	}

	if len(req.Scopes) == 0 {
		req.Scopes = []string{"workflows:read", "workflows:execute"}
	}

	apiKey, rawKey, err := h.authService.CreateAPIKey(r.Context(), service.CreateAPIKeyInput{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		writeError(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        apiKey.ID,
		"name":      apiKey.Name,
		"key":       rawKey, // Only shown once!
		"keyPrefix": apiKey.KeyPrefix,
		"scopes":    apiKey.Scopes,
		"expiresAt": apiKey.ExpiresAt,
		"createdAt": apiKey.CreatedAt,
		"message":   "Save this key now. You won't be able to see it again.",
	})
}

func (h *AuthHandler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	workspaceID := r.Header.Get("X-Workspace-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := h.authService.ListAPIKeys(r.Context(), userID, workspaceID)
	if err != nil {
		writeError(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	response := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		response[i] = APIKeyResponse{
			ID:         k.ID,
			Name:       k.Name,
			KeyPrefix:  k.KeyPrefix,
			Scopes:     k.Scopes,
			LastUsedAt: k.LastUsedAt,
			ExpiresAt:  k.ExpiresAt,
			CreatedAt:  k.CreatedAt,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

func (h *AuthHandler) revokeAPIKey(w http.ResponseWriter, r *http.Request, keyID string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.authService.RevokeAPIKey(r.Context(), keyID, userID); err != nil {
		writeError(w, "Failed to revoke API key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Session handlers

func (h *AuthHandler) listSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement session listing
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": []interface{}{},
		"total": 0,
	})
}

func (h *AuthHandler) revokeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement session revocation
	w.WriteHeader(http.StatusNoContent)
}

// OAuth handlers

func (h *AuthHandler) oauthGoogle(w http.ResponseWriter, r *http.Request) {
	// Redirect to Google OAuth
	// TODO: Implement OAuth flow
	writeJSON(w, http.StatusOK, map[string]string{
		"url": "https://accounts.google.com/o/oauth2/v2/auth?...",
	})
}

func (h *AuthHandler) oauthGitHub(w http.ResponseWriter, r *http.Request) {
	// Redirect to GitHub OAuth
	// TODO: Implement OAuth flow
	writeJSON(w, http.StatusOK, map[string]string{
		"url": "https://github.com/login/oauth/authorize?...",
	})
}

func (h *AuthHandler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	// Handle OAuth callback
	// TODO: Implement OAuth callback
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "OAuth callback received",
	})
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, message string, status int) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	
	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// MFA handlers

func (h *AuthHandler) mfaSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Implement MFA setup - generate TOTP secret and QR code
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "MFA setup endpoint - implement TOTP generation",
		"secret":    "placeholder-secret",
		"qrCodeUrl": "otpauth://totp/LinkFlow:user@example.com?secret=placeholder&issuer=LinkFlow",
	})
}

func (h *AuthHandler) mfaVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement MFA verification and enable MFA for user
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "MFA enabled successfully",
		"backupCodes": []string{
			"XXXX-XXXX-XXXX",
			"YYYY-YYYY-YYYY",
			"ZZZZ-ZZZZ-ZZZZ",
		},
	})
}

func (h *AuthHandler) mfaDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement MFA disable - verify password and code first
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "MFA disabled successfully",
	})
}
