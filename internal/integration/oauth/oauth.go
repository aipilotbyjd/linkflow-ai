// Package oauth provides OAuth2 flow implementation for integrations
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Provider represents an OAuth provider configuration
type Provider struct {
	Name            string
	ClientID        string
	ClientSecret    string
	AuthURL         string
	TokenURL        string
	Scopes          []string
	RedirectURL     string
	UserInfoURL     string
	AdditionalParams map[string]string
}

// Providers contains all supported OAuth providers
var Providers = map[string]*Provider{
	"google": {
		Name:        "Google",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		AdditionalParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"google_sheets": {
		Name:        "Google Sheets",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/spreadsheets",
			"https://www.googleapis.com/auth/drive.file",
		},
		AdditionalParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"google_drive": {
		Name:        "Google Drive",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/drive",
		},
		AdditionalParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"gmail": {
		Name:        "Gmail",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.readonly",
		},
		AdditionalParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"github": {
		Name:        "GitHub",
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
		Scopes:      []string{"repo", "user:email"},
	},
	"slack": {
		Name:        "Slack",
		AuthURL:     "https://slack.com/oauth/v2/authorize",
		TokenURL:    "https://slack.com/api/oauth.v2.access",
		Scopes:      []string{"chat:write", "channels:read", "users:read"},
	},
	"microsoft": {
		Name:        "Microsoft",
		AuthURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		UserInfoURL: "https://graph.microsoft.com/v1.0/me",
		Scopes:      []string{"openid", "profile", "email", "offline_access"},
	},
	"notion": {
		Name:     "Notion",
		AuthURL:  "https://api.notion.com/v1/oauth/authorize",
		TokenURL: "https://api.notion.com/v1/oauth/token",
		Scopes:   []string{},
		AdditionalParams: map[string]string{
			"owner": "user",
		},
	},
	"dropbox": {
		Name:        "Dropbox",
		AuthURL:     "https://www.dropbox.com/oauth2/authorize",
		TokenURL:    "https://api.dropboxapi.com/oauth2/token",
		Scopes:      []string{},
		AdditionalParams: map[string]string{
			"token_access_type": "offline",
		},
	},
	"discord": {
		Name:        "Discord",
		AuthURL:     "https://discord.com/api/oauth2/authorize",
		TokenURL:    "https://discord.com/api/oauth2/token",
		UserInfoURL: "https://discord.com/api/users/@me",
		Scopes:      []string{"identify", "email", "guilds"},
	},
	"jira": {
		Name:     "Jira",
		AuthURL:  "https://auth.atlassian.com/authorize",
		TokenURL: "https://auth.atlassian.com/oauth/token",
		Scopes:   []string{"read:jira-work", "write:jira-work", "offline_access"},
		AdditionalParams: map[string]string{
			"audience":    "api.atlassian.com",
			"prompt":      "consent",
		},
	},
	"airtable": {
		Name:     "Airtable",
		AuthURL:  "https://airtable.com/oauth2/v1/authorize",
		TokenURL: "https://airtable.com/oauth2/v1/token",
		Scopes:   []string{"data.records:read", "data.records:write", "schema.bases:read"},
	},
}

// Token represents an OAuth token
type Token struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	if t.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(t.ExpiresAt.Add(-5 * time.Minute)) // 5 min buffer
}

// OAuthState represents the state during OAuth flow
type OAuthState struct {
	ID            string
	Provider      string
	UserID        string
	WorkspaceID   string
	IntegrationID string
	RedirectURI   string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	Metadata      map[string]interface{}
}

// OAuthManager manages OAuth flows
type OAuthManager struct {
	providers    map[string]*Provider
	states       map[string]*OAuthState
	tokens       map[string]*Token
	httpClient   *http.Client
	baseURL      string
	mu           sync.RWMutex
	stateStore   StateStore
	tokenStore   TokenStore
}

// StateStore interface for persisting OAuth states
type StateStore interface {
	Save(ctx context.Context, state *OAuthState) error
	Get(ctx context.Context, stateID string) (*OAuthState, error)
	Delete(ctx context.Context, stateID string) error
}

// TokenStore interface for persisting tokens
type TokenStore interface {
	Save(ctx context.Context, integrationID string, token *Token) error
	Get(ctx context.Context, integrationID string) (*Token, error)
	Delete(ctx context.Context, integrationID string) error
}

// OAuthConfig holds OAuth manager configuration
type OAuthConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	StateStore StateStore
	TokenStore TokenStore
}

// NewOAuthManager creates a new OAuth manager
func NewOAuthManager(config *OAuthConfig) *OAuthManager {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &OAuthManager{
		providers:  make(map[string]*Provider),
		states:     make(map[string]*OAuthState),
		tokens:     make(map[string]*Token),
		httpClient: httpClient,
		baseURL:    config.BaseURL,
		stateStore: config.StateStore,
		tokenStore: config.TokenStore,
	}
}

// RegisterProvider registers an OAuth provider
func (m *OAuthManager) RegisterProvider(name string, provider *Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	provider.RedirectURL = fmt.Sprintf("%s/api/v1/oauth/callback/%s", m.baseURL, name)
	m.providers[name] = provider
}

// ConfigureProvider configures a provider with client credentials
func (m *OAuthManager) ConfigureProvider(name, clientID, clientSecret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider, exists := Providers[name]
	if !exists {
		return fmt.Errorf("unknown provider: %s", name)
	}

	// Copy provider config
	p := *provider
	p.ClientID = clientID
	p.ClientSecret = clientSecret
	p.RedirectURL = fmt.Sprintf("%s/api/v1/oauth/callback/%s", m.baseURL, name)
	
	m.providers[name] = &p
	return nil
}

// GetAuthorizationURL generates the authorization URL
func (m *OAuthManager) GetAuthorizationURL(ctx context.Context, providerName string, params *AuthParams) (string, error) {
	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("provider %s not configured", providerName)
	}

	// Generate state
	stateID, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	state := &OAuthState{
		ID:            stateID,
		Provider:      providerName,
		UserID:        params.UserID,
		WorkspaceID:   params.WorkspaceID,
		IntegrationID: params.IntegrationID,
		RedirectURI:   params.RedirectURI,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
		Metadata:      params.Metadata,
	}

	// Store state
	if m.stateStore != nil {
		if err := m.stateStore.Save(ctx, state); err != nil {
			return "", fmt.Errorf("failed to save state: %w", err)
		}
	} else {
		m.mu.Lock()
		m.states[stateID] = state
		m.mu.Unlock()
	}

	// Build authorization URL
	authURL, err := url.Parse(provider.AuthURL)
	if err != nil {
		return "", fmt.Errorf("invalid auth URL: %w", err)
	}

	q := authURL.Query()
	q.Set("client_id", provider.ClientID)
	q.Set("redirect_uri", provider.RedirectURL)
	q.Set("response_type", "code")
	q.Set("state", stateID)

	// Set scopes
	scopes := provider.Scopes
	if len(params.Scopes) > 0 {
		scopes = params.Scopes
	}
	if len(scopes) > 0 {
		q.Set("scope", strings.Join(scopes, " "))
	}

	// Add additional params
	for k, v := range provider.AdditionalParams {
		q.Set(k, v)
	}

	authURL.RawQuery = q.Encode()
	return authURL.String(), nil
}

// AuthParams holds parameters for authorization
type AuthParams struct {
	UserID        string
	WorkspaceID   string
	IntegrationID string
	Scopes        []string
	RedirectURI   string
	Metadata      map[string]interface{}
}

// HandleCallback handles the OAuth callback
func (m *OAuthManager) HandleCallback(ctx context.Context, providerName, code, stateID string) (*CallbackResult, error) {
	// Validate state
	var state *OAuthState
	if m.stateStore != nil {
		var err error
		state, err = m.stateStore.Get(ctx, stateID)
		if err != nil {
			return nil, fmt.Errorf("invalid state: %w", err)
		}
		defer m.stateStore.Delete(ctx, stateID)
	} else {
		m.mu.Lock()
		var exists bool
		state, exists = m.states[stateID]
		if exists {
			delete(m.states, stateID)
		}
		m.mu.Unlock()
		if !exists {
			return nil, fmt.Errorf("invalid state")
		}
	}

	// Check expiration
	if time.Now().After(state.ExpiresAt) {
		return nil, fmt.Errorf("state expired")
	}

	// Verify provider matches
	if state.Provider != providerName {
		return nil, fmt.Errorf("provider mismatch")
	}

	// Exchange code for token
	token, err := m.exchangeCode(ctx, providerName, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Store token
	if state.IntegrationID != "" {
		if m.tokenStore != nil {
			if err := m.tokenStore.Save(ctx, state.IntegrationID, token); err != nil {
				return nil, fmt.Errorf("failed to save token: %w", err)
			}
		} else {
			m.mu.Lock()
			m.tokens[state.IntegrationID] = token
			m.mu.Unlock()
		}
	}

	return &CallbackResult{
		Token:         token,
		UserID:        state.UserID,
		WorkspaceID:   state.WorkspaceID,
		IntegrationID: state.IntegrationID,
		RedirectURI:   state.RedirectURI,
		Metadata:      state.Metadata,
	}, nil
}

// CallbackResult holds the result of OAuth callback
type CallbackResult struct {
	Token         *Token
	UserID        string
	WorkspaceID   string
	IntegrationID string
	RedirectURI   string
	Metadata      map[string]interface{}
}

func (m *OAuthManager) exchangeCode(ctx context.Context, providerName, code string) (*Token, error) {
	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not configured", providerName)
	}

	// Build token request
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)
	data.Set("redirect_uri", provider.RedirectURL)

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token exchange failed: %v", errResp)
	}

	var token Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	// Set expiration time
	if token.ExpiresIn > 0 {
		token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	}

	return &token, nil
}

// RefreshToken refreshes an OAuth token
func (m *OAuthManager) RefreshToken(ctx context.Context, providerName, integrationID string) (*Token, error) {
	// Get existing token
	var token *Token
	if m.tokenStore != nil {
		var err error
		token, err = m.tokenStore.Get(ctx, integrationID)
		if err != nil {
			return nil, fmt.Errorf("token not found: %w", err)
		}
	} else {
		m.mu.RLock()
		token = m.tokens[integrationID]
		m.mu.RUnlock()
		if token == nil {
			return nil, fmt.Errorf("token not found")
		}
	}

	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not configured", providerName)
	}

	// Build refresh request
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return nil, fmt.Errorf("token refresh failed: %v", errResp)
	}

	var newToken Token
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return nil, err
	}

	// Keep refresh token if not returned
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	// Set expiration time
	if newToken.ExpiresIn > 0 {
		newToken.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	}

	// Store new token
	if m.tokenStore != nil {
		if err := m.tokenStore.Save(ctx, integrationID, &newToken); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	} else {
		m.mu.Lock()
		m.tokens[integrationID] = &newToken
		m.mu.Unlock()
	}

	return &newToken, nil
}

// GetToken retrieves a valid token, refreshing if needed
func (m *OAuthManager) GetToken(ctx context.Context, providerName, integrationID string) (*Token, error) {
	var token *Token
	if m.tokenStore != nil {
		var err error
		token, err = m.tokenStore.Get(ctx, integrationID)
		if err != nil {
			return nil, fmt.Errorf("token not found: %w", err)
		}
	} else {
		m.mu.RLock()
		token = m.tokens[integrationID]
		m.mu.RUnlock()
		if token == nil {
			return nil, fmt.Errorf("token not found")
		}
	}

	// Check if expired and refresh
	if token.IsExpired() && token.RefreshToken != "" {
		return m.RefreshToken(ctx, providerName, integrationID)
	}

	return token, nil
}

// RevokeToken revokes an OAuth token
func (m *OAuthManager) RevokeToken(ctx context.Context, providerName, integrationID string) error {
	// Delete token from store
	if m.tokenStore != nil {
		return m.tokenStore.Delete(ctx, integrationID)
	}

	m.mu.Lock()
	delete(m.tokens, integrationID)
	m.mu.Unlock()

	return nil
}

// ListProviders returns all configured providers
func (m *OAuthManager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// InMemoryStateStore implements StateStore in memory
type InMemoryStateStore struct {
	states map[string]*OAuthState
	mu     sync.RWMutex
}

// NewInMemoryStateStore creates a new in-memory state store
func NewInMemoryStateStore() *InMemoryStateStore {
	return &InMemoryStateStore{
		states: make(map[string]*OAuthState),
	}
}

func (s *InMemoryStateStore) Save(ctx context.Context, state *OAuthState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state.ID] = state
	return nil
}

func (s *InMemoryStateStore) Get(ctx context.Context, stateID string) (*OAuthState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, exists := s.states[stateID]
	if !exists {
		return nil, fmt.Errorf("state not found")
	}
	return state, nil
}

func (s *InMemoryStateStore) Delete(ctx context.Context, stateID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, stateID)
	return nil
}

// InMemoryTokenStore implements TokenStore in memory
type InMemoryTokenStore struct {
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewInMemoryTokenStore creates a new in-memory token store
func NewInMemoryTokenStore() *InMemoryTokenStore {
	return &InMemoryTokenStore{
		tokens: make(map[string]*Token),
	}
}

func (s *InMemoryTokenStore) Save(ctx context.Context, integrationID string, token *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[integrationID] = token
	return nil
}

func (s *InMemoryTokenStore) Get(ctx context.Context, integrationID string) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, exists := s.tokens[integrationID]
	if !exists {
		return nil, fmt.Errorf("token not found")
	}
	return token, nil
}

func (s *InMemoryTokenStore) Delete(ctx context.Context, integrationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, integrationID)
	return nil
}

// UserInfo represents user info from OAuth provider
type UserInfo struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Provider string `json:"provider"`
}

// GetUserInfo fetches user info from provider
func (m *OAuthManager) GetUserInfo(ctx context.Context, providerName string, token *Token) (*UserInfo, error) {
	m.mu.RLock()
	provider, exists := m.providers[providerName]
	m.mu.RUnlock()

	if !exists || provider.UserInfoURL == "" {
		return nil, fmt.Errorf("user info not available for provider %s", providerName)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	// Map provider-specific fields to common format
	userInfo := &UserInfo{Provider: providerName}
	
	switch providerName {
	case "google", "google_sheets", "google_drive", "gmail":
		if v, ok := info["id"].(string); ok {
			userInfo.ID = v
		}
		if v, ok := info["email"].(string); ok {
			userInfo.Email = v
		}
		if v, ok := info["name"].(string); ok {
			userInfo.Name = v
		}
		if v, ok := info["picture"].(string); ok {
			userInfo.Picture = v
		}
	case "github":
		if v, ok := info["id"].(float64); ok {
			userInfo.ID = fmt.Sprintf("%.0f", v)
		}
		if v, ok := info["email"].(string); ok {
			userInfo.Email = v
		}
		if v, ok := info["name"].(string); ok {
			userInfo.Name = v
		}
		if v, ok := info["avatar_url"].(string); ok {
			userInfo.Picture = v
		}
	case "microsoft":
		if v, ok := info["id"].(string); ok {
			userInfo.ID = v
		}
		if v, ok := info["mail"].(string); ok {
			userInfo.Email = v
		}
		if v, ok := info["displayName"].(string); ok {
			userInfo.Name = v
		}
	case "discord":
		if v, ok := info["id"].(string); ok {
			userInfo.ID = v
		}
		if v, ok := info["email"].(string); ok {
			userInfo.Email = v
		}
		if v, ok := info["username"].(string); ok {
			userInfo.Name = v
		}
		if v, ok := info["avatar"].(string); ok && userInfo.ID != "" {
			userInfo.Picture = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userInfo.ID, v)
		}
	}

	return userInfo, nil
}

// CredentialID generates a credential ID for an integration
func CredentialID(userID, provider string) string {
	return fmt.Sprintf("%s:%s:%s", userID, provider, uuid.New().String()[:8])
}
