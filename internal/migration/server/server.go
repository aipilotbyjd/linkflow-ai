package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/migration/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/migration/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type Server struct {
	config           *config.Config
	logger           logger.Logger
	httpServer       *http.Server
	migrationService *service.MigrationService
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) { s.config = cfg }
}

func WithLogger(logger logger.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

func WithMigrationService(svc *service.MigrationService) Option {
	return func(s *Server) { s.migrationService = svc }
}

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
	s.setupHTTPServer()
	return nil
}

func (s *Server) setupHTTPServer() {
	router := mux.NewRouter()

	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// Migration endpoints
	if s.migrationService != nil {
		h := handlers.NewMigrationHandler(s.migrationService)
		router.HandleFunc("/api/v1/migrations", h.HandleList).Methods("GET")
		router.HandleFunc("/api/v1/migrations/status", h.HandleStatus).Methods("GET")
		router.HandleFunc("/api/v1/migrations/up", h.HandleMigrateUp).Methods("POST")
		router.HandleFunc("/api/v1/migrations/down", h.HandleMigrateDown).Methods("POST")
		router.HandleFunc("/api/v1/migrations/reset", h.HandleReset).Methods("POST")
		router.HandleFunc("/api/v1/migrations/seed", h.HandleSeed).Methods("POST")
		router.HandleFunc("/api/v1/migrations/version", h.HandleVersion).Methods("GET")
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler:      router,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
		IdleTimeout:  s.config.HTTP.IdleTimeout,
	}
}

func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.config.HTTP.Port)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready"}`)
}
