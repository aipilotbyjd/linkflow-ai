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

	// File operations
	router.HandleFunc("/api/v1/files/upload", s.handleUpload).Methods("POST")
	router.HandleFunc("/api/v1/files/{id}/download", s.handleDownload).Methods("GET")
	router.HandleFunc("/api/v1/files/{id}", s.handleDelete).Methods("DELETE")
	router.HandleFunc("/api/v1/files", s.handleList).Methods("GET")

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

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"File upload endpoint"}`)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"File download endpoint"}`)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"message":"File delete endpoint"}`)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"files":[]}`)
}
