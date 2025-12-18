// Package oauth provides OAuth HTTP handlers
package oauth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// OAuthHandler handles OAuth HTTP routes
type OAuthHandler struct {
	manager *OAuthManager
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(manager *OAuthManager) *OAuthHandler {
	return &OAuthHandler{manager: manager}
}

// RegisterRoutes registers OAuth routes on a mux
func (h *OAuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/oauth/providers", h.ListProviders)
	mux.HandleFunc("/api/v1/oauth/authorize/", h.Authorize)
	mux.HandleFunc("/api/v1/oauth/callback/", h.Callback)
	mux.HandleFunc("/api/v1/oauth/refresh/", h.Refresh)
	mux.HandleFunc("/api/v1/oauth/revoke/", h.Revoke)
	mux.HandleFunc("/api/v1/oauth/token/", h.GetToken)
}

// ListProviders returns available OAuth providers
func (h *OAuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := h.manager.ListProviders()
	
	response := make([]map[string]interface{}, 0, len(providers))
	for _, name := range providers {
		if p, exists := Providers[name]; exists {
			response = append(response, map[string]interface{}{
				"name":        name,
				"displayName": p.Name,
				"scopes":      p.Scopes,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": response,
	})
}

// AuthorizeRequest holds authorization request
type AuthorizeRequest struct {
	UserID        string                 `json:"userId"`
	WorkspaceID   string                 `json:"workspaceId"`
	IntegrationID string                 `json:"integrationId"`
	Scopes        []string               `json:"scopes,omitempty"`
	RedirectURI   string                 `json:"redirectUri,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Authorize initiates OAuth authorization
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract provider from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}
	provider := parts[len(parts)-1]

	var params AuthParams
	
	if r.Method == http.MethodPost {
		var req AuthorizeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		params = AuthParams{
			UserID:        req.UserID,
			WorkspaceID:   req.WorkspaceID,
			IntegrationID: req.IntegrationID,
			Scopes:        req.Scopes,
			RedirectURI:   req.RedirectURI,
			Metadata:      req.Metadata,
		}
	} else {
		// GET request - params from query string
		params = AuthParams{
			UserID:        r.URL.Query().Get("userId"),
			WorkspaceID:   r.URL.Query().Get("workspaceId"),
			IntegrationID: r.URL.Query().Get("integrationId"),
			RedirectURI:   r.URL.Query().Get("redirectUri"),
		}
		if scopes := r.URL.Query().Get("scopes"); scopes != "" {
			params.Scopes = strings.Split(scopes, ",")
		}
	}

	authURL, err := h.manager.GetAuthorizationURL(r.Context(), provider, &params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// For GET requests, redirect directly
	if r.Method == http.MethodGet {
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		return
	}

	// For POST requests, return URL
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"authorizationUrl": authURL,
	})
}

// Callback handles OAuth callback
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	// Extract provider from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}
	provider := parts[len(parts)-1]

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// Check for error from provider
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":       errorParam,
			"description": errorDesc,
		})
		return
	}

	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	result, err := h.manager.HandleCallback(r.Context(), provider, code, state)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// If redirect URI is set, redirect with success
	if result.RedirectURI != "" {
		redirectURL := result.RedirectURI
		if strings.Contains(redirectURL, "?") {
			redirectURL += "&"
		} else {
			redirectURL += "?"
		}
		redirectURL += "success=true"
		if result.IntegrationID != "" {
			redirectURL += "&integrationId=" + result.IntegrationID
		}
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"integrationId": result.IntegrationID,
		"userId":        result.UserID,
		"workspaceId":   result.WorkspaceID,
		"expiresAt":     result.Token.ExpiresAt,
	})
}

// RefreshRequest holds refresh request
type RefreshRequest struct {
	IntegrationID string `json:"integrationId"`
}

// Refresh refreshes an OAuth token
func (h *OAuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract provider from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}
	provider := parts[len(parts)-1]

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.manager.RefreshToken(r.Context(), provider, req.IntegrationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"expiresAt": token.ExpiresAt,
	})
}

// RevokeRequest holds revoke request
type RevokeRequest struct {
	IntegrationID string `json:"integrationId"`
}

// Revoke revokes an OAuth token
func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract provider from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}
	provider := parts[len(parts)-1]

	var req RevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.manager.RevokeToken(r.Context(), provider, req.IntegrationID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetToken retrieves a valid token
func (h *OAuthHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract provider from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}
	provider := parts[len(parts)-1]

	integrationID := r.URL.Query().Get("integrationId")
	if integrationID == "" {
		http.Error(w, "Integration ID required", http.StatusBadRequest)
		return
	}

	token, err := h.manager.GetToken(r.Context(), provider, integrationID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accessToken": token.AccessToken,
		"tokenType":   token.TokenType,
		"expiresAt":   token.ExpiresAt,
		"scope":       token.Scope,
	})
}
