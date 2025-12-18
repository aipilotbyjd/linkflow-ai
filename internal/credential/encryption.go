// Package credential provides credential encryption
package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// Encryptor handles credential encryption/decryption
type Encryptor struct {
	key []byte
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	Key          string // Base64 encoded key or passphrase
	KeyType      string // "raw", "passphrase"
	Salt         string // For passphrase derivation
	Iterations   int    // PBKDF2 iterations
}

// DefaultEncryptionConfig returns default encryption config
func DefaultEncryptionConfig() *EncryptionConfig {
	return &EncryptionConfig{
		KeyType:    "passphrase",
		Iterations: 100000,
	}
}

// NewEncryptor creates a new encryptor
func NewEncryptor(config *EncryptionConfig) (*Encryptor, error) {
	var key []byte

	switch config.KeyType {
	case "raw":
		var err error
		key, err = base64.StdEncoding.DecodeString(config.Key)
		if err != nil {
			return nil, fmt.Errorf("invalid key: %w", err)
		}
	case "passphrase":
		salt := []byte(config.Salt)
		if len(salt) == 0 {
			salt = []byte("linkflow-default-salt")
		}
		key = pbkdf2.Key([]byte(config.Key), salt, config.Iterations, 32, sha256.New)
	default:
		return nil, fmt.Errorf("unknown key type: %s", config.KeyType)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256")
	}

	return &Encryptor{key: key}, nil
}

// Encrypt encrypts data using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64 encoded result
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts a base64 encoded string
func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("invalid base64: %w", err)
	}
	plaintext, err := e.Decrypt(data)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptJSON encrypts a JSON-serializable object
func (e *Encryptor) EncryptJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return e.EncryptString(string(data))
}

// DecryptJSON decrypts and unmarshals JSON
func (e *Encryptor) DecryptJSON(ciphertext string, v interface{}) error {
	plaintext, err := e.DecryptString(ciphertext)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(plaintext), v)
}

// CredentialData represents encrypted credential data
type CredentialData struct {
	Type       string                 `json:"type"`
	Data       map[string]interface{} `json:"data"`
	Encrypted  string                 `json:"encrypted,omitempty"`
	IsEncrypted bool                  `json:"isEncrypted"`
}

// CredentialEncryptionService handles credential encryption
type CredentialEncryptionService struct {
	encryptor *Encryptor
}

// NewCredentialEncryptionService creates a new credential encryption service
func NewCredentialEncryptionService(encryptor *Encryptor) *CredentialEncryptionService {
	return &CredentialEncryptionService{encryptor: encryptor}
}

// EncryptCredential encrypts credential data
func (s *CredentialEncryptionService) EncryptCredential(cred *CredentialData) error {
	if cred.IsEncrypted {
		return nil
	}

	// Sensitive fields to encrypt
	sensitiveFields := map[string]bool{
		"password":     true,
		"secret":       true,
		"api_key":      true,
		"apiKey":       true,
		"access_token": true,
		"accessToken":  true,
		"refresh_token": true,
		"refreshToken": true,
		"private_key":  true,
		"privateKey":   true,
		"client_secret": true,
		"clientSecret": true,
	}

	for key, value := range cred.Data {
		if sensitiveFields[key] {
			if strVal, ok := value.(string); ok {
				encrypted, err := s.encryptor.EncryptString(strVal)
				if err != nil {
					return fmt.Errorf("failed to encrypt %s: %w", key, err)
				}
				cred.Data[key] = "encrypted:" + encrypted
			}
		}
	}

	cred.IsEncrypted = true
	return nil
}

// DecryptCredential decrypts credential data
func (s *CredentialEncryptionService) DecryptCredential(cred *CredentialData) error {
	if !cred.IsEncrypted {
		return nil
	}

	for key, value := range cred.Data {
		if strVal, ok := value.(string); ok {
			if len(strVal) > 10 && strVal[:10] == "encrypted:" {
				decrypted, err := s.encryptor.DecryptString(strVal[10:])
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %w", key, err)
				}
				cred.Data[key] = decrypted
			}
		}
	}

	cred.IsEncrypted = false
	return nil
}

// GenerateKey generates a new encryption key
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// HashCredential creates a hash for credential comparison
func HashCredential(data map[string]interface{}) string {
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// MaskCredential masks sensitive fields for display
func MaskCredential(data map[string]interface{}) map[string]interface{} {
	sensitiveFields := map[string]bool{
		"password": true, "secret": true, "api_key": true, "apiKey": true,
		"access_token": true, "accessToken": true, "refresh_token": true,
		"refreshToken": true, "private_key": true, "privateKey": true,
		"client_secret": true, "clientSecret": true,
	}

	masked := make(map[string]interface{})
	for key, value := range data {
		if sensitiveFields[key] {
			if strVal, ok := value.(string); ok && len(strVal) > 0 {
				if len(strVal) > 4 {
					masked[key] = "****" + strVal[len(strVal)-4:]
				} else {
					masked[key] = "****"
				}
			} else {
				masked[key] = "****"
			}
		} else {
			masked[key] = value
		}
	}
	return masked
}
