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
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	// Import node implementations to register them
	_ "github.com/linkflow-ai/linkflow-ai/internal/node/runtime/nodes"

	"github.com/linkflow-ai/linkflow-ai/internal/engine"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/di"
	"github.com/linkflow-ai/linkflow-ai/pkg/middleware"
)

// Config holds server configuration
type Config struct {
	Port            string
	DatabaseURL     string
	JWTSecret       string
	StripeKey       string
	WebhookSecret   string
	Environment     string
	RateLimitPerMin int
	AllowedOrigins  []string
}

func main() {
	// Load configuration
	cfg := loadConfig()

	// Initialize DI container
	container := initContainer(cfg)
	defer container.Close(context.Background())

	// Initialize workflow engine
	workflowEngine := engine.NewEngine()
	container.Register(di.ServiceEngine, workflowEngine)

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

	// Build middleware chain
	handler := buildMiddlewareChain(router, cfg)

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
		log.Printf("Starting API server on port %s (env: %s)", cfg.Port, cfg.Environment)
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

// initContainer initializes the DI container with all services
func initContainer(cfg Config) *di.Container {
	container := di.New()

	// Register config
	container.Register(di.ServiceConfig, cfg)

	// Register factories for lazy initialization

	// Database (would connect when first accessed)
	container.RegisterFactory(di.ServiceDB, func(c *di.Container) (interface{}, error) {
		// In production, return actual DB connection
		// db, err := database.New(cfg.DatabaseURL)
		log.Println("Database service initialized")
		return nil, nil
	})

	// Cache (Redis or in-memory)
	container.RegisterFactory(di.ServiceCache, func(c *di.Container) (interface{}, error) {
		log.Println("Cache service initialized")
		return nil, nil
	})

	// Event bus
	container.RegisterFactory(di.ServiceEventBus, func(c *di.Container) (interface{}, error) {
		log.Println("Event bus initialized")
		return nil, nil
	})

	return container
}

// buildMiddlewareChain builds the middleware stack
func buildMiddlewareChain(router *mux.Router, cfg Config) http.Handler {
	var handler http.Handler = router

	// CORS middleware (should be outermost for preflight)
	corsConfig := &middleware.CORSConfig{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-API-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
	handler = middleware.CORS(corsConfig)(handler)

	// Rate limiting middleware
	rateLimitConfig := &middleware.RateLimitConfig{
		RequestsPerMinute: cfg.RateLimitPerMin,
		BurstSize:         cfg.RateLimitPerMin * 2,
		SkipPaths:         []string{"/health", "/api/v1/health"},
	}
	handler = middleware.RateLimit(rateLimitConfig)(handler)

	// Request ID middleware
	handler = middleware.RequestID(handler)

	// Recovery middleware (must be innermost to catch panics)
	handler = middleware.SimpleRecovery(handler)

	return handler
}

func loadConfig() Config {
	allowedOrigins := strings.Split(getEnv("ALLOWED_ORIGINS", "*"), ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}

	rateLimitPerMin := 100
	if val := os.Getenv("RATE_LIMIT_PER_MIN"); val != "" {
		fmt.Sscanf(val, "%d", &rateLimitPerMin)
	}

	return Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://localhost:5432/linkflow?sslmode=disable"),
		JWTSecret:       getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		StripeKey:       getEnv("STRIPE_SECRET_KEY", ""),
		WebhookSecret:   getEnv("STRIPE_WEBHOOK_SECRET", ""),
		Environment:     getEnv("ENVIRONMENT", "development"),
		RateLimitPerMin: rateLimitPerMin,
		AllowedOrigins:  allowedOrigins,
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
