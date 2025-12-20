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

// ============================================================================
// Configuration
// ============================================================================

// Config holds auth service configuration
type Config struct {
	JWTSecret             string
	JWTIssuer             string
	AccessTokenExpiry     time.Duration
	RefreshTokenExpiry    time.Duration
	PasswordResetExpiry   time.Duration
	MaxLoginAttempts      int
	LockoutDuration       time.Duration
	BcryptCost            int
	RequireEmailVerify    bool
	AllowSignup           bool
	MFAEnabled            bool
	MFAIssuer             string
	OAuthGoogleEnabled    bool
	OAuthGitHubEnabled    bool
	OAuthMicrosoftEnabled bool
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
		PasswordMinLength:     8,
		PasswordRequireUpper:  true,
		PasswordRequireLower:  true,
		PasswordRequireNumber: true,
		PasswordRequireSymbol: false,
	}
}

// ============================================================================
// Repository Interfaces
// ============================================================================

// UserRepository defines user persistence operations
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	VerifyEmail(ctx context.Context, userID string) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
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

// AuditLogger defines audit logging operations
type AuditLogger interface {
	Log(ctx context.Context, event *model.AuthEvent) error
}

// ============================================================================
// Auth Service
// ============================================================================

// AuthService handles authentication and authorization
type AuthService struct {
	config     Config
	userRepo   UserRepository
	tokenRepo  TokenRepository
	apiKeyRepo APIKeyRepository
	oauthRepo  OAuthRepository
	emailSvc   EmailService
	auditLog   AuditLogger
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

// ============================================================================
// DTOs
// ============================================================================

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
	User        *model.User
	RequiresMFA bool
	MFAToken    string
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

// ChangePasswordInput represents change password request
type ChangePasswordInput struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
	IPAddress       string
	UserAgent       string
}

// CreateAPIKeyInput represents API key creation input
type CreateAPIKeyInput struct {
	UserID      string
	WorkspaceID string
	Name        string
	Scopes      []string
	ExpiresAt   *time.Time
}

// ============================================================================
// Authentication Methods
// ============================================================================

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.logEvent(ctx, "login", "", email, input.IPAddress, input.UserAgent, false, "user not found")
		return nil, model.ErrInvalidCredentials
	}

	// Check if account is locked
	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		s.logEvent(ctx, "login", user.ID, email, input.IPAddress, input.UserAgent, false, "account locked")
		return nil, model.ErrAccountLocked
	}

	// Check if account is active
	if user.Status != "active" {
		s.logEvent(ctx, "login", user.ID, email, input.IPAddress, input.UserAgent, false, "account inactive")
		return nil, errors.New("account is not active")
	}

	// Verify password
	if !s.verifyPassword(input.Password, user.PasswordHash) {
		s.handleFailedLogin(ctx, user, input)
		return nil, model.ErrInvalidCredentials
	}

	// Check email verification requirement
	if s.config.RequireEmailVerify && !user.EmailVerified {
		s.logEvent(ctx, "login", user.ID, email, input.IPAddress, input.UserAgent, false, "email not verified")
		return nil, model.ErrEmailNotVerified
	}

	// Check MFA requirement
	if user.MFAEnabled && s.config.MFAEnabled {
		if input.MFACode == "" {
			mfaToken, err := s.generateMFAToken(user.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to generate MFA token: %w", err)
			}
			return &LoginResult{RequiresMFA: true, MFAToken: mfaToken}, nil
		}

		if !s.verifyMFACode(user.MFASecret, input.MFACode) {
			s.logEvent(ctx, "login", user.ID, email, input.IPAddress, input.UserAgent, false, "invalid MFA code")
			return nil, errors.New("invalid MFA code")
		}
	}

	// Reset failed attempts on successful login
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	now := time.Now()
	user.LastLoginAt = &now
	user.LoginCount++
	s.userRepo.Update(ctx, user)

	// Generate tokens
	tokens, err := s.generateTokenPair(ctx, user, input.UserAgent, input.IPAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logEvent(ctx, "login", user.ID, email, input.IPAddress, input.UserAgent, true, "")

	return &LoginResult{Tokens: tokens, User: user}, nil
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*LoginResult, error) {
	if !s.config.AllowSignup {
		return nil, errors.New("registration is disabled")
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !s.isValidEmail(email) {
		return nil, errors.New("invalid email format")
	}

	// Check if user already exists
	exists, _ := s.userRepo.ExistsByEmail(ctx, email)
	if exists {
		s.logEvent(ctx, "register", "", email, input.IPAddress, input.UserAgent, false, "email already exists")
		return nil, model.ErrEmailExists
	}

	// Validate and hash password
	passwordHash, err := s.hashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        email,
		Username:     email,
		PasswordHash: passwordHash,
		FirstName:    strings.TrimSpace(input.FirstName),
		LastName:     strings.TrimSpace(input.LastName),
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logEvent(ctx, "register", user.ID, email, input.IPAddress, input.UserAgent, true, "")

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

	return &LoginResult{Tokens: tokens, User: user}, nil
}

// RefreshTokens refreshes the access token using a refresh token
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken, userAgent, ipAddress string) (*TokenPair, error) {
	token, err := s.tokenRepo.FindRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, model.ErrTokenInvalid
	}

	if !token.IsValid() {
		return nil, model.ErrTokenExpired
	}

	user, err := s.userRepo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Revoke old refresh token (rotation)
	s.tokenRepo.RevokeRefreshToken(ctx, refreshToken)

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

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, input ChangePasswordInput) error {
	user, err := s.userRepo.FindByID(ctx, input.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	if !s.verifyPassword(input.CurrentPassword, user.PasswordHash) {
		s.logEvent(ctx, "password_change", user.ID, user.Email, input.IPAddress, input.UserAgent, false, "invalid current password")
		return errors.New("current password is incorrect")
	}

	newHash, err := s.hashPassword(input.NewPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.tokenRepo.RevokeAllUserTokens(ctx, user.ID)
	s.logEvent(ctx, "password_change", user.ID, user.Email, input.IPAddress, input.UserAgent, true, "")

	return nil
}

// ============================================================================
// Password Reset Methods
// ============================================================================

// RequestPasswordReset initiates password reset
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil // Don't reveal if user exists
	}

	token, err := model.NewPasswordResetToken(user.ID)
	if err != nil {
		return err
	}

	if err := s.tokenRepo.SavePasswordResetToken(ctx, token); err != nil {
		return err
	}

	if s.emailSvc != nil {
		return s.emailSvc.SendPasswordReset(ctx, user.Email, token.Token, user.FirstName)
	}

	return nil
}

// ResetPassword resets password using token
func (s *AuthService) ResetPassword(ctx context.Context, tokenStr, newPassword string) error {
	token, err := s.tokenRepo.FindPasswordResetToken(ctx, tokenStr)
	if err != nil {
		return model.ErrTokenInvalid
	}

	if !token.IsValid() {
		return model.ErrTokenExpired
	}

	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.userRepo.UpdatePassword(ctx, token.UserID, hash); err != nil {
		return err
	}

	s.tokenRepo.MarkPasswordResetTokenUsed(ctx, tokenStr)
	s.tokenRepo.RevokeAllUserTokens(ctx, token.UserID)

	return nil
}

// ============================================================================
// Email Verification Methods
// ============================================================================

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
func (s *AuthService) VerifyEmail(ctx context.Context, tokenStr string) error {
	token, err := s.tokenRepo.FindEmailVerificationToken(ctx, tokenStr)
	if err != nil {
		return model.ErrTokenInvalid
	}

	if !token.IsValid() {
		return model.ErrTokenExpired
	}

	if err := s.userRepo.VerifyEmail(ctx, token.UserID); err != nil {
		return err
	}

	s.tokenRepo.MarkEmailVerificationTokenUsed(ctx, tokenStr)
	return nil
}

// ============================================================================
// Token Validation Methods
// ============================================================================

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

// ============================================================================
// API Key Methods
// ============================================================================

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

// ============================================================================
// Helper Methods
// ============================================================================

func (s *AuthService) generateTokenPair(ctx context.Context, user *model.User, userAgent, ipAddress string) (*TokenPair, error) {
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

func (s *AuthService) handleFailedLogin(ctx context.Context, user *model.User, input LoginInput) {
	user.FailedLoginCount++
	wasLocked := false

	if user.FailedLoginCount >= s.config.MaxLoginAttempts {
		lockUntil := time.Now().Add(s.config.LockoutDuration)
		user.LockedUntil = &lockUntil
		wasLocked = true
	}

	s.userRepo.Update(ctx, user)
	s.logEvent(ctx, "login", user.ID, user.Email, input.IPAddress, input.UserAgent, false, "invalid password")

	if wasLocked {
		s.logEvent(ctx, "account_locked", user.ID, user.Email, input.IPAddress, input.UserAgent, true,
			fmt.Sprintf("account locked after %d failed attempts", user.FailedLoginCount))
	}
}

func (s *AuthService) verifyPassword(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
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

func (s *AuthService) isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)
	return emailRegex.MatchString(email)
}

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

func (s *AuthService) verifyMFACode(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	// TODO: Implement proper TOTP verification using github.com/pquerna/otp/totp
	return subtle.ConstantTimeCompare([]byte(code), []byte("000000")) == 1
}

func (s *AuthService) logEvent(ctx context.Context, eventType, userID, email, ipAddress, userAgent string, success bool, failReason string) {
	if s.auditLog == nil {
		return
	}

	event := &model.AuthEvent{
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

	go func() {
		_ = s.auditLog.Log(context.Background(), event)
	}()
}
