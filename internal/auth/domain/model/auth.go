// Package model defines auth domain models
package model

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TokenType represents the type of token
type TokenType string

const (
	TokenTypeAccess       TokenType = "access"
	TokenTypeRefresh      TokenType = "refresh"
	TokenTypePasswordReset TokenType = "password_reset"
	TokenTypeEmailVerify  TokenType = "email_verify"
	TokenTypeAPIKey       TokenType = "api_key"
)

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
	UserAgent string
	IPAddress string
}

// NewRefreshToken creates a new refresh token
func NewRefreshToken(userID, userAgent, ipAddress string, expiry time.Duration) (*RefreshToken, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	return &RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(expiry),
		CreatedAt: time.Now(),
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}, nil
}

// IsValid checks if the token is valid
func (t *RefreshToken) IsValid() bool {
	return t.RevokedAt == nil && time.Now().Before(t.ExpiresAt)
}

// Revoke revokes the token
func (t *RefreshToken) Revoke() {
	now := time.Now()
	t.RevokedAt = &now
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// NewPasswordResetToken creates a new password reset token
func NewPasswordResetToken(userID string) (*PasswordResetToken, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	return &PasswordResetToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

// IsValid checks if the token is valid
func (t *PasswordResetToken) IsValid() bool {
	return t.UsedAt == nil && time.Now().Before(t.ExpiresAt)
}

// MarkUsed marks the token as used
func (t *PasswordResetToken) MarkUsed() {
	now := time.Now()
	t.UsedAt = &now
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        string
	UserID    string
	Email     string
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// NewEmailVerificationToken creates a new email verification token
func NewEmailVerificationToken(userID, email string) (*EmailVerificationToken, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	return &EmailVerificationToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		Email:     email,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}, nil
}

// IsValid checks if the token is valid
func (t *EmailVerificationToken) IsValid() bool {
	return t.UsedAt == nil && time.Now().Before(t.ExpiresAt)
}

// MarkUsed marks the token as used
func (t *EmailVerificationToken) MarkUsed() {
	now := time.Now()
	t.UsedAt = &now
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string
	UserID      string
	WorkspaceID string
	Name        string
	KeyHash     string
	KeyPrefix   string // First 8 chars for identification
	Scopes      []string
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	RevokedAt   *time.Time
}

// NewAPIKey creates a new API key
func NewAPIKey(userID, workspaceID, name string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	// Generate a secure API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}
	
	rawKey := "lf_" + base64.RawURLEncoding.EncodeToString(keyBytes)
	
	// Hash the key for storage
	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	return &APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		WorkspaceID: workspaceID,
		Name:        name,
		KeyHash:     string(hash),
		KeyPrefix:   rawKey[:11], // "lf_" + first 8 chars
		Scopes:      scopes,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}, rawKey, nil
}

// Verify verifies the API key
func (k *APIKey) Verify(rawKey string) bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)) == nil
}

// Revoke revokes the API key
func (k *APIKey) Revoke() {
	now := time.Now()
	k.RevokedAt = &now
}

// MarkUsed updates last used timestamp
func (k *APIKey) MarkUsed() {
	now := time.Now()
	k.LastUsedAt = &now
}

// HasScope checks if the key has a specific scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "*" {
			return true
		}
	}
	return false
}

// OAuthProvider represents an OAuth provider
type OAuthProvider string

const (
	OAuthProviderGoogle OAuthProvider = "google"
	OAuthProviderGitHub OAuthProvider = "github"
	OAuthProviderMicrosoft OAuthProvider = "microsoft"
)

// OAuthConnection represents a user's OAuth connection
type OAuthConnection struct {
	ID           string
	UserID       string
	Provider     OAuthProvider
	ProviderID   string
	Email        string
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	Metadata     map[string]interface{}
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewOAuthConnection creates a new OAuth connection
func NewOAuthConnection(userID string, provider OAuthProvider, providerID, email, accessToken, refreshToken string, expiresAt *time.Time) *OAuthConnection {
	return &OAuthConnection{
		ID:           uuid.New().String(),
		UserID:       userID,
		Provider:     provider,
		ProviderID:   providerID,
		Email:        email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		Metadata:     make(map[string]interface{}),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// Session represents a user session
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	UserAgent string
	IPAddress string
	CreatedAt time.Time
	LastSeenAt time.Time
}

// NewSession creates a new session
func NewSession(userID, userAgent, ipAddress string, expiry time.Duration) (*Session, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Session{
		ID:         uuid.New().String(),
		UserID:     userID,
		Token:      token,
		ExpiresAt:  now.Add(expiry),
		UserAgent:  userAgent,
		IPAddress:  ipAddress,
		CreatedAt:  now,
		LastSeenAt: now,
	}, nil
}

// IsValid checks if the session is valid
func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// Touch updates last seen time
func (s *Session) Touch() {
	s.LastSeenAt = time.Now()
}

// LoginAttempt tracks login attempts for rate limiting
type LoginAttempt struct {
	ID        string
	Email     string
	IPAddress string
	Success   bool
	Reason    string
	CreatedAt time.Time
}

// Errors
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account is locked")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenRevoked       = errors.New("token has been revoked")
)

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
