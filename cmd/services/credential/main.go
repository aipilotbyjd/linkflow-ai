package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/credential/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/credential/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

const (
	serviceName = "credential-service"
	servicePort = 8018
)

type Server struct {
	credentialService *service.CredentialService
	logger            logger.Logger
	httpServer        *http.Server
}

func main() {
	// Initialize logger
	log := logger.New(config.LoggerConfig{
		Level:  "info",
		Format: "json",
	})
	log.Info("Starting Credential Service", "port", servicePort)

	// Load configuration
	cfg, err := config.Load(".")
	if err != nil {
		log.Warn("Failed to load config, using defaults", "error", err)
	}

	// Get encryption key from environment or config
	encryptionKey := os.Getenv("CREDENTIAL_ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "default-32-byte-encryption-key!!" // 32 bytes for AES-256
	}

	// Initialize credential service
	credService := service.NewCredentialService(encryptionKey)

	// Create server
	srv := &Server{
		credentialService: credService,
		logger:            log,
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	srv.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", servicePort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Info("HTTP server starting", "addr", srv.httpServer.Addr)
		if err := srv.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.httpServer.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server stopped")
	_ = cfg
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Health endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/health/live", s.handleHealth)
	mux.HandleFunc("/health/ready", s.handleHealth)

	// Credential endpoints
	mux.HandleFunc("/api/v1/credentials", s.handleCredentials)
	mux.HandleFunc("/api/v1/credentials/", s.handleCredentialByID)

	// Variable endpoints
	mux.HandleFunc("/api/v1/variables", s.handleVariables)
	mux.HandleFunc("/api/v1/variables/", s.handleVariableByID)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": serviceName,
		"time":    time.Now().UTC(),
	})
}

func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listCredentials(w, r)
	case http.MethodPost:
		s.createCredential(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCredentialByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	id := r.URL.Path[len("/api/v1/credentials/"):]
	if id == "" {
		http.Error(w, "Credential ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getCredential(w, r, id)
	case http.MethodPut:
		s.updateCredential(w, r, id)
	case http.MethodDelete:
		s.deleteCredential(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) createCredential(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID         string                 `json:"userId"`
		OrganizationID string                 `json:"organizationId"`
		Name           string                 `json:"name"`
		Type           model.CredentialType   `json:"type"`
		Provider       string                 `json:"provider"`
		Data           map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cred, err := s.credentialService.CreateCredential(
		r.Context(),
		req.UserID,
		req.OrganizationID,
		req.Name,
		req.Type,
		req.Provider,
		req.Data,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cred)
}

func (s *Server) getCredential(w http.ResponseWriter, r *http.Request, id string) {
	cred, err := s.credentialService.GetCredential(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cred)
}

func (s *Server) listCredentials(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId parameter required", http.StatusBadRequest)
		return
	}

	credentials, err := s.credentialService.ListCredentials(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"credentials": credentials,
		"total":       len(credentials),
	})
}

func (s *Server) updateCredential(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.credentialService.UpdateCredential(r.Context(), id, req.Data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCredential(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.credentialService.DeleteCredential(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleVariables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listVariables(w, r)
	case http.MethodPost:
		s.createVariable(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleVariableByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/variables/"):]
	if id == "" {
		http.Error(w, "Variable ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getVariable(w, r, id)
	case http.MethodPut:
		s.updateVariable(w, r, id)
	case http.MethodDelete:
		s.deleteVariable(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) createVariable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string              `json:"userId"`
		Key    string              `json:"key"`
		Value  string              `json:"value"`
		Type   model.VariableType  `json:"type"`
		Scope  model.VariableScope `json:"scope"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	variable, err := s.credentialService.CreateVariable(
		r.Context(),
		req.UserID,
		req.Key,
		req.Value,
		req.Type,
		req.Scope,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(variable)
}

func (s *Server) getVariable(w http.ResponseWriter, r *http.Request, id string) {
	variable, err := s.credentialService.GetVariable(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(variable)
}

func (s *Server) listVariables(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	scope := model.VariableScope(r.URL.Query().Get("scope"))

	if userID == "" {
		http.Error(w, "userId parameter required", http.StatusBadRequest)
		return
	}

	if scope == "" {
		scope = model.VariableScopeGlobal
	}

	variables, err := s.credentialService.ListVariables(r.Context(), userID, scope)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"variables": variables,
		"total":     len(variables),
	})
}

func (s *Server) updateVariable(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.credentialService.UpdateVariable(r.Context(), id, req.Value); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteVariable(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.credentialService.DeleteVariable(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
