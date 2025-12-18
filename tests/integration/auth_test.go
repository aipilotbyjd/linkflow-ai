// +build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock auth service for integration testing
type AuthService struct {
	users  map[string]*User
	tokens map[string]*Session
}

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	User         struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	} `json:"user"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func NewAuthService() *AuthService {
	return &AuthService{
		users:  make(map[string]*User),
		tokens: make(map[string]*Session),
	}
}

func (s *AuthService) Register(email, password, name string) (*User, error) {
	if _, exists := s.users[email]; exists {
		return nil, assert.AnError
	}
	user := &User{
		ID:           "user-" + time.Now().Format("20060102150405"),
		Email:        email,
		PasswordHash: "hashed:" + password,
		Name:         name,
		CreatedAt:    time.Now(),
	}
	s.users[email] = user
	return user, nil
}

func (s *AuthService) Login(email, password string) (*Session, error) {
	user, exists := s.users[email]
	if !exists {
		return nil, assert.AnError
	}
	if user.PasswordHash != "hashed:"+password {
		return nil, assert.AnError
	}
	session := &Session{
		Token:     "token-" + time.Now().Format("20060102150405"),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	s.tokens[session.Token] = session
	return session, nil
}

func (s *AuthService) ValidateToken(token string) (*Session, error) {
	session, exists := s.tokens[token]
	if !exists {
		return nil, assert.AnError
	}
	if time.Now().After(session.ExpiresAt) {
		delete(s.tokens, token)
		return nil, assert.AnError
	}
	return session, nil
}

func (s *AuthService) Logout(token string) error {
	delete(s.tokens, token)
	return nil
}

func (s *AuthService) GetUser(email string) *User {
	return s.users[email]
}

// HTTP Handlers
func (s *AuthService) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := s.Register(req.Email, req.Password, req.Name)
	if err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func (s *AuthService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	session, err := s.Login(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	user := s.GetUser(req.Email)
	resp := LoginResponse{
		AccessToken:  session.Token,
		RefreshToken: "refresh-" + session.Token,
		ExpiresIn:    3600,
	}
	resp.User.ID = user.ID
	resp.User.Email = user.Email
	resp.User.Name = user.Name

	json.NewEncoder(w).Encode(resp)
}

// Integration Tests
func TestAuthService_Registration(t *testing.T) {
	svc := NewAuthService()
	server := httptest.NewServer(http.HandlerFunc(svc.HandleRegister))
	defer server.Close()

	t.Run("successful registration", func(t *testing.T) {
		body := RegisterRequest{
			Email:    "test@example.com",
			Password: "password123",
			Name:     "Test User",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		assert.Equal(t, "test@example.com", result["email"])
		assert.NotEmpty(t, result["id"])
	})

	t.Run("duplicate email fails", func(t *testing.T) {
		body := RegisterRequest{
			Email:    "test@example.com",
			Password: "password456",
			Name:     "Another User",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestAuthService_Login(t *testing.T) {
	svc := NewAuthService()
	_, _ = svc.Register("user@example.com", "correctpassword", "Test User")

	server := httptest.NewServer(http.HandlerFunc(svc.HandleLogin))
	defer server.Close()

	t.Run("successful login", func(t *testing.T) {
		body := LoginRequest{
			Email:    "user@example.com",
			Password: "correctpassword",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result LoginResponse
		json.NewDecoder(resp.Body).Decode(&result)
		assert.NotEmpty(t, result.AccessToken)
		assert.NotEmpty(t, result.RefreshToken)
		assert.Equal(t, 3600, result.ExpiresIn)
		assert.Equal(t, "user@example.com", result.User.Email)
	})

	t.Run("wrong password fails", func(t *testing.T) {
		body := LoginRequest{
			Email:    "user@example.com",
			Password: "wrongpassword",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("non-existent user fails", func(t *testing.T) {
		body := LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL, "application/json", bytes.NewBuffer(jsonBody))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAuthService_TokenValidation(t *testing.T) {
	svc := NewAuthService()
	_, _ = svc.Register("user@example.com", "password", "Test User")
	session, _ := svc.Login("user@example.com", "password")

	t.Run("valid token", func(t *testing.T) {
		validSession, err := svc.ValidateToken(session.Token)
		assert.NoError(t, err)
		assert.Equal(t, session.UserID, validSession.UserID)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.ValidateToken("invalid-token")
		assert.Error(t, err)
	})

	t.Run("logout invalidates token", func(t *testing.T) {
		err := svc.Logout(session.Token)
		assert.NoError(t, err)

		_, err = svc.ValidateToken(session.Token)
		assert.Error(t, err)
	})
}
