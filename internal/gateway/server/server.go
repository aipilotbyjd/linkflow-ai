package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/gateway/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/gateway/middleware"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type Server struct {
	config     *config.Config
	logger     logger.Logger
	httpServer *http.Server
	proxies    map[string]*httputil.ReverseProxy
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) {
		s.config = cfg
	}
}

func WithLogger(logger logger.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

func New(opts ...Option) (*Server, error) {
	s := &Server{
		proxies: make(map[string]*httputil.ReverseProxy),
	}

	for _, opt := range opts {
		opt(s)
	}

	if err := s.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	return s, nil
}

func (s *Server) initialize() error {
	// Initialize service proxies - ALL 18 SERVICES
	services := map[string]string{
		"auth":         "http://localhost:8001",
		"user":         "http://localhost:8002",
		"execution":    "http://localhost:8003",
		"workflow":     "http://localhost:8004",
		"node":         "http://localhost:8005",
		"schedule":     "http://localhost:8006",
		"webhook":      "http://localhost:8007",
		"notification": "http://localhost:8008",
		"analytics":    "http://localhost:8009",
		"search":       "http://localhost:8010",
		"storage":      "http://localhost:8011",
		"integration":  "http://localhost:8012",
		"monitoring":   "http://localhost:8013",
		"config":       "http://localhost:8014",
		"migration":    "http://localhost:8015",
		"backup":       "http://localhost:8016",
		"admin":        "http://localhost:8017",
	}

	for name, addr := range services {
		target, err := url.Parse(addr)
		if err != nil {
			return fmt.Errorf("failed to parse %s URL: %w", name, err)
		}
		s.proxies[name] = httputil.NewSingleHostReverseProxy(target)
	}

	s.setupHTTPServer()
	return nil
}

func (s *Server) setupHTTPServer() {
	router := mux.NewRouter()

	// Add middleware
	router.Use(s.loggingMiddleware)
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RateLimitMiddleware())
	
	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// Gateway info
	router.HandleFunc("/gateway/info", handlers.GatewayInfo).Methods("GET")
	
	// Service routes with path-based routing - ALL 18 SERVICES
	router.PathPrefix("/api/v1/auth").Handler(s.proxies["auth"])
	router.PathPrefix("/api/v1/users").Handler(s.proxies["user"])
	router.PathPrefix("/api/v1/workflows").Handler(s.proxies["workflow"])
	router.PathPrefix("/api/v1/executions").Handler(s.proxies["execution"])
	router.PathPrefix("/api/v1/nodes").Handler(s.proxies["node"])
	router.PathPrefix("/api/v1/schedules").Handler(s.proxies["schedule"])
	router.PathPrefix("/api/v1/webhooks").Handler(s.proxies["webhook"])
	router.PathPrefix("/api/v1/notifications").Handler(s.proxies["notification"])
	router.PathPrefix("/api/v1/analytics").Handler(s.proxies["analytics"])
	router.PathPrefix("/api/v1/search").Handler(s.proxies["search"])
	router.PathPrefix("/api/v1/files").Handler(s.proxies["storage"])
	router.PathPrefix("/api/v1/integrations").Handler(s.proxies["integration"])
	router.PathPrefix("/api/v1/metrics").Handler(s.proxies["monitoring"])
	router.PathPrefix("/api/v1/configs").Handler(s.proxies["config"])
	router.PathPrefix("/api/v1/migrations").Handler(s.proxies["migration"])
	router.PathPrefix("/api/v1/backups").Handler(s.proxies["backup"])
	router.PathPrefix("/api/v1/admin").Handler(s.proxies["admin"])

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
	s.logger.Info("Shutting down server")
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

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("Gateway request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
