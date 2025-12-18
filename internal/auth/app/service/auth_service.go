// Package service provides auth business logic
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
)

// Config holds auth service configuration
type Config struct {
	JWTSecret           string
	JWTIssuer           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	PasswordResetExpiry time.Duration
	MaxLoginAttempts    int
	LockoutDuration     time.Duration
}

// DefaultConfig returns default auth configuration
func DefaultConfig() Config {
	return Config{
		JWTSecret:           "change-me-in-production",
		JWTIssuer:           "linkflow-ai",
		AccessTokenExpiry:   15 * time.Minute,
		RefreshTokenExpiry:  7 * 24 * time.Hour,
		PasswordResetExpiry: 1 * time.Hour,
		MaxLoginAttempts:    5,
		LockoutDuration:     15 * time.Minute,
	}
}

// UserRepository defines user persistence operations
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	VerifyEmail(ctx context.Context, userID string) error
}

// TokenRepository defines token persistence operations
type TokenRepository interface {
	SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error
	FindRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, token string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	
	SavePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error
	FindPasswordResetToken(ctx context.Context, token string) (*model.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, token string) error
	
	SaveEmailVerificationToken(ctx context.Context, token *model.EmailVerificationToken) error
	FindEmailVerificationToken(ctx context.Context, token string) (*model.EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, token string) error
}

// APIKeyRepository defines API key persistence operations
type APIKeyRepository interface {
	Save(ctx context.Context, key *model.APIKey) error
	FindByID(ctx context.Context, id string) (*model.APIKey, error)
	FindByPrefix(ctx context.Context, prefix string) (*model.APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]*model.APIKey, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.APIKey, error)
	Revoke(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string) error
}

// OAuthRepository defines OAuth persistence operations
type OAuthRepository interface {
	Save(ctx context.Context, conn *model.OAuthConnection) error
	FindByProviderID(ctx context.Context, provider model.OAuthProvider, providerID string) (*model.OAuthConnection, error)
	FindByUser(ctx context.Context, userID string) ([]*model.OAuthConnection, error)
	Delete(ctx context.Context, id string) error
}

// EmailService defines email sending operations
type EmailService interface {
	SendPasswordReset(ctx context.Context, email, token, name string) error
	SendEmailVerification(ctx context.Context, email, token, name string) error
	SendWelcome(ctx context.Context, email, name string) error
}

// User represents a user for auth purposes
type User struct {
	ID            string
	Email         string
	PasswordHash  string
	FirstName     string
	LastName      string
	EmailVerified bool
	Status        string
	FailedAttempts int
	LockedUntil   *time.Time
	CreatedAt     time.Time
}

// AuthService handles authentication and authorization
type AuthService struct {
	config      Config
	userRepo    UserRepository
	tokenRepo   TokenRepository
	apiKeyRepo  APIKeyRepository
	oauthRepo   OAuthRepository
	emailSvc    EmailService
}

// NewAuthService creates a new auth service
func NewAuthService(
	config Config,
	userRepo UserRepository,
	tokenRepo TokenRepository,
	apiKeyRepo APIKeyRepository,
	oauthRepo OAuthRepository,
	emailSvc EmailService,
) *AuthService {
	return &AuthService{
		config:     config,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		apiKeyRepo: apiKeyRepo,
		oauthRepo:  oauthRepo,
		emailSvc:   emailSvc,
	}
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	TokenType    string    `json:"tokenType"`
}

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	WorkspaceID string   `json:"wid,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Scopes      []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// LoginInput represents login request
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*TokenPair, *User, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, nil, model.ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		return nil, nil, model.ErrAccountLocked
	}

	// Check if account is active
	if user.Status != "active" {
		return nil, nil, errors.New("account is not active")
	}

	// Verify password (this should be done by comparing hashes)
	if !s.verifyPassword(input.Password, user.PasswordHash) {
		// Increment failed attempts
		user.FailedAttempts++
		if user.FailedAttempts >= s.config.MaxLoginAttempts {
			lockUntil := time.Now().Add(s.config.LockoutDuration)
			user.LockedUntil = &lockUntil
		}
		s.userRepo.Update(ctx, user)
		return nil, nil, model.ErrInvalidCredentials
	}

	// Reset failed attempts on successful login
	user.FailedAttempts = 0
	user.LockedUntil = nil
	s.userRepo.Update(ctx, user)

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, user, nil
}

// RefreshTokens refreshes the access token using a refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken, userAgent, ipAddress string) (*TokenPair, error) {
	// Find refresh token
	token, err := s.tokenRepo.FindRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, model.ErrTokenInvalid
	}

	if !token.IsValid() {
		return nil, model.ErrTokenExpired
	}

	// Get user
	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Revoke old refresh token (rotation)
	s.tokenRepo.RevokeRefreshToken(ctx, refreshToken)

	// Generate new tokens
	return s.generateTokenPair(ctx, user, userAgent, ipAddress)
}

// Logout revokes all tokens for a user
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	return s.tokenRepo.RevokeAllUserTokens(ctx, userID)
}

// LogoutSession revokes a specific refresh token
func (s *AuthService) LogoutSession(ctx context.Context, refreshToken string) error {
	return s.tokenRepo.RevokeRefreshToken(ctx, refreshToken)
}

// RequestPasswordReset initiates password reset
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Don't reveal if user exists
		return nil
	}

	// Create reset token
	token, err := model.NewPasswordResetToken(user.ID)
	if err != nil {
		return err
	}

	if err := s.tokenRepo.SavePasswordResetToken(ctx, token); err != nil {
		return err
	}

	// Send email
	if s.emailSvc != nil {
		return s.emailSvc.SendPasswordReset(ctx, user.Email, token.Token, user.FirstName)
	}

	return nil
}

// ResetPassword resets password using token
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	resetToken, err := s.tokenRepo.FindPasswordResetToken(ctx, token)
	if err != nil {
		return model.ErrTokenInvalid
	}

	if !resetToken.IsValid() {
		return model.ErrTokenExpired
	}

	// Hash new password
	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, resetToken.UserID, hash); err != nil {
		return err
	}

	// Mark token as used
	s.tokenRepo.MarkPasswordResetTokenUsed(ctx, token)

	// Revoke all existing tokens (force re-login)
	s.tokenRepo.RevokeAllUserTokens(ctx, resetToken.UserID)

	return nil
}

// SendEmailVerification sends email verification
func (s *AuthService) SendEmailVerification(ctx context.Context, userID string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.EmailVerified {
		return errors.New("email already verified")
	}

	token, err := model.NewEmailVerificationToken(userID, user.Email)
	if err != nil {
		return err
	}

	if err := s.tokenRepo.SaveEmailVerificationToken(ctx, token); err != nil {
		return err
	}

	if s.emailSvc != nil {
		return s.emailSvc.SendEmailVerification(ctx, user.Email, token.Token, user.FirstName)
	}

	return nil
}

// VerifyEmail verifies email using token
func (s *AuthService) VerifyEmail(ctx context.Context, token string) error {
	verifyToken, err := s.tokenRepo.FindEmailVerificationToken(ctx, token)
	if err != nil {
		return model.ErrTokenInvalid
	}

	if !verifyToken.IsValid() {
		return model.ErrTokenExpired
	}

	if err := s.userRepo.VerifyEmail(ctx, verifyToken.UserID); err != nil {
		return err
	}

	s.tokenRepo.MarkEmailVerificationTokenUsed(ctx, token)
	return nil
}

// ValidateToken validates a JWT token and returns claims
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, model.ErrTokenInvalid
}

// API Key Management

// CreateAPIKeyInput represents API key creation input
type CreateAPIKeyInput struct {
	UserID      string
	WorkspaceID string
	Name        string
	Scopes      []string
	ExpiresAt   *time.Time
}

// CreateAPIKey creates a new API key
func (s *AuthService) CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (*model.APIKey, string, error) {
	apiKey, rawKey, err := model.NewAPIKey(input.UserID, input.WorkspaceID, input.Name, input.Scopes, input.ExpiresAt)
	if err != nil {
		return nil, "", err
	}

	if err := s.apiKeyRepo.Save(ctx, apiKey); err != nil {
		return nil, "", err
	}

	return apiKey, rawKey, nil
}

// ValidateAPIKey validates an API key and returns the key details
func (s *AuthService) ValidateAPIKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	if len(rawKey) < 11 {
		return nil, model.ErrTokenInvalid
	}

	prefix := rawKey[:11]
	apiKey, err := s.apiKeyRepo.FindByPrefix(ctx, prefix)
	if err != nil {
		return nil, model.ErrTokenInvalid
	}

	if !apiKey.Verify(rawKey) {
		return nil, model.ErrTokenInvalid
	}

	// Update last used
	s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID)
	apiKey.MarkUsed()

	return apiKey, nil
}

// ListAPIKeys lists API keys for a user or workspace
func (s *AuthService) ListAPIKeys(ctx context.Context, userID, workspaceID string) ([]*model.APIKey, error) {
	if workspaceID != "" {
		return s.apiKeyRepo.ListByWorkspace(ctx, workspaceID)
	}
	return s.apiKeyRepo.ListByUser(ctx, userID)
}

// RevokeAPIKey revokes an API key
func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	key, err := s.apiKeyRepo.FindByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key.UserID != userID {
		return errors.New("unauthorized")
	}

	return s.apiKeyRepo.Revoke(ctx, keyID)
}

// Helper methods

func (s *AuthService) generateTokenPair(ctx context.Context, user *User, userAgent, ipAddress string) (*TokenPair, error) {
	// Generate access token
	expiresAt := time.Now().Add(s.config.AccessTokenExpiry)
	claims := JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.JWTIssuer,
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := model.NewRefreshToken(user.ID, userAgent, ipAddress, s.config.RefreshTokenExpiry)
	if err != nil {
		return nil, err
	}

	if err := s.tokenRepo.SaveRefreshToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) verifyPassword(password, hash string) bool {
	// Use bcrypt to compare
	return password != "" && hash != "" // Placeholder - implement with bcrypt
}

func (s *AuthService) hashPassword(password string) (string, error) {
	// Use bcrypt to hash
	return password, nil // Placeholder - implement with bcrypt
}
