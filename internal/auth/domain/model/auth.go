// Package model defines auth domain models
package model

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ============================================================================
// Custom Types
// ============================================================================

// TokenType represents the type of token
type TokenType string

const (
	TokenTypeAccess        TokenType = "access"
	TokenTypeRefresh       TokenType = "refresh"
	TokenTypePasswordReset TokenType = "password_reset"
	TokenTypeEmailVerify   TokenType = "email_verify"
	TokenTypeAPIKey        TokenType = "api_key"
)

// OAuthProvider represents an OAuth provider
type OAuthProvider string

const (
	OAuthProviderGoogle    OAuthProvider = "google"
	OAuthProviderGitHub    OAuthProvider = "github"
	OAuthProviderMicrosoft OAuthProvider = "microsoft"
)

// StringArray is a custom type for handling PostgreSQL text[] or JSON arrays
type StringArray []string

// Scan implements sql.Scanner for StringArray
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan StringArray")
	}
	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for StringArray
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(s)
}

// JSONMap is a custom type for handling PostgreSQL JSONB
type JSONMap map[string]interface{}

// Scan implements sql.Scanner for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONMap")
	}
	return json.Unmarshal(bytes, j)
}

// Value implements driver.Valuer for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(j)
}

// ============================================================================
// Auth User Model
// ============================================================================

// User represents a user for authentication
type User struct {
	ID               string     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email            string     `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Username         string     `json:"username" gorm:"type:varchar(100);uniqueIndex;not null"`
	PasswordHash     string     `json:"-" gorm:"type:varchar(255);not null"`
	FirstName        string     `json:"firstName" gorm:"type:varchar(100)"`
	LastName         string     `json:"lastName" gorm:"type:varchar(100)"`
	AvatarURL        string     `json:"avatarUrl,omitempty" gorm:"type:text"`
	Phone            string     `json:"phone,omitempty" gorm:"type:varchar(20)"`
	Status           string     `json:"status" gorm:"type:varchar(50);not null;default:'active';index"`
	EmailVerified    bool       `json:"emailVerified" gorm:"default:false"`
	EmailVerifiedAt  *time.Time `json:"-" gorm:"type:timestamptz"`
	PhoneVerified    bool       `json:"-" gorm:"default:false"`
	LastLoginAt      *time.Time `json:"lastLoginAt,omitempty" gorm:"type:timestamptz"`
	LoginCount       int        `json:"-" gorm:"default:0"`
	FailedLoginCount int        `json:"-" gorm:"default:0"`
	LockedUntil      *time.Time `json:"-" gorm:"type:timestamptz"`
	MFAEnabled       bool       `json:"mfaEnabled" gorm:"default:false"`
	MFASecret        string     `json:"-" gorm:"type:varchar(255)"`
	Preferences      JSONMap    `json:"-" gorm:"type:jsonb;default:'{}'"`
	Metadata         JSONMap    `json:"-" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time  `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt        time.Time  `json:"updatedAt" gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt        *time.Time `json:"-" gorm:"type:timestamptz;index"`
}

// TableName specifies the table name for GORM
func (User) TableName() string { return "users" }

// BeforeCreate hook to generate UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.Username == "" {
		u.Username = u.Email
	}
	return nil
}

// ============================================================================
// Refresh Token Model
// ============================================================================

// RefreshToken represents a refresh token
type RefreshToken struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    string     `json:"userId" gorm:"type:uuid;index;not null"`
	Token     string     `json:"-" gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"type:timestamptz;not null"`
	RevokedAt *time.Time `json:"-" gorm:"type:timestamptz;index"`
	UserAgent string     `json:"-" gorm:"type:varchar(500)"`
	IPAddress string     `json:"-" gorm:"type:varchar(45)"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (RefreshToken) TableName() string { return "refresh_tokens" }

// BeforeCreate hook to generate UUID
func (t *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
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

// ============================================================================
// Password Reset Token Model
// ============================================================================

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    string     `json:"userId" gorm:"type:uuid;index;not null"`
	Token     string     `json:"-" gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"type:timestamptz;not null"`
	UsedAt    *time.Time `json:"-" gorm:"type:timestamptz"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (PasswordResetToken) TableName() string { return "password_reset_tokens" }

// BeforeCreate hook to generate UUID
func (t *PasswordResetToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
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

// ============================================================================
// Email Verification Token Model
// ============================================================================

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID        string     `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID    string     `json:"userId" gorm:"type:uuid;index;not null"`
	Email     string     `json:"email" gorm:"type:varchar(255);not null"`
	Token     string     `json:"-" gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt time.Time  `json:"expiresAt" gorm:"type:timestamptz;not null"`
	UsedAt    *time.Time `json:"-" gorm:"type:timestamptz"`
	CreatedAt time.Time  `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (EmailVerificationToken) TableName() string { return "email_verification_tokens" }

// BeforeCreate hook to generate UUID
func (t *EmailVerificationToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
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

// ============================================================================
// API Key Model
// ============================================================================

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          string      `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID      string      `json:"userId" gorm:"type:uuid;index;not null"`
	WorkspaceID *string     `json:"workspaceId,omitempty" gorm:"type:uuid;index"`
	Name        string      `json:"name" gorm:"type:varchar(255);not null"`
	KeyHash     string      `json:"-" gorm:"type:text;not null"`
	KeyPrefix   string      `json:"keyPrefix" gorm:"type:varchar(20);index;not null"`
	Scopes      StringArray `json:"scopes" gorm:"type:jsonb;default:'[]'"`
	LastUsedAt  *time.Time  `json:"lastUsedAt,omitempty" gorm:"type:timestamptz"`
	ExpiresAt   *time.Time  `json:"expiresAt,omitempty" gorm:"type:timestamptz"`
	RevokedAt   *time.Time  `json:"-" gorm:"type:timestamptz"`
	CreatedAt   time.Time   `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (APIKey) TableName() string { return "api_keys" }

// BeforeCreate hook to generate UUID
func (k *APIKey) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
		k.ID = uuid.New().String()
	}
	return nil
}

// NewAPIKey creates a new API key
func NewAPIKey(userID, workspaceID, name string, scopes []string, expiresAt *time.Time) (*APIKey, string, error) {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", err
	}

	rawKey := "lf_" + base64.RawURLEncoding.EncodeToString(keyBytes)

	hash, err := bcrypt.GenerateFromPassword([]byte(rawKey), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	var wsID *string
	if workspaceID != "" {
		wsID = &workspaceID
	}

	return &APIKey{
		ID:          uuid.New().String(),
		UserID:      userID,
		WorkspaceID: wsID,
		Name:        name,
		KeyHash:     string(hash),
		KeyPrefix:   rawKey[:11],
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

// ============================================================================
// OAuth Connection Model
// ============================================================================

// OAuthConnection represents a user's OAuth connection
type OAuthConnection struct {
	ID           string        `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID       string        `json:"userId" gorm:"type:uuid;index;not null"`
	Provider     OAuthProvider `json:"provider" gorm:"type:varchar(50);not null"`
	ProviderID   string        `json:"providerId" gorm:"type:varchar(255);not null"`
	Email        string        `json:"email" gorm:"type:varchar(255)"`
	AccessToken  string        `json:"-" gorm:"type:text"`
	RefreshToken string        `json:"-" gorm:"type:text"`
	ExpiresAt    *time.Time    `json:"-" gorm:"type:timestamptz"`
	Metadata     JSONMap       `json:"-" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time     `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt    time.Time     `json:"updatedAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (OAuthConnection) TableName() string { return "oauth_connections" }

// BeforeCreate hook to generate UUID
func (o *OAuthConnection) BeforeCreate(tx *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.New().String()
	}
	return nil
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

// ============================================================================
// Session Model
// ============================================================================

// Session represents a user session
type Session struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID     string    `json:"userId" gorm:"type:uuid;index;not null"`
	Token      string    `json:"-" gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt  time.Time `json:"expiresAt" gorm:"type:timestamptz;not null"`
	UserAgent  string    `json:"-" gorm:"type:varchar(500)"`
	IPAddress  string    `json:"-" gorm:"type:varchar(45)"`
	CreatedAt  time.Time `json:"createdAt" gorm:"type:timestamptz;not null;default:now()"`
	LastSeenAt time.Time `json:"lastSeenAt" gorm:"type:timestamptz;not null;default:now()"`

	// Associations
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName specifies the table name for GORM
func (Session) TableName() string { return "sessions" }

// BeforeCreate hook to generate UUID
func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
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

// ============================================================================
// Login Attempt Model (for rate limiting)
// ============================================================================

// LoginAttempt tracks login attempts for rate limiting
type LoginAttempt struct {
	ID        string    `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email     string    `json:"email" gorm:"type:varchar(255);index;not null"`
	IPAddress string    `json:"ipAddress" gorm:"type:varchar(45);index;not null"`
	Success   bool      `json:"success" gorm:"not null"`
	Reason    string    `json:"reason,omitempty" gorm:"type:varchar(255)"`
	CreatedAt time.Time `json:"createdAt" gorm:"type:timestamptz;not null;default:now();index"`
}

// TableName specifies the table name for GORM
func (LoginAttempt) TableName() string { return "login_attempts" }

// BeforeCreate hook to generate UUID
func (l *LoginAttempt) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// ============================================================================
// Auth Event Model (for audit logging)
// ============================================================================

// AuthEvent represents an authentication event for audit logging
type AuthEvent struct {
	ID         string    `json:"id" gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Type       string    `json:"type" gorm:"type:varchar(50);index;not null"`
	UserID     string    `json:"userId,omitempty" gorm:"type:uuid;index"`
	Email      string    `json:"email,omitempty" gorm:"type:varchar(255);index"`
	IPAddress  string    `json:"ipAddress" gorm:"type:varchar(45);index"`
	UserAgent  string    `json:"userAgent,omitempty" gorm:"type:varchar(500)"`
	Success    bool      `json:"success" gorm:"not null"`
	FailReason string    `json:"failReason,omitempty" gorm:"type:varchar(255)"`
	Metadata   JSONMap   `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	OccurredAt time.Time `json:"occurredAt" gorm:"type:timestamptz;not null;default:now();index"`
}

// TableName specifies the table name for GORM
func (AuthEvent) TableName() string { return "auth_events" }

// BeforeCreate hook to generate UUID
func (e *AuthEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// ============================================================================
// Errors
// ============================================================================

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account is locked")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrTokenExpired       = errors.New("token has expired")
	ErrTokenInvalid       = errors.New("token is invalid")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrUsernameExists     = errors.New("username already exists")
)

// ============================================================================
// Helper Functions
// ============================================================================

func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
