// Package handlers provides HTTP handlers for the config service
package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// ConfigHandler handles configuration-related HTTP requests
type ConfigHandler struct {
	// Add service dependencies here
}

// NewConfigHandler creates a new config handler
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// RegisterRoutes registers config routes
func (h *ConfigHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/configs", h.HandleConfigs)
	mux.HandleFunc("/api/v1/configs/global", h.HandleGlobalConfig)
	mux.HandleFunc("/api/v1/configs/tenant", h.HandleTenantConfig)
	mux.HandleFunc("/api/v1/configs/feature-flags", h.HandleFeatureFlags)
	mux.HandleFunc("/api/v1/configs/history", h.GetHistory)
}

// Configuration represents a configuration entry
type Configuration struct {
	ID          string          `json:"id"`
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Scope       string          `json:"scope"` // global, tenant, user
	ScopeID     string          `json:"scopeId,omitempty"`
	Description string          `json:"description,omitempty"`
	Type        string          `json:"type"` // string, number, boolean, json
	Encrypted   bool            `json:"encrypted"`
	Version     int             `json:"version"`
	UpdatedBy   string          `json:"updatedBy"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	CreatedAt   time.Time       `json:"createdAt"`
}

// HandleConfigs handles configuration CRUD operations
func (h *ConfigHandler) HandleConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listConfigs(w, r)
	case http.MethodPost:
		h.createConfig(w, r)
	case http.MethodPut:
		h.updateConfig(w, r)
	case http.MethodDelete:
		h.deleteConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ConfigHandler) listConfigs(w http.ResponseWriter, r *http.Request) {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "global"
	}

	// Mock configuration data
	configs := []Configuration{
		{
			ID:          "cfg-001",
			Key:         "rate_limit.requests_per_minute",
			Value:       json.RawMessage(`100`),
			Scope:       "global",
			Description: "Maximum requests per minute per user",
			Type:        "number",
			Version:     3,
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:          "cfg-002",
			Key:         "execution.timeout_seconds",
			Value:       json.RawMessage(`300`),
			Scope:       "global",
			Description: "Default execution timeout in seconds",
			Type:        "number",
			Version:     1,
			UpdatedAt:   time.Now().Add(-7 * 24 * time.Hour),
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:          "cfg-003",
			Key:         "workflow.max_nodes",
			Value:       json.RawMessage(`100`),
			Scope:       "global",
			Description: "Maximum nodes allowed per workflow",
			Type:        "number",
			Version:     2,
			UpdatedAt:   time.Now().Add(-14 * 24 * time.Hour),
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": configs,
		"total": len(configs),
	})
}

func (h *ConfigHandler) createConfig(w http.ResponseWriter, r *http.Request) {
	var config Configuration
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config.ID = "cfg-" + time.Now().Format("20060102150405")
	config.Version = 1
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(config)
}

func (h *ConfigHandler) updateConfig(w http.ResponseWriter, r *http.Request) {
	var config Configuration
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config.Version++
	config.UpdatedAt = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

func (h *ConfigHandler) deleteConfig(w http.ResponseWriter, r *http.Request) {
	configID := r.URL.Query().Get("id")
	if configID == "" {
		http.Error(w, "Config ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration deleted successfully",
	})
}

// HandleGlobalConfig handles global configuration
func (h *ConfigHandler) HandleGlobalConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		globalConfig := map[string]interface{}{
			"app": map[string]interface{}{
				"name":        "LinkFlow AI",
				"version":     "1.0.0",
				"environment": "production",
			},
			"limits": map[string]interface{}{
				"maxWorkflows":         100,
				"maxExecutionsPerDay":  10000,
				"maxNodesPerWorkflow":  100,
				"maxCredentials":       50,
				"maxStorageMB":         10000,
				"executionTimeoutSec":  300,
			},
			"features": map[string]interface{}{
				"workflowVersioning": true,
				"multiTenancy":       true,
				"auditLogging":       true,
				"webhooks":           true,
				"schedules":          true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(globalConfig)

	case http.MethodPut:
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Global configuration updated",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleTenantConfig handles tenant-specific configuration
func (h *ConfigHandler) HandleTenantConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenantId")
	if tenantID == "" {
		http.Error(w, "Tenant ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		tenantConfig := map[string]interface{}{
			"tenantId": tenantID,
			"limits": map[string]interface{}{
				"maxWorkflows":        50,
				"maxExecutionsPerDay": 5000,
				"maxNodesPerWorkflow": 50,
			},
			"features": map[string]interface{}{
				"customNodes":    true,
				"apiAccess":      true,
				"prioritySupport": false,
			},
			"branding": map[string]interface{}{
				"logoUrl":      "",
				"primaryColor": "#4A90D9",
				"customDomain": "",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tenantConfig)

	case http.MethodPut:
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Tenant configuration updated",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// FeatureFlag represents a feature flag
type FeatureFlag struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Key         string          `json:"key"`
	Enabled     bool            `json:"enabled"`
	Description string          `json:"description,omitempty"`
	Rollout     int             `json:"rollout"` // percentage 0-100
	Conditions  json.RawMessage `json:"conditions,omitempty"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// HandleFeatureFlags handles feature flag operations
func (h *ConfigHandler) HandleFeatureFlags(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		flags := []FeatureFlag{
			{ID: "ff-001", Name: "Workflow Versioning", Key: "workflow_versioning", Enabled: true, Rollout: 100, UpdatedAt: time.Now()},
			{ID: "ff-002", Name: "AI Suggestions", Key: "ai_suggestions", Enabled: false, Rollout: 0, Description: "AI-powered workflow suggestions", UpdatedAt: time.Now()},
			{ID: "ff-003", Name: "New UI", Key: "new_ui", Enabled: true, Rollout: 50, Description: "New user interface beta", UpdatedAt: time.Now()},
			{ID: "ff-004", Name: "Advanced Analytics", Key: "advanced_analytics", Enabled: true, Rollout: 100, UpdatedAt: time.Now()},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": flags,
			"total": len(flags),
		})

	case http.MethodPut:
		var flag FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&flag); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		flag.UpdatedAt = time.Now()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flag)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ConfigHistory represents a configuration change history entry
type ConfigHistory struct {
	ID        string          `json:"id"`
	ConfigID  string          `json:"configId"`
	Key       string          `json:"key"`
	OldValue  json.RawMessage `json:"oldValue,omitempty"`
	NewValue  json.RawMessage `json:"newValue"`
	ChangedBy string          `json:"changedBy"`
	ChangedAt time.Time       `json:"changedAt"`
	Reason    string          `json:"reason,omitempty"`
}

// GetHistory returns configuration change history
func (h *ConfigHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	history := []ConfigHistory{
		{
			ID:        "hist-001",
			ConfigID:  "cfg-001",
			Key:       "rate_limit.requests_per_minute",
			OldValue:  json.RawMessage(`80`),
			NewValue:  json.RawMessage(`100`),
			ChangedBy: "admin@linkflow.ai",
			ChangedAt: time.Now().Add(-24 * time.Hour),
			Reason:    "Increased rate limit for better performance",
		},
		{
			ID:        "hist-002",
			ConfigID:  "cfg-003",
			Key:       "workflow.max_nodes",
			OldValue:  json.RawMessage(`50`),
			NewValue:  json.RawMessage(`100`),
			ChangedBy: "admin@linkflow.ai",
			ChangedAt: time.Now().Add(-14 * 24 * time.Hour),
			Reason:    "Increased max nodes limit",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": history,
		"total": len(history),
	})
}
