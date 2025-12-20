// Package postgres provides PostgreSQL repository implementations for auth using GORM
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
	"gorm.io/gorm"
)

// ============================================================================
// User Repository
// ============================================================================

// UserRepository implements user persistence with GORM
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("email = ? AND deleted_at IS NULL", email).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ? AND deleted_at IS NULL", username).
		First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("password_hash", passwordHash).Error
}

// UpdateLoginStats updates login statistics
func (r *UserRepository) UpdateLoginStats(ctx context.Context, userID string, success bool) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if success {
		updates["last_login_at"] = time.Now()
		updates["login_count"] = gorm.Expr("login_count + 1")
		updates["failed_login_count"] = 0
		updates["locked_until"] = nil
	} else {
		updates["failed_login_count"] = gorm.Expr("failed_login_count + 1")
	}

	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

// LockAccount locks a user account
func (r *UserRepository) LockAccount(ctx context.Context, userID string, until time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Update("locked_until", until).Error
}

// VerifyEmail marks email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"email_verified":    true,
			"email_verified_at": time.Now(),
		}).Error
}

// ExistsByEmail checks if a user exists by email
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("email = ? AND deleted_at IS NULL", email).
		Count(&count).Error
	return count > 0, err
}

// ExistsByUsername checks if a user exists by username
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("username = ? AND deleted_at IS NULL", username).
		Count(&count).Error
	return count > 0, err
}

// ============================================================================
// Token Repository
// ============================================================================

// TokenRepository implements token persistence with GORM
type TokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *gorm.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// SaveRefreshToken saves a refresh token
func (r *TokenRepository) SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindRefreshToken finds a refresh token
func (r *TokenRepository) FindRefreshToken(ctx context.Context, tokenStr string) (*model.RefreshToken, error) {
	var token model.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token = ?", tokenStr).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenInvalid
		}
		return nil, err
	}
	return &token, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, tokenStr string) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("token = ?", tokenStr).
		Update("revoked_at", time.Now()).Error
}

// RevokeAllUserTokens revokes all tokens for a user
func (r *TokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&model.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", time.Now()).Error
}

// CleanupExpiredTokens removes expired tokens
func (r *TokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ? OR revoked_at IS NOT NULL", time.Now()).
		Delete(&model.RefreshToken{}).Error
}

// SavePasswordResetToken saves a password reset token
func (r *TokenRepository) SavePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindPasswordResetToken finds a password reset token
func (r *TokenRepository) FindPasswordResetToken(ctx context.Context, tokenStr string) (*model.PasswordResetToken, error) {
	var token model.PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token = ?", tokenStr).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenInvalid
		}
		return nil, err
	}
	return &token, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (r *TokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenStr string) error {
	return r.db.WithContext(ctx).
		Model(&model.PasswordResetToken{}).
		Where("token = ?", tokenStr).
		Update("used_at", time.Now()).Error
}

// SaveEmailVerificationToken saves an email verification token
func (r *TokenRepository) SaveEmailVerificationToken(ctx context.Context, token *model.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindEmailVerificationToken finds an email verification token
func (r *TokenRepository) FindEmailVerificationToken(ctx context.Context, tokenStr string) (*model.EmailVerificationToken, error) {
	var token model.EmailVerificationToken
	err := r.db.WithContext(ctx).
		Where("token = ?", tokenStr).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenInvalid
		}
		return nil, err
	}
	return &token, nil
}

// MarkEmailVerificationTokenUsed marks an email verification token as used
func (r *TokenRepository) MarkEmailVerificationTokenUsed(ctx context.Context, tokenStr string) error {
	return r.db.WithContext(ctx).
		Model(&model.EmailVerificationToken{}).
		Where("token = ?", tokenStr).
		Update("used_at", time.Now()).Error
}

// ============================================================================
// API Key Repository
// ============================================================================

// APIKeyRepository implements API key persistence with GORM
type APIKeyRepository struct {
	db *gorm.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Save saves an API key
func (r *APIKeyRepository) Save(ctx context.Context, key *model.APIKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

// FindByID finds an API key by ID
func (r *APIKeyRepository) FindByID(ctx context.Context, id string) (*model.APIKey, error) {
	var key model.APIKey
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenInvalid
		}
		return nil, err
	}
	return &key, nil
}

// FindByPrefix finds an API key by prefix
func (r *APIKeyRepository) FindByPrefix(ctx context.Context, prefix string) (*model.APIKey, error) {
	var key model.APIKey
	err := r.db.WithContext(ctx).
		Where("key_prefix = ? AND revoked_at IS NULL", prefix).
		First(&key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrTokenInvalid
		}
		return nil, err
	}
	return &key, nil
}

// ListByUser lists API keys by user
func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// ListByWorkspace lists API keys by workspace
func (r *APIKeyRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.APIKey, error) {
	var keys []*model.APIKey
	err := r.db.WithContext(ctx).
		Where("workspace_id = ? AND revoked_at IS NULL", workspaceID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// Revoke revokes an API key
func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}

// UpdateLastUsed updates last used timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Model(&model.APIKey{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

// ============================================================================
// OAuth Repository
// ============================================================================

// OAuthRepository implements OAuth persistence with GORM
type OAuthRepository struct {
	db *gorm.DB
}

// NewOAuthRepository creates a new OAuth repository
func NewOAuthRepository(db *gorm.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// Save saves an OAuth connection (upsert)
func (r *OAuthRepository) Save(ctx context.Context, conn *model.OAuthConnection) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", conn.UserID, conn.Provider).
		Assign(model.OAuthConnection{
			ProviderID:   conn.ProviderID,
			Email:        conn.Email,
			AccessToken:  conn.AccessToken,
			RefreshToken: conn.RefreshToken,
			ExpiresAt:    conn.ExpiresAt,
			Metadata:     conn.Metadata,
			UpdatedAt:    time.Now(),
		}).
		FirstOrCreate(conn).Error
}

// FindByProviderID finds an OAuth connection by provider ID
func (r *OAuthRepository) FindByProviderID(ctx context.Context, provider model.OAuthProvider, providerID string) (*model.OAuthConnection, error) {
	var conn model.OAuthConnection
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_id = ?", provider, providerID).
		First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &conn, nil
}

// FindByUser finds OAuth connections by user
func (r *OAuthRepository) FindByUser(ctx context.Context, userID string) ([]*model.OAuthConnection, error) {
	var conns []*model.OAuthConnection
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&conns).Error
	return conns, err
}

// Delete deletes an OAuth connection
func (r *OAuthRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.OAuthConnection{}).Error
}

// ============================================================================
// Session Repository
// ============================================================================

// SessionRepository implements session persistence with GORM
type SessionRepository struct {
	db *gorm.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Save saves a session
func (r *SessionRepository) Save(ctx context.Context, session *model.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// FindByToken finds a session by token
func (r *SessionRepository) FindByToken(ctx context.Context, token string) (*model.Session, error) {
	var session model.Session
	err := r.db.WithContext(ctx).
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&session).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// ListByUser lists active sessions by user
func (r *SessionRepository) ListByUser(ctx context.Context, userID string) ([]*model.Session, error) {
	var sessions []*model.Session
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("last_seen_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// Touch updates last seen time
func (r *SessionRepository) Touch(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).
		Model(&model.Session{}).
		Where("id = ?", sessionID).
		Update("last_seen_at", time.Now()).Error
}

// Delete deletes a session
func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.Session{}).Error
}

// DeleteAllByUser deletes all sessions for a user
func (r *SessionRepository) DeleteAllByUser(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&model.Session{}).Error
}

// CleanupExpired removes expired sessions
func (r *SessionRepository) CleanupExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.Session{}).Error
}

// ============================================================================
// Auth Event Repository (Audit Logging)
// ============================================================================

// AuthEventRepository implements auth event persistence with GORM
type AuthEventRepository struct {
	db *gorm.DB
}

// NewAuthEventRepository creates a new auth event repository
func NewAuthEventRepository(db *gorm.DB) *AuthEventRepository {
	return &AuthEventRepository{db: db}
}

// Save saves an auth event
func (r *AuthEventRepository) Save(ctx context.Context, event *model.AuthEvent) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// ListByUser lists auth events by user
func (r *AuthEventRepository) ListByUser(ctx context.Context, userID string, limit int) ([]*model.AuthEvent, error) {
	var events []*model.AuthEvent
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// ListByEmail lists auth events by email
func (r *AuthEventRepository) ListByEmail(ctx context.Context, email string, limit int) ([]*model.AuthEvent, error) {
	var events []*model.AuthEvent
	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// CountFailedAttempts counts failed login attempts in a time window
func (r *AuthEventRepository) CountFailedAttempts(ctx context.Context, email string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.AuthEvent{}).
		Where("email = ? AND type = ? AND success = ? AND occurred_at > ?",
			email, "login", false, since).
		Count(&count).Error
	return count, err
}

// ============================================================================
// Login Attempt Repository
// ============================================================================

// LoginAttemptRepository implements login attempt persistence with GORM
type LoginAttemptRepository struct {
	db *gorm.DB
}

// NewLoginAttemptRepository creates a new login attempt repository
func NewLoginAttemptRepository(db *gorm.DB) *LoginAttemptRepository {
	return &LoginAttemptRepository{db: db}
}

// Save saves a login attempt
func (r *LoginAttemptRepository) Save(ctx context.Context, attempt *model.LoginAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

// CountRecentFailed counts recent failed attempts
func (r *LoginAttemptRepository) CountRecentFailed(ctx context.Context, email, ip string, since time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&model.LoginAttempt{}).Where("success = ? AND created_at > ?", false, since)

	if email != "" {
		query = query.Where("email = ?", email)
	}
	if ip != "" {
		query = query.Where("ip_address = ?", ip)
	}

	err := query.Count(&count).Error
	return count, err
}

// CleanupOld removes old login attempts
func (r *LoginAttemptRepository) CleanupOld(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&model.LoginAttempt{}).Error
}
