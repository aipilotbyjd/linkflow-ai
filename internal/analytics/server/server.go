package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/linkflow-ai/linkflow-ai/internal/analytics/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/analytics/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/analytics/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type Server struct {
	config           *config.Config
	logger           logger.Logger
	telemetry        interface{}
	httpServer       *http.Server
	db               *database.DB
	cache            *cache.RedisCache
	analyticsService *service.AnalyticsService
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) { s.config = cfg }
}

func WithLogger(logger logger.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

func WithTelemetry(telemetry interface{}) Option {
	return func(s *Server) { s.telemetry = telemetry }
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
			KeyPrefix: "analytics",
		}
		redisCache, err := cache.NewRedisCache(cacheConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize Redis cache", "error", err)
		} else {
			s.cache = redisCache
		}
	}

	// Initialize repository
	eventRepo := postgres.NewEventRepository(db)

	// Initialize service
	s.analyticsService = service.NewAnalyticsService(eventRepo, s.cache, s.logger)

	// Setup HTTP server
	s.setupHTTPServer()

	return nil
}

func (s *Server) setupHTTPServer() {
	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"alive"}`)
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ready"}`)
	})

	// Register handlers
	handler := handlers.NewAnalyticsHandler(s.analyticsService, s.logger)
	handler.RegisterRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler:      mux,
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

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
	}

	if s.db != nil {
		_ = s.db.Close()
	}
	if s.cache != nil {
		_ = s.cache.Close()
	}

	return nil
}
