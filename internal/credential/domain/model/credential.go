package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// CredentialType represents the type of credential
type CredentialType string

const (
	CredentialTypeAPIKey       CredentialType = "api_key"
	CredentialTypeOAuth2       CredentialType = "oauth2"
	CredentialTypeBasicAuth    CredentialType = "basic_auth"
	CredentialTypeBearerToken  CredentialType = "bearer_token"
	CredentialTypeSSHKey       CredentialType = "ssh_key"
	CredentialTypeDatabaseConn CredentialType = "database_connection"
	CredentialTypeCustom       CredentialType = "custom"
)

// Credential represents a stored credential
type Credential struct {
	ID             string
	UserID         string
	OrganizationID string
	Name           string
	Description    string
	Type           CredentialType
	Provider       string
	Data           map[string]interface{} // Encrypted in storage
	Metadata       map[string]string
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
}

// NewCredential creates a new credential
func NewCredential(userID, orgID, name string, credType CredentialType, provider string) (*Credential, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if name == "" {
		return nil, errors.New("credential name is required")
	}

	return &Credential{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: orgID,
		Name:           name,
		Type:           credType,
		Provider:       provider,
		Data:           make(map[string]interface{}),
		Metadata:       make(map[string]string),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Version:        1,
	}, nil
}

// SetData sets the credential data (will be encrypted)
func (c *Credential) SetData(data map[string]interface{}) {
	c.Data = data
	c.UpdatedAt = time.Now()
	c.Version++
}

// MarkUsed updates the last used timestamp
func (c *Credential) MarkUsed() {
	now := time.Now()
	c.LastUsedAt = &now
}

// IsExpired checks if the credential has expired
func (c *Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

// SetExpiration sets the expiration time
func (c *Credential) SetExpiration(expiresAt time.Time) {
	c.ExpiresAt = &expiresAt
	c.UpdatedAt = time.Now()
}

// OAuth2Token represents OAuth2 credential data
type OAuth2Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        []string  `json:"scope"`
}

// IsAccessTokenExpired checks if the access token is expired
func (t *OAuth2Token) IsAccessTokenExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// NeedsRefresh checks if the token needs to be refreshed
func (t *OAuth2Token) NeedsRefresh() bool {
	// Refresh 5 minutes before expiration
	return time.Now().Add(5 * time.Minute).After(t.ExpiresAt)
}

// APIKeyCredential represents API key credential data
type APIKeyCredential struct {
	Key       string `json:"key"`
	Secret    string `json:"secret,omitempty"`
	KeyHeader string `json:"key_header,omitempty"`
}

// BasicAuthCredential represents basic auth credential data
type BasicAuthCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SSHKeyCredential represents SSH key credential data
type SSHKeyCredential struct {
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
	Passphrase string `json:"passphrase,omitempty"`
}

// DatabaseCredential represents database connection credential data
type DatabaseCredential struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"ssl_mode,omitempty"`
}

// Variable represents a user-defined variable
type Variable struct {
	ID             string
	UserID         string
	OrganizationID string
	WorkflowID     *string // nil means global
	Key            string
	Value          string // Encrypted if sensitive
	Description    string
	Type           VariableType
	Sensitive      bool
	Scope          VariableScope
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// VariableType represents the type of variable value
type VariableType string

const (
	VariableTypeString  VariableType = "string"
	VariableTypeNumber  VariableType = "number"
	VariableTypeBoolean VariableType = "boolean"
	VariableTypeJSON    VariableType = "json"
	VariableTypeSecret  VariableType = "secret"
)

// VariableScope represents the scope of a variable
type VariableScope string

const (
	VariableScopeGlobal       VariableScope = "global"
	VariableScopeOrganization VariableScope = "organization"
	VariableScopeWorkflow     VariableScope = "workflow"
)

// NewVariable creates a new variable
func NewVariable(userID, key, value string, varType VariableType, scope VariableScope) (*Variable, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if key == "" {
		return nil, errors.New("variable key is required")
	}

	return &Variable{
		ID:        uuid.New().String(),
		UserID:    userID,
		Key:       key,
		Value:     value,
		Type:      varType,
		Scope:     scope,
		Sensitive: varType == VariableTypeSecret,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// Update updates the variable value
func (v *Variable) Update(value string) {
	v.Value = value
	v.UpdatedAt = time.Now()
}

// SetWorkflowScope sets the variable to workflow scope
func (v *Variable) SetWorkflowScope(workflowID string) {
	v.WorkflowID = &workflowID
	v.Scope = VariableScopeWorkflow
	v.UpdatedAt = time.Now()
}
