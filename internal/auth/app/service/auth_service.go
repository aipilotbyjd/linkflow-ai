// Package service provides auth business logic
package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
	"golang.org/x/crypto/bcrypt"
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
	BcryptCost          int
	RequireEmailVerify  bool
	AllowSignup         bool
	// MFA settings
	MFAEnabled          bool
	MFAIssuer           string
	// OAuth settings
	OAuthGoogleEnabled    bool
	OAuthGitHubEnabled    bool
	OAuthMicrosoftEnabled bool
	// Password policy
	PasswordMinLength     int
	PasswordRequireUpper  bool
	PasswordRequireLower  bool
	PasswordRequireNumber bool
	PasswordRequireSymbol bool
}

// DefaultConfig returns default auth configuration
func DefaultConfig() Config {
	return Config{
		JWTSecret:             "change-me-in-production",
		JWTIssuer:             "linkflow-ai",
		AccessTokenExpiry:     15 * time.Minute,
		RefreshTokenExpiry:    7 * 24 * time.Hour,
		PasswordResetExpiry:   1 * time.Hour,
		MaxLoginAttempts:      5,
		LockoutDuration:       15 * time.Minute,
		BcryptCost:            bcrypt.DefaultCost,
		RequireEmailVerify:    false,
		AllowSignup:           true,
		MFAEnabled:            false,
		MFAIssuer:             "LinkFlow",
		OAuthGoogleEnabled:    false,
		OAuthGitHubEnabled:    false,
		OAuthMicrosoftEnabled: false,
		PasswordMinLength:     8,
		PasswordRequireUpper:  true,
		PasswordRequireLower:  true,
		PasswordRequireNumber: true,
		PasswordRequireSymbol: false,
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
	SendLoginAlert(ctx context.Context, email, name, ipAddress, userAgent string) error
	SendMFACode(ctx context.Context, email, code, name string) error
}

// AuditLogger defines audit logging operations
type AuditLogger interface {
	LogAuthEvent(ctx context.Context, event *AuthEvent) error
}

// AuthEvent represents an authentication event for audit logging
type AuthEvent struct {
	ID          string
	Type        AuthEventType
	UserID      string
	Email       string
	IPAddress   string
	UserAgent   string
	Success     bool
	FailReason  string
	Metadata    map[string]interface{}
	OccurredAt  time.Time
}

// AuthEventType represents types of auth events
type AuthEventType string

const (
	AuthEventLogin           AuthEventType = "login"
	AuthEventLogout          AuthEventType = "logout"
	AuthEventRegister        AuthEventType = "register"
	AuthEventPasswordReset   AuthEventType = "password_reset"
	AuthEventPasswordChange  AuthEventType = "password_change"
	AuthEventEmailVerify     AuthEventType = "email_verify"
	AuthEventMFAEnable       AuthEventType = "mfa_enable"
	AuthEventMFADisable      AuthEventType = "mfa_disable"
	AuthEventAPIKeyCreate    AuthEventType = "api_key_create"
	AuthEventAPIKeyRevoke    AuthEventType = "api_key_revoke"
	AuthEventTokenRefresh    AuthEventType = "token_refresh"
	AuthEventOAuthLogin      AuthEventType = "oauth_login"
	AuthEventAccountLocked   AuthEventType = "account_locked"
	AuthEventAccountUnlocked AuthEventType = "account_unlocked"
)

// User represents a user for auth purposes
type User struct {
	ID             string
	Email          string
	PasswordHash   string
	FirstName      string
	LastName       string
	EmailVerified  bool
	Status         string
	FailedAttempts int
	LockedUntil    *time.Time
	MFAEnabled     bool
	MFASecret      string
	Roles          []string
	CreatedAt      time.Time
}

// AuthService handles authentication and authorization
type AuthService struct {
	config      Config
	userRepo    UserRepository
	tokenRepo   TokenRepository
	apiKeyRepo  APIKeyRepository
	oauthRepo   OAuthRepository
	emailSvc    EmailService
	auditLog    AuditLogger
}

// NewAuthService creates a new auth service
func NewAuthService(
	config Config,
	userRepo UserRepository,
	tokenRepo TokenRepository,
	apiKeyRepo APIKeyRepository,
	oauthRepo OAuthRepository,
	emailSvc EmailService,
	auditLog AuditLogger,
) *AuthService {
	return &AuthService{
		config:     config,
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		apiKeyRepo: apiKeyRepo,
		oauthRepo:  oauthRepo,
		emailSvc:   emailSvc,
		auditLog:   auditLog,
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
	MFACode   string
	UserAgent string
	IPAddress string
}

// LoginResult represents login response with MFA status
type LoginResult struct {
	Tokens      *TokenPair
	User        *User
	RequiresMFA bool
	MFAToken    string
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	// Normalize email
	email := strings.ToLower(strings.TrimSpace(input.Email))
	
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.logAuthEvent(ctx, AuthEventLogin, "", email, input.IPAddress, input.UserAgent, false, "user not found")
		return nil, model.ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		s.logAuthEvent(ctx, AuthEventLogin, user.ID, email, input.IPAddress, input.UserAgent, false, "account locked")
		return nil, model.ErrAccountLocked
	}

	// Check if account is active
	if user.Status != "active" {
		s.logAuthEvent(ctx, AuthEventLogin, user.ID, email, input.IPAddress, input.UserAgent, false, "account inactive")
		return nil, errors.New("account is not active")
	}

	// Verify password using constant-time comparison
	if !s.verifyPassword(input.Password, user.PasswordHash) {
		s.handleFailedLogin(ctx, user, input)
		return nil, model.ErrInvalidCredentials
	}

	// Check email verification requirement
	if s.config.RequireEmailVerify && !user.EmailVerified {
		s.logAuthEvent(ctx, AuthEventLogin, user.ID, email, input.IPAddress, input.UserAgent, false, "email not verified")
		return nil, model.ErrEmailNotVerified
	}

	// Check MFA requirement
	if user.MFAEnabled && s.config.MFAEnabled {
		if input.MFACode == "" {
			// Return MFA required response
			mfaToken, err := s.generateMFAToken(user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to generate MFA token: %w", err)
			}
			return &LoginResult{
				RequiresMFA: true,
				MFAToken:    mfaToken,
			}, nil
		}
		
		// Verify MFA code
		if !s.verifyMFACode(user.MFASecret, input.MFACode) {
			s.logAuthEvent(ctx, AuthEventLogin, user.ID, email, input.IPAddress, input.UserAgent, false, "invalid MFA code")
			return nil, errors.New("invalid MFA code")
		}
	}

	// Reset failed attempts on successful login
	user.FailedAttempts = 0
	user.LockedUntil = nil
	s.userRepo.Update(ctx, user)

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Log successful login
	s.logAuthEvent(ctx, AuthEventLogin, user.ID, email, input.IPAddress, input.UserAgent, true, "")

	return &LoginResult{
		Tokens: tokens,
		User:   user,
	}, nil
}

// handleFailedLogin handles failed login attempt
func (s *AuthService) handleFailedLogin(ctx context.Context, user *User, input LoginInput) {
	user.FailedAttempts++
	wasLocked := false
	
	if user.FailedAttempts >= s.config.MaxLoginAttempts {
		lockUntil := time.Now().Add(s.config.LockoutDuration)
		user.LockedUntil = &lockUntil
		wasLocked = true
	}
	
	s.userRepo.Update(ctx, user)
	s.logAuthEvent(ctx, AuthEventLogin, user.ID, user.Email, input.IPAddress, input.UserAgent, false, "invalid password")
	
	if wasLocked {
		s.logAuthEvent(ctx, AuthEventAccountLocked, user.ID, user.Email, input.IPAddress, input.UserAgent, true, 
			fmt.Sprintf("account locked after %d failed attempts", user.FailedAttempts))
	}
}

// RegisterInput represents user registration input
type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
	UserAgent string
	IPAddress string
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*LoginResult, error) {
	if !s.config.AllowSignup {
		return nil, errors.New("registration is disabled")
	}

	// Normalize and validate email
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !s.isValidEmail(email) {
		return nil, errors.New("invalid email format")
	}

	// Check if user already exists
	existing, _ := s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		s.logAuthEvent(ctx, AuthEventRegister, "", email, input.IPAddress, input.UserAgent, false, "email already exists")
		return nil, errors.New("email already registered")
	}

	// Validate and hash password
	passwordHash, err := s.hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &User{
		ID:            uuid.New().String(),
		Email:         email,
		PasswordHash:  passwordHash,
		FirstName:     strings.TrimSpace(input.FirstName),
		LastName:      strings.TrimSpace(input.LastName),
		EmailVerified: false,
		Status:        "active",
		Roles:         []string{"user"},
		CreatedAt:     time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Log registration
	s.logAuthEvent(ctx, AuthEventRegister, user.ID, email, input.IPAddress, input.UserAgent, true, "")

	// Send welcome email
	if s.emailSvc != nil {
		go s.emailSvc.SendWelcome(context.Background(), user.Email, user.FirstName)
	}

	// Send email verification if required
	if s.config.RequireEmailVerify {
		go s.SendEmailVerification(context.Background(), user.ID)
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &LoginResult{
		Tokens: tokens,
		User:   user,
	}, nil
}

// ChangePasswordInput represents change password request
type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
	IPAddress       string
	UserAgent       string
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if !s.verifyPassword(input.CurrentPassword, user.PasswordHash) {
		s.logAuthEvent(ctx, AuthEventPasswordChange, user.ID, user.Email, input.IPAddress, input.UserAgent, false, "invalid current password")
		return errors.New("current password is incorrect")
	}

	// Hash new password (validates internally)
	newHash, err := s.hashPassword(input.NewPassword)
	if err != nil {
		return err
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all existing tokens (force re-login on other devices)
	s.tokenRepo.RevokeAllUserTokens(ctx, user.ID)

	// Log event
	s.logAuthEvent(ctx, AuthEventPasswordChange, user.ID, user.Email, input.IPAddress, input.UserAgent, true, "")

	return nil
}

// isValidEmail validates email format
func (s *AuthService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return emailRegex.MatchString(email)
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
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *AuthService) hashPassword(password string) (string, error) {
	if err := s.validatePassword(password); err != nil {
		return "", err
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), s.config.BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// validatePassword validates password against configured policy
func (s *AuthService) validatePassword(password string) error {
	if len(password) < s.config.PasswordMinLength {
		return fmt.Errorf("password must be at least %d characters", s.config.PasswordMinLength)
	}
	
	if s.config.PasswordRequireUpper && !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return errors.New("password must contain at least one uppercase letter")
	}
	
	if s.config.PasswordRequireLower && !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return errors.New("password must contain at least one lowercase letter")
	}
	
	if s.config.PasswordRequireNumber && !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return errors.New("password must contain at least one digit")
	}
	
	if s.config.PasswordRequireSymbol && !regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password) {
		return errors.New("password must contain at least one special character")
	}
	
	return nil
}

// generateMFAToken generates a temporary token for MFA flow
func (s *AuthService) generateMFAToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"uid":  userID,
		"type": "mfa",
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

// verifyMFACode verifies TOTP code (placeholder - integrate with TOTP library)
func (s *AuthService) verifyMFACode(secret, code string) bool {
	// TODO: Implement proper TOTP verification using github.com/pquerna/otp/totp
	// For now, use constant-time comparison for basic verification
	if secret == "" || code == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(code), []byte("000000")) == 1 // Placeholder
}

// logAuthEvent logs an authentication event
func (s *AuthService) logAuthEvent(ctx context.Context, eventType AuthEventType, userID, email, ipAddress, userAgent string, success bool, failReason string) {
	if s.auditLog == nil {
		return
	}
	
	event := &AuthEvent{
		ID:         uuid.New().String(),
		Type:       eventType,
		UserID:     userID,
		Email:      email,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		FailReason: failReason,
		Metadata:   make(map[string]interface{}),
		OccurredAt: time.Now(),
	}
	
	// Fire and forget - don't block auth flow for audit logging
	go func() {
		_ = s.auditLog.LogAuthEvent(context.Background(), event)
	}()
}
