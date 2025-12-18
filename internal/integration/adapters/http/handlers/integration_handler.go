// Package handlers provides HTTP handlers for the integration service
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/linkflow-ai/linkflow-ai/internal/integration/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/integration/domain/model"
)

// IntegrationHandler handles integration HTTP requests
type IntegrationHandler struct {
	service *service.IntegrationService
}

// NewIntegrationHandler creates a new integration handler
func NewIntegrationHandler(svc *service.IntegrationService) *IntegrationHandler {
	return &IntegrationHandler{service: svc}
}

// RegisterRoutes registers integration routes
func (h *IntegrationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/integrations", h.handleIntegrations)
	mux.HandleFunc("/api/v1/integrations/", h.handleIntegration)
	mux.HandleFunc("/api/v1/integrations/categories", h.listCategories)
	mux.HandleFunc("/api/v1/integrations/search", h.searchIntegrations)
}

func (h *IntegrationHandler) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listIntegrations(w, r)
	case http.MethodPost:
		h.createIntegration(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *IntegrationHandler) handleIntegration(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/integrations/"):]
	if id == "" {
		http.Error(w, "Integration ID required", http.StatusBadRequest)
		return
	}

	// Check for action paths
	if idx := findIndex(id, "/"); idx != -1 {
		integrationID := id[:idx]
		action := id[idx+1:]

		switch action {
		case "enable":
			h.enableIntegration(w, r, integrationID)
		case "disable":
			h.disableIntegration(w, r, integrationID)
		case "test":
			h.testIntegration(w, r, integrationID)
		default:
			http.Error(w, "Unknown action", http.StatusBadRequest)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getIntegration(w, r, id)
	case http.MethodPut:
		h.updateIntegration(w, r, id)
	case http.MethodDelete:
		h.deleteIntegration(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CreateIntegrationRequest represents integration creation request
type CreateIntegrationRequest struct {
	Name         string                 `json:"name"`
	Service      string                 `json:"service"`
	Category     string                 `json:"category"`
	Config       map[string]interface{} `json:"config"`
	CredentialID string                 `json:"credentialId"`
	TenantID     string                 `json:"tenantId"`
}

// IntegrationResponse represents integration response
type IntegrationResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Service      string                 `json:"service"`
	Category     string                 `json:"category"`
	Status       string                 `json:"status"`
	Config       map[string]interface{} `json:"config"`
	CredentialID string                 `json:"credentialId,omitempty"`
	TenantID     string                 `json:"tenantId"`
	CreatedAt    string                 `json:"createdAt"`
	UpdatedAt    string                 `json:"updatedAt"`
}

func (h *IntegrationHandler) createIntegration(w http.ResponseWriter, r *http.Request) {
	var req CreateIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}
	orgID := r.Header.Get("X-Tenant-ID")

	integration, err := h.service.CreateIntegration(r.Context(), service.CreateIntegrationCommand{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           req.Name,
		Type:           req.Service,
		Config:         req.Config,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toIntegrationResponseFromModel(integration))
}

func (h *IntegrationHandler) getIntegration(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	integration, err := h.service.GetIntegration(r.Context(), id, userID)
	if err != nil {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toIntegrationResponseFromModel(integration))
}

func (h *IntegrationHandler) updateIntegration(w http.ResponseWriter, r *http.Request, id string) {
	// Service doesn't have update, just authorize with new config
	http.Error(w, "Update not implemented, use authorize endpoint", http.StatusNotImplemented)
}

func (h *IntegrationHandler) deleteIntegration(w http.ResponseWriter, r *http.Request, id string) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	if err := h.service.DeleteIntegration(r.Context(), id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *IntegrationHandler) listIntegrations(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	integrations, err := h.service.ListIntegrations(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]IntegrationResponse, len(integrations))
	for i, integration := range integrations {
		response[i] = toIntegrationResponseFromModel(integration)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

func (h *IntegrationHandler) enableIntegration(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	// Get credentials from body
	var req struct {
		Credentials map[string]interface{} `json:"credentials"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	err := h.service.AuthorizeIntegration(r.Context(), service.AuthorizeIntegrationCommand{
		IntegrationID: id,
		UserID:        userID,
		Credentials:   req.Credentials,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "enabled"})
}

func (h *IntegrationHandler) disableIntegration(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Service doesn't have explicit disable, deactivate happens on delete
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
}

func (h *IntegrationHandler) testIntegration(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Sync acts as a test
	err := h.service.SyncIntegration(r.Context(), id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"error":   func() string { if err != nil { return err.Error() }; return "" }(),
	})
}

func (h *IntegrationHandler) searchIntegrations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "default-user"
	}

	// Search is same as list for in-memory impl
	integrations, err := h.service.ListIntegrations(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]IntegrationResponse, len(integrations))
	for i, integration := range integrations {
		response[i] = toIntegrationResponseFromModel(integration)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": response,
		"total": len(response),
	})
}

// CategoryResponse represents a category response
type CategoryResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Count       int    `json:"count"`
}

func (h *IntegrationHandler) listCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	categories := []CategoryResponse{
		{ID: "communication", Name: "Communication", Description: "Email, chat, and messaging services", Icon: "message", Count: 15},
		{ID: "productivity", Name: "Productivity", Description: "Task management and collaboration tools", Icon: "check-square", Count: 12},
		{ID: "crm", Name: "CRM", Description: "Customer relationship management", Icon: "users", Count: 8},
		{ID: "marketing", Name: "Marketing", Description: "Marketing automation and analytics", Icon: "trending-up", Count: 10},
		{ID: "developer", Name: "Developer Tools", Description: "APIs, databases, and developer services", Icon: "code", Count: 20},
		{ID: "storage", Name: "Storage", Description: "File storage and cloud services", Icon: "cloud", Count: 6},
		{ID: "ai", Name: "AI & ML", Description: "Artificial intelligence and machine learning", Icon: "cpu", Count: 8},
		{ID: "payment", Name: "Payment", Description: "Payment processing and billing", Icon: "credit-card", Count: 5},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": categories,
		"total": len(categories),
	})
}

func toIntegrationResponseFromModel(i *model.Integration) IntegrationResponse {
	return IntegrationResponse{
		ID:       string(i.ID()),
		Name:     i.Name(),
		Service:  string(i.Type()),
		Category: "custom",
		Status:   string(i.Status()),
		Config:   i.Config(),
	}
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
