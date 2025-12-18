package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/node/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/node/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/node/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
)

// Server represents the node service server
type Server struct {
	config         *config.Config
	logger         logger.Logger
	telemetry      interface{}
	httpServer     *http.Server
	db             *database.DB
	cache          *cache.RedisCache
	eventPublisher *kafka.EventPublisher
	nodeService    *service.NodeService
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

	// Initialize cache (optional)
	if s.config.Redis.Host != "" {
		cacheConfig := cache.Config{
			Host:      s.config.Redis.Host,
			Port:      s.config.Redis.Port,
			Password:  s.config.Redis.Password,
			DB:        s.config.Redis.DB,
			KeyPrefix: "node",
		}
		redisCache, err := cache.NewRedisCache(cacheConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize Redis cache, continuing without cache", "error", err)
		} else {
			s.cache = redisCache
		}
	}

	// Initialize Kafka publisher (optional)
	if len(s.config.Kafka.Brokers) > 0 {
		kafkaConfig := &kafka.Config{
			Brokers: s.config.Kafka.Brokers,
			Topic:   "node-events",
		}
		publisher, err := kafka.NewEventPublisher(kafkaConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize Kafka publisher, continuing without events", "error", err)
		} else {
			s.eventPublisher = publisher
		}
	}

	// Initialize repository
	nodeRepo := postgres.NewNodeDefinitionRepository(db)

	// Initialize application service
	s.nodeService = service.NewNodeService(
		nodeRepo,
		s.eventPublisher,
		s.cache,
		s.logger,
	)

	// Initialize system nodes
	go func() {
		ctx := context.Background()
		if err := s.nodeService.InitializeSystemNodes(ctx); err != nil {
			s.logger.Error("Failed to initialize system nodes", "error", err)
		}
	}()

	// Setup HTTP server
	s.setupHTTPServer()

	return nil
}

func (s *Server) setupHTTPServer() {
	router := mux.NewRouter()

	// Add middleware
	router.Use(s.loggingMiddleware)
	router.Use(s.recoveryMiddleware)
	
	// Add auth middleware
	authMiddleware := middleware.NewAuthMiddleware([]byte(s.config.Auth.JWTSecret))
	router.Use(authMiddleware.Middleware)

	// Health checks (no auth required)
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()

	// Initialize handlers
	nodeHandler := handlers.NewNodeHandler(s.nodeService, s.logger)

	// Register routes
	nodeHandler.RegisterRoutes(apiRouter)

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

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
	}

	// Close connections
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Database close error", "error", err)
		}
	}

	if s.cache != nil {
		if err := s.cache.Close(); err != nil {
			s.logger.Error("Cache close error", "error", err)
		}
	}

	if s.eventPublisher != nil {
		if err := s.eventPublisher.Close(); err != nil {
			s.logger.Error("Event publisher close error", "error", err)
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
