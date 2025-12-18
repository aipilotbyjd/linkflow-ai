package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
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

	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// Monitoring endpoints
	router.HandleFunc("/api/v1/metrics", s.handleMetrics).Methods("GET")
	router.HandleFunc("/api/v1/metrics/collect", s.handleCollect).Methods("POST")
	router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")
	router.HandleFunc("/api/v1/alerts", s.handleAlerts).Methods("GET")
	router.HandleFunc("/api/v1/alerts", s.handleCreateAlert).Methods("POST")

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

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"metrics":{"cpu":45.2,"memory":67.8,"disk":23.4}}`)
}

func (s *Server) handleCollect(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"Metrics collected"}`)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"services":{"all":"healthy"}}`)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"alerts":[]}`)
}

func (s *Server) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"message":"Alert created"}`)
}
