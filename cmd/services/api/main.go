// Package main provides the main API server entry point
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	// Import node implementations to register them
	_ "github.com/linkflow-ai/linkflow-ai/internal/node/runtime/nodes"

	"github.com/linkflow-ai/linkflow-ai/internal/engine"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/middleware"
)

// Config holds server configuration
type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	StripeKey      string
	WebhookSecret  string
	Environment    string
}

func main() {
	// Load configuration
	cfg := loadConfig()

	// Initialize workflow engine
	workflowEngine := engine.NewEngine()

	// Create router
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Node registry endpoint
	router.HandleFunc("/api/v1/nodes", nodeListHandler).Methods("GET")
	router.HandleFunc("/api/v1/nodes/{type}", nodeInfoHandler).Methods("GET")

	// Workflow execution endpoint (simplified for demo)
	router.HandleFunc("/api/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		executeHandler(w, r, workflowEngine)
	}).Methods("POST")

	// Executions list
	router.HandleFunc("/api/v1/executions", func(w http.ResponseWriter, r *http.Request) {
		listExecutionsHandler(w, r, workflowEngine)
	}).Methods("GET")

	// Apply CORS middleware
	corsConfig := &middleware.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{"*"},
	}
	handler := middleware.CORS(corsConfig)(router)

	_ = cfg // Use config later for full wiring

	// Create server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func loadConfig() Config {
	return Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://localhost:5432/linkflow?sslmode=disable"),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		StripeKey:     getEnv("STRIPE_SECRET_KEY", ""),
		WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
}

func nodeListHandler(w http.ResponseWriter, r *http.Request) {
	nodes := runtime.List()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"total": len(nodes),
	})
}

func nodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeType := vars["type"]
	
	executor, err := runtime.Get(nodeType)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Node type not found"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(executor.GetMetadata())
}

func executeHandler(w http.ResponseWriter, r *http.Request, eng *engine.Engine) {
	var req struct {
		Workflow engine.WorkflowDefinition `json:"workflow"`
		Options  engine.ExecutionOptions   `json:"options"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	
	// Execute workflow
	state, err := eng.Execute(r.Context(), &req.Workflow, &req.Options)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":       err.Error(),
			"executionId": state.ID,
			"status":      state.Status,
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executionId": state.ID,
		"status":      state.Status,
		"outputs":     state.NodeOutputs,
		"logs":        state.Logs,
	})
}

func listExecutionsHandler(w http.ResponseWriter, r *http.Request, eng *engine.Engine) {
	workflowID := r.URL.Query().Get("workflowId")
	executions := eng.ListExecutions(workflowID, 50)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
		"total":      len(executions),
	})
}
