// Package postgres provides PostgreSQL repository implementations for auth
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/auth/domain/model"
)

// TokenRepository implements token persistence with PostgreSQL
type TokenRepository struct {
	db *sql.DB
}

// NewTokenRepository creates a new token repository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{db: db}
}

// SaveRefreshToken saves a refresh token
func (r *TokenRepository) SaveRefreshToken(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, created_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Token, token.ExpiresAt,
		token.CreatedAt, token.UserAgent, token.IPAddress,
	)
	return err
}

// FindRefreshToken finds a refresh token
func (r *TokenRepository) FindRefreshToken(ctx context.Context, token string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at, revoked_at, user_agent, ip_address
		FROM refresh_tokens WHERE token = $1
	`
	var t model.RefreshToken
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&t.ID, &t.UserID, &t.Token, &t.ExpiresAt,
		&t.CreatedAt, &t.RevokedAt, &t.UserAgent, &t.IPAddress,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// RevokeRefreshToken revokes a refresh token
func (r *TokenRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// RevokeAllUserTokens revokes all tokens for a user
func (r *TokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	return err
}

// SavePasswordResetToken saves a password reset token
func (r *TokenRepository) SavePasswordResetToken(ctx context.Context, token *model.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Token, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// FindPasswordResetToken finds a password reset token
func (r *TokenRepository) FindPasswordResetToken(ctx context.Context, token string) (*model.PasswordResetToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens WHERE token = $1
	`
	var t model.PasswordResetToken
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&t.ID, &t.UserID, &t.Token, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (r *TokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, token string) error {
	query := `UPDATE password_reset_tokens SET used_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// SaveEmailVerificationToken saves an email verification token
func (r *TokenRepository) SaveEmailVerificationToken(ctx context.Context, token *model.EmailVerificationToken) error {
	query := `
		INSERT INTO email_verification_tokens (id, user_id, email, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID, token.UserID, token.Email, token.Token, token.ExpiresAt, token.CreatedAt,
	)
	return err
}

// FindEmailVerificationToken finds an email verification token
func (r *TokenRepository) FindEmailVerificationToken(ctx context.Context, token string) (*model.EmailVerificationToken, error) {
	query := `
		SELECT id, user_id, email, token, expires_at, used_at, created_at
		FROM email_verification_tokens WHERE token = $1
	`
	var t model.EmailVerificationToken
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&t.ID, &t.UserID, &t.Email, &t.Token, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkEmailVerificationTokenUsed marks an email verification token as used
func (r *TokenRepository) MarkEmailVerificationTokenUsed(ctx context.Context, token string) error {
	query := `UPDATE email_verification_tokens SET used_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// APIKeyRepository implements API key persistence with PostgreSQL
type APIKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Save saves an API key
func (r *APIKeyRepository) Save(ctx context.Context, key *model.APIKey) error {
	query := `
		INSERT INTO api_keys (id, user_id, workspace_id, name, key_hash, key_prefix, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		key.ID, key.UserID, key.WorkspaceID, key.Name,
		key.KeyHash, key.KeyPrefix, key.Scopes, key.ExpiresAt, key.CreatedAt,
	)
	return err
}

// FindByID finds an API key by ID
func (r *APIKeyRepository) FindByID(ctx context.Context, id string) (*model.APIKey, error) {
	query := `
		SELECT id, user_id, workspace_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_keys WHERE id = $1
	`
	return r.scanAPIKey(r.db.QueryRowContext(ctx, query, id))
}

// FindByPrefix finds an API key by prefix
func (r *APIKeyRepository) FindByPrefix(ctx context.Context, prefix string) (*model.APIKey, error) {
	query := `
		SELECT id, user_id, workspace_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_keys WHERE key_prefix = $1 AND revoked_at IS NULL
	`
	return r.scanAPIKey(r.db.QueryRowContext(ctx, query, prefix))
}

// ListByUser lists API keys by user
func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]*model.APIKey, error) {
	query := `
		SELECT id, user_id, workspace_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_keys WHERE user_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC
	`
	return r.listAPIKeys(ctx, query, userID)
}

// ListByWorkspace lists API keys by workspace
func (r *APIKeyRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]*model.APIKey, error) {
	query := `
		SELECT id, user_id, workspace_id, name, key_hash, key_prefix, scopes, last_used_at, expires_at, created_at, revoked_at
		FROM api_keys WHERE workspace_id = $1 AND revoked_at IS NULL ORDER BY created_at DESC
	`
	return r.listAPIKeys(ctx, query, workspaceID)
}

// Revoke revokes an API key
func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET revoked_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// UpdateLastUsed updates last used timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

func (r *APIKeyRepository) scanAPIKey(row *sql.Row) (*model.APIKey, error) {
	var k model.APIKey
	var scopesJSON []byte
	err := row.Scan(
		&k.ID, &k.UserID, &k.WorkspaceID, &k.Name, &k.KeyHash, &k.KeyPrefix,
		&scopesJSON, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	// Parse scopes from JSON array
	return &k, nil
}

func (r *APIKeyRepository) listAPIKeys(ctx context.Context, query, param string) ([]*model.APIKey, error) {
	rows, err := r.db.QueryContext(ctx, query, param)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		var k model.APIKey
		var scopesJSON []byte
		err := rows.Scan(
			&k.ID, &k.UserID, &k.WorkspaceID, &k.Name, &k.KeyHash, &k.KeyPrefix,
			&scopesJSON, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt, &k.RevokedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}
	return keys, nil
}

// OAuthRepository implements OAuth persistence with PostgreSQL
type OAuthRepository struct {
	db *sql.DB
}

// NewOAuthRepository creates a new OAuth repository
func NewOAuthRepository(db *sql.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// Save saves an OAuth connection
func (r *OAuthRepository) Save(ctx context.Context, conn *model.OAuthConnection) error {
	query := `
		INSERT INTO oauth_connections (id, user_id, provider, provider_id, email, access_token, refresh_token, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, provider) DO UPDATE SET
			access_token = $6, refresh_token = $7, expires_at = $8, updated_at = $10
	`
	_, err := r.db.ExecContext(ctx, query,
		conn.ID, conn.UserID, conn.Provider, conn.ProviderID, conn.Email,
		conn.AccessToken, conn.RefreshToken, conn.ExpiresAt, conn.CreatedAt, conn.UpdatedAt,
	)
	return err
}

// FindByProviderID finds an OAuth connection by provider ID
func (r *OAuthRepository) FindByProviderID(ctx context.Context, provider model.OAuthProvider, providerID string) (*model.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_id, email, access_token, refresh_token, expires_at, created_at, updated_at
		FROM oauth_connections WHERE provider = $1 AND provider_id = $2
	`
	var conn model.OAuthConnection
	err := r.db.QueryRowContext(ctx, query, provider, providerID).Scan(
		&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderID, &conn.Email,
		&conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.CreatedAt, &conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &conn, nil
}

// FindByUser finds OAuth connections by user
func (r *OAuthRepository) FindByUser(ctx context.Context, userID string) ([]*model.OAuthConnection, error) {
	query := `
		SELECT id, user_id, provider, provider_id, email, access_token, refresh_token, expires_at, created_at, updated_at
		FROM oauth_connections WHERE user_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conns []*model.OAuthConnection
	for rows.Next() {
		var conn model.OAuthConnection
		err := rows.Scan(
			&conn.ID, &conn.UserID, &conn.Provider, &conn.ProviderID, &conn.Email,
			&conn.AccessToken, &conn.RefreshToken, &conn.ExpiresAt, &conn.CreatedAt, &conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		conns = append(conns, &conn)
	}
	return conns, nil
}

// Delete deletes an OAuth connection
func (r *OAuthRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM oauth_connections WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// UserRepository implements user persistence for auth
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// AuthUser represents a user for the auth service (to avoid import cycles)
type AuthUser struct {
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

// FindByID finds a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id string) (*AuthUser, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, status, 
		       failed_attempts, locked_until, COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''), created_at
		FROM users WHERE id = $1 AND deleted_at IS NULL
	`
	return r.scanUser(r.db.QueryRowContext(ctx, query, id))
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*AuthUser, error) {
	query := `
		SELECT id, email, password_hash, first_name, last_name, email_verified, status, 
		       failed_attempts, locked_until, COALESCE(mfa_enabled, false), COALESCE(mfa_secret, ''), created_at
		FROM users WHERE email = $1 AND deleted_at IS NULL
	`
	return r.scanUser(r.db.QueryRowContext(ctx, query, email))
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *AuthUser) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, last_name, email_verified, status, 
		                   failed_attempts, mfa_enabled, mfa_secret, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.FirstName, user.LastName,
		user.EmailVerified, user.Status, user.FailedAttempts, user.MFAEnabled, user.MFASecret, user.CreatedAt,
	)
	return err
}

// Update updates user auth fields
func (r *UserRepository) Update(ctx context.Context, user *AuthUser) error {
	query := `
		UPDATE users SET failed_attempts = $1, locked_until = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, user.FailedAttempts, user.LockedUntil, user.ID)
	return err
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, passwordHash, userID)
	return err
}

// VerifyEmail marks email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, userID string) error {
	query := `UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *UserRepository) scanUser(row *sql.Row) (*AuthUser, error) {
	var u AuthUser
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FirstName, &u.LastName,
		&u.EmailVerified, &u.Status, &u.FailedAttempts, &u.LockedUntil,
		&u.MFAEnabled, &u.MFASecret, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Roles = []string{"user"} // Default role
	return &u, nil
}
