package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
)

type Server struct {
	config     *config.Config
	logger     logger.Logger
	httpServer *http.Server
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) { s.config = cfg }
}

func WithLogger(logger logger.Logger) Option {
	return func(s *Server) { s.logger = logger }
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

	// Add auth middleware
	authMiddleware := middleware.NewAuthMiddleware([]byte(s.config.Auth.JWTSecret))
	router.Use(authMiddleware.Middleware)

	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// Admin endpoints
	router.HandleFunc("/api/v1/admin/dashboard", s.handleDashboard).Methods("GET")
	router.HandleFunc("/api/v1/admin/users", s.handleUsers).Methods("GET")
	router.HandleFunc("/api/v1/admin/system", s.handleSystem).Methods("GET")
	router.HandleFunc("/api/v1/admin/services", s.handleServices).Methods("GET")
	router.HandleFunc("/api/v1/admin/logs", s.handleLogs).Methods("GET")
	router.HandleFunc("/api/v1/admin/settings", s.handleSettings).Methods("GET")
	router.HandleFunc("/api/v1/admin/settings", s.handleUpdateSettings).Methods("PUT")

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
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready"}`)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"users":100,"workflows":250,"executions":1500,"services_healthy":18}`)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"users":[],"total":100}`)
}

func (s *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"cpu":45.2,"memory":67.8,"disk":23.4,"uptime":"72h"}`)
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"services":[{"name":"auth","status":"healthy"},{"name":"user","status":"healthy"}]}`)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"logs":[]}`)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"settings":{"maintenance_mode":false,"max_users":1000}}`)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"Settings updated"}`)
}
