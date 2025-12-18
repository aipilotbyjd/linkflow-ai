// Package handlers provides HTTP handlers for the credential service
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/linkflow-ai/linkflow-ai/internal/credential/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/credential/domain/model"
)

// CredentialHandler handles credential HTTP requests
type CredentialHandler struct {
	service *service.CredentialService
}

// NewCredentialHandler creates a new credential handler
func NewCredentialHandler(svc *service.CredentialService) *CredentialHandler {
	return &CredentialHandler{service: svc}
}

// RegisterRoutes registers credential routes
func (h *CredentialHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/credentials", h.handleCredentials)
	mux.HandleFunc("/api/v1/credentials/", h.handleCredential)
	mux.HandleFunc("/api/v1/credentials/test", h.testCredential)
	mux.HandleFunc("/api/v1/credentials/types", h.listCredentialTypes)
}

func (h *CredentialHandler) handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listCredentials(w, r)
	case http.MethodPost:
		h.createCredential(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *CredentialHandler) handleCredential(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/credentials/"):]
	if id == "" {
		http.Error(w, "Credential ID required", http.StatusBadRequest)
		return
	}

	// Check for action paths
	if idx := findIndex(id, "/"); idx != -1 {
		credID := id[:idx]
		action := id[idx+1:]

		switch action {
		case "test":
			h.testCredentialByID(w, r, credID)
		case "refresh":
			h.refreshCredential(w, r, credID)
		default:
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getCredential(w, r, id)
	case http.MethodPut:
		h.updateCredential(w, r, id)
	case http.MethodDelete:
		h.deleteCredential(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateCredentialRequest represents credential creation request
type CreateCredentialRequest struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Service     string                 `json:"service"`
	Credentials map[string]interface{} `json:"credentials"`
	TenantID    string                 `json:"tenantId"`
}

// CredentialResponse represents credential response
type CredentialResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Service   string `json:"service"`
	Status    string `json:"status"`
	TenantID  string `json:"tenantId"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

func (h *CredentialHandler) createCredential(w http.ResponseWriter, r *http.Request) {
	var req CreateCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user/tenant from headers
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}
	orgID := r.Header.Get("X-Tenant-ID")

	credType := model.CredentialType(req.Type)
	cred, err := h.service.CreateCredential(r.Context(), userID, orgID, req.Name, credType, req.Service, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toCredentialResponseFromModel(cred))
}

func (h *CredentialHandler) getCredential(w http.ResponseWriter, r *http.Request, id string) {
	cred, err := h.service.GetCredential(r.Context(), id)
	if err != nil {
		http.Error(w, "Credential not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCredentialResponseFromModel(cred))
}

func (h *CredentialHandler) updateCredential(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Name        string                 `json:"name"`
		Credentials map[string]interface{} `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	err := h.service.UpdateCredential(r.Context(), id, req.Credentials)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Re-fetch to return updated credential
	cred, _ := h.service.GetCredential(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCredentialResponseFromModel(cred))
}

func (h *CredentialHandler) deleteCredential(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.service.DeleteCredential(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CredentialHandler) listCredentials(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	creds, err := h.service.ListCredentials(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]CredentialResponse, len(creds))
	for i, c := range creds {
		response[i] = toCredentialResponseFromModel(c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": response,
		"total": len(creds),
	})
}

func (h *CredentialHandler) testCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type        string                 `json:"type"`
		Service     string                 `json:"service"`
		Credentials map[string]interface{} `json:"credentials"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple validation - credentials exist
	valid := len(req.Credentials) > 0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   valid,
		"message": "Credentials tested successfully",
	})
}

func (h *CredentialHandler) testCredentialByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if credential exists
	_, err := h.service.GetCredential(r.Context(), id)
	valid := err == nil

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   valid,
		"message": "Credential tested successfully",
	})
}

func (h *CredentialHandler) refreshCredential(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.service.RefreshOAuth2Token(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cred, _ := h.service.GetCredential(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toCredentialResponseFromModel(cred))
}

// CredentialTypeResponse represents credential type response
type CredentialTypeResponse struct {
	Type        string   `json:"type"`
	Service     string   `json:"service"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Fields      []Field  `json:"fields"`
	AuthURL     string   `json:"authUrl,omitempty"`
}

// Field represents a credential field
type Field struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder,omitempty"`
}

func (h *CredentialHandler) listCredentialTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return supported credential types
	types := []CredentialTypeResponse{
		{
			Type:        "api_key",
			Service:     "generic",
			Name:        "API Key",
			Description: "Generic API key authentication",
			Fields: []Field{
				{Name: "api_key", Type: "password", Required: true, Description: "API Key"},
			},
		},
		{
			Type:        "oauth2",
			Service:     "google",
			Name:        "Google OAuth2",
			Description: "Google OAuth2 authentication",
			AuthURL:     "/auth/google/connect",
			Fields:      []Field{},
		},
		{
			Type:        "oauth2",
			Service:     "github",
			Name:        "GitHub OAuth2",
			Description: "GitHub OAuth2 authentication",
			AuthURL:     "/auth/github/connect",
			Fields:      []Field{},
		},
		{
			Type:        "basic",
			Service:     "generic",
			Name:        "Basic Auth",
			Description: "Username and password authentication",
			Fields: []Field{
				{Name: "username", Type: "text", Required: true, Description: "Username"},
				{Name: "password", Type: "password", Required: true, Description: "Password"},
			},
		},
		{
			Type:        "bearer",
			Service:     "generic",
			Name:        "Bearer Token",
			Description: "Bearer token authentication",
			Fields: []Field{
				{Name: "token", Type: "password", Required: true, Description: "Bearer Token"},
			},
		},
		{
			Type:        "api_key",
			Service:     "openai",
			Name:        "OpenAI API Key",
			Description: "OpenAI API authentication",
			Fields: []Field{
				{Name: "api_key", Type: "password", Required: true, Description: "OpenAI API Key"},
			},
		},
		{
			Type:        "api_key",
			Service:     "slack",
			Name:        "Slack Bot Token",
			Description: "Slack Bot authentication",
			Fields: []Field{
				{Name: "bot_token", Type: "password", Required: true, Description: "Bot User OAuth Token"},
			},
		},
		{
			Type:        "webhook",
			Service:     "generic",
			Name:        "Webhook",
			Description: "Webhook with optional secret",
			Fields: []Field{
				{Name: "url", Type: "url", Required: true, Description: "Webhook URL"},
				{Name: "secret", Type: "password", Required: false, Description: "Webhook Secret"},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": types,
		"total": len(types),
	})
}

func toCredentialResponseFromModel(c *model.Credential) CredentialResponse {
	status := "active"
	if c.IsExpired() {
		status = "expired"
	}
	resp := CredentialResponse{
		ID:        c.ID,
		Name:      c.Name,
		Type:      string(c.Type),
		Service:   c.Provider,
		Status:    status,
		TenantID:  c.OrganizationID,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if c.ExpiresAt != nil {
		resp.ExpiresAt = c.ExpiresAt.Format("2006-01-02T15:04:05Z")
	}
	return resp
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
