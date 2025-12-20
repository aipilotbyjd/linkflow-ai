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

	"github.com/linkflow-ai/linkflow-ai/internal/executor/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

const (
	serviceName = "executor-service"
	servicePort = 8020
)

type Server struct {
	workerPool *model.WorkerPool
	logger     logger.Logger
}

func main() {
	log := logger.New(config.LoggerConfig{Level: "info", Format: "json", OutputPath: "stdout"})
	log.Info("Starting Executor Service", "port", servicePort)

	// Create sandbox pool
	sandboxPool := model.NewSandboxPool(10, func() (model.Sandbox, error) {
		return model.NewNativeSandbox(model.DefaultConstraints()), nil
	})

	// Create worker pool
	workerPool := model.NewWorkerPool(5, 100, sandboxPool)
	workerPool.Start()

	srv := &Server{
		workerPool: workerPool,
		logger:     log,
	}

	// Process results
	go srv.processResults()

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", servicePort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	workerPool.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	httpServer.Shutdown(ctx)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/execute", s.handleExecute)
	mux.HandleFunc("/api/v1/execute/async", s.handleExecuteAsync)
	mux.HandleFunc("/api/v1/environments", s.handleEnvironments)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": serviceName,
		"workers": 5,
	})
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.NodeExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Synchronous execution
	sandbox := model.NewNativeSandbox(req.Config.Constraints)
	result, err := sandbox.Execute(r.Context(), &req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleExecuteAsync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.NodeExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.workerPool.Submit(&req); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requestId": req.ID,
		"status":    "queued",
	})
}

func (s *Server) handleEnvironments(w http.ResponseWriter, r *http.Request) {
	envs := []map[string]interface{}{
		{"id": "native", "name": "Native Go", "description": "Fast native execution"},
		{"id": "v8_isolate", "name": "V8 Isolate", "description": "JavaScript sandbox"},
		{"id": "wasm", "name": "WebAssembly", "description": "WASM sandbox"},
		{"id": "container", "name": "Container", "description": "Docker container isolation"},
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"environments": envs})
}

func (s *Server) processResults() {
	for result := range s.workerPool.Results() {
		s.logger.Info("Execution completed",
			"requestId", result.RequestID,
			"status", result.Status,
			"duration", result.Metrics.DurationMS,
		)
	}
}
