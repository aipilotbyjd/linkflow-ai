package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/service/domainservice"
)

// Server represents the workflow service server
type Server struct {
	config           *config.Config
	logger           logger.Logger
	telemetry        interface{}
	httpServer       *http.Server
	db               *database.DB
	workflowService  *service.WorkflowService
}

// Option is a server configuration option
type Option func(*Server)

// WithConfig sets the server config
func WithConfig(cfg *config.Config) Option {
	return func(s *Server) {
		s.config = cfg
	}
}

// WithLogger sets the server logger
func WithLogger(logger logger.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithTelemetry sets the server telemetry
func WithTelemetry(telemetry interface{}) Option {
	return func(s *Server) {
		s.telemetry = telemetry
	}
}

// New creates a new server instance
func New(opts ...Option) (*Server, error) {
	s := &Server{}

	for _, opt := range opts {
		opt(s)
	}

	if err := s.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	return s, nil
}

func (s *Server) initialize() error {
	// Initialize database
	db, err := database.New(s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	s.db = db

	// Initialize repositories
	workflowRepo := postgres.NewWorkflowRepository(db)

	// Initialize domain service
	domainService := domainservice.NewWorkflowDomainService(workflowRepo)

	// Initialize application service
	s.workflowService = service.NewWorkflowService(
		domainService,
		workflowRepo,
		s.logger,
	)

	// Setup HTTP server
	s.setupHTTPServer()

	return nil
}

func (s *Server) setupHTTPServer() {
	router := mux.NewRouter()

	// Add middleware
	router.Use(s.loggingMiddleware)
	router.Use(s.recoveryMiddleware)

	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Initialize handlers
	workflowHandler := handlers.NewWorkflowHandler(s.workflowService, s.logger)

	// Register routes
	workflowHandler.RegisterRoutes(apiRouter)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler:      router,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
		IdleTimeout:  s.config.HTTP.IdleTimeout,
	}
}

// Start starts the server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.config.HTTP.Port)
	return s.httpServer.ListenAndServe()
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	
	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
	}

	// Close database connection
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Database close error", "error", err)
		}
	}

	return nil
}

// Health check handlers
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := s.db.HealthCheck(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not ready","error":"%s"}`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready"}`)
}

// Middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request
		s.logger.Debug("HTTP request", 
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		
		next.ServeHTTP(w, r)
		
		// Log response
		s.logger.Info("HTTP request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Error("panic recovered", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, `{"error":"internal server error"}`)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
