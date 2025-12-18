package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/credential/domain/model"
)

// CredentialService manages credentials and variables
type CredentialService struct {
	credentials map[string]*model.Credential
	variables   map[string]*model.Variable
	encryptKey  []byte
	mu          sync.RWMutex
}

// NewCredentialService creates a new credential service
func NewCredentialService(encryptionKey string) *CredentialService {
	key := make([]byte, 32)
	copy(key, []byte(encryptionKey))

	return &CredentialService{
		credentials: make(map[string]*model.Credential),
		variables:   make(map[string]*model.Variable),
		encryptKey:  key,
	}
}

// CreateCredential creates a new credential
func (s *CredentialService) CreateCredential(ctx context.Context, userID, orgID, name string, credType model.CredentialType, provider string, data map[string]interface{}) (*model.Credential, error) {
	cred, err := model.NewCredential(userID, orgID, name, credType, provider)
	if err != nil {
		return nil, err
	}

	// Encrypt sensitive data
	encryptedData, err := s.encryptData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt credential data: %w", err)
	}
	cred.Data = map[string]interface{}{"encrypted": encryptedData}

	s.mu.Lock()
	s.credentials[cred.ID] = cred
	s.mu.Unlock()

	return cred, nil
}

// GetCredential retrieves a credential by ID
func (s *CredentialService) GetCredential(ctx context.Context, id string) (*model.Credential, error) {
	s.mu.RLock()
	cred, ok := s.credentials[id]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.New("credential not found")
	}

	return cred, nil
}

// GetCredentialData retrieves and decrypts credential data
func (s *CredentialService) GetCredentialData(ctx context.Context, id string) (map[string]interface{}, error) {
	cred, err := s.GetCredential(ctx, id)
	if err != nil {
		return nil, err
	}

	if cred.IsExpired() {
		return nil, errors.New("credential has expired")
	}

	encryptedData, ok := cred.Data["encrypted"].(string)
	if !ok {
		return nil, errors.New("invalid credential data format")
	}

	data, err := s.decryptData(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt credential data: %w", err)
	}

	cred.MarkUsed()

	return data, nil
}

// UpdateCredential updates a credential
func (s *CredentialService) UpdateCredential(ctx context.Context, id string, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred, ok := s.credentials[id]
	if !ok {
		return errors.New("credential not found")
	}

	encryptedData, err := s.encryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credential data: %w", err)
	}

	cred.SetData(map[string]interface{}{"encrypted": encryptedData})

	return nil
}

// DeleteCredential deletes a credential
func (s *CredentialService) DeleteCredential(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.credentials[id]; !ok {
		return errors.New("credential not found")
	}

	delete(s.credentials, id)
	return nil
}

// ListCredentials lists all credentials for a user
func (s *CredentialService) ListCredentials(ctx context.Context, userID string) ([]*model.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Credential
	for _, cred := range s.credentials {
		if cred.UserID == userID {
			result = append(result, cred)
		}
	}

	return result, nil
}

// RefreshOAuth2Token refreshes an OAuth2 token
func (s *CredentialService) RefreshOAuth2Token(ctx context.Context, id string) error {
	data, err := s.GetCredentialData(ctx, id)
	if err != nil {
		return err
	}

	// Parse token data
	tokenData, _ := json.Marshal(data)
	var token model.OAuth2Token
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return fmt.Errorf("invalid OAuth2 token data: %w", err)
	}

	if !token.NeedsRefresh() {
		return nil
	}

	// In a real implementation, you would call the OAuth2 provider's token refresh endpoint
	// For now, we'll just extend the expiration
	token.ExpiresAt = time.Now().Add(1 * time.Hour)

	// Update the credential
	newData := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_type":    token.TokenType,
		"expires_at":    token.ExpiresAt,
		"scope":         token.Scope,
	}

	return s.UpdateCredential(ctx, id, newData)
}

// Variable Management

// CreateVariable creates a new variable
func (s *CredentialService) CreateVariable(ctx context.Context, userID, key, value string, varType model.VariableType, scope model.VariableScope) (*model.Variable, error) {
	variable, err := model.NewVariable(userID, key, value, varType, scope)
	if err != nil {
		return nil, err
	}

	// Encrypt if sensitive
	if variable.Sensitive {
		encrypted, err := s.encrypt([]byte(value))
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt variable: %w", err)
		}
		variable.Value = encrypted
	}

	s.mu.Lock()
	s.variables[variable.ID] = variable
	s.mu.Unlock()

	return variable, nil
}

// GetVariable retrieves a variable by ID
func (s *CredentialService) GetVariable(ctx context.Context, id string) (*model.Variable, error) {
	s.mu.RLock()
	variable, ok := s.variables[id]
	s.mu.RUnlock()

	if !ok {
		return nil, errors.New("variable not found")
	}

	return variable, nil
}

// GetVariableValue retrieves and decrypts a variable value
func (s *CredentialService) GetVariableValue(ctx context.Context, id string) (string, error) {
	variable, err := s.GetVariable(ctx, id)
	if err != nil {
		return "", err
	}

	if variable.Sensitive {
		decrypted, err := s.decrypt(variable.Value)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt variable: %w", err)
		}
		return string(decrypted), nil
	}

	return variable.Value, nil
}

// UpdateVariable updates a variable
func (s *CredentialService) UpdateVariable(ctx context.Context, id, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	variable, ok := s.variables[id]
	if !ok {
		return errors.New("variable not found")
	}

	if variable.Sensitive {
		encrypted, err := s.encrypt([]byte(value))
		if err != nil {
			return fmt.Errorf("failed to encrypt variable: %w", err)
		}
		value = encrypted
	}

	variable.Update(value)
	return nil
}

// DeleteVariable deletes a variable
func (s *CredentialService) DeleteVariable(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.variables[id]; !ok {
		return errors.New("variable not found")
	}

	delete(s.variables, id)
	return nil
}

// ListVariables lists all variables for a user
func (s *CredentialService) ListVariables(ctx context.Context, userID string, scope model.VariableScope) ([]*model.Variable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Variable
	for _, v := range s.variables {
		if v.UserID == userID && v.Scope == scope {
			result = append(result, v)
		}
	}

	return result, nil
}

// GetWorkflowVariables retrieves all variables for a workflow
func (s *CredentialService) GetWorkflowVariables(ctx context.Context, workflowID string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string)
	for _, v := range s.variables {
		if v.WorkflowID != nil && *v.WorkflowID == workflowID {
			value := v.Value
			if v.Sensitive {
				decrypted, err := s.decrypt(v.Value)
				if err != nil {
					return nil, err
				}
				value = string(decrypted)
			}
			result[v.Key] = value
		}
	}

	return result, nil
}

// Encryption helpers

func (s *CredentialService) encryptData(data map[string]interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return s.encrypt(jsonData)
}

func (s *CredentialService) decryptData(encrypted string) (map[string]interface{}, error) {
	decrypted, err := s.decrypt(encrypted)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(decrypted, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (s *CredentialService) encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *CredentialService) decrypt(encrypted string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
