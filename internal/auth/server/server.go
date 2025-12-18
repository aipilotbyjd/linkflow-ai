package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"
)

// Server represents the auth service server
type Server struct {
	config     interface{}
	logger     interface{}
	telemetry  interface{}
	httpServer *http.Server
}

// Option is a server configuration option
type Option func(*Server)

// WithConfig sets the server config
func WithConfig(cfg interface{}) Option {
	return func(s *Server) {
		s.config = cfg
	}
}

// WithLogger sets the server logger
func WithLogger(logger interface{}) Option {
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
	
	// Setup HTTP server
	router := mux.NewRouter()
	
	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")
	
	// Auth routes
	router.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
	router.HandleFunc("/auth/register", s.handleRegister).Methods("POST")
	router.HandleFunc("/auth/refresh", s.handleRefresh).Methods("POST")
	router.HandleFunc("/auth/logout", s.handleLogout).Methods("POST")
	
	s.httpServer = &http.Server{
		Addr:         ":8001",
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	return s, nil
}

// Start starts the server
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Health check handlers
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	// Check dependencies
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready"}`)
}

// Auth handlers
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"token":"jwt-token-here","refresh_token":"refresh-token-here"}`)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"user_id":"123","email":"user@example.com"}`)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"token":"new-jwt-token","refresh_token":"new-refresh-token"}`)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
