package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/middleware"
)

type Server struct {
	config               *config.Config
	logger               logger.Logger
	telemetry            interface{}
	httpServer           *http.Server
	db                   *database.DB
	cache                *cache.RedisCache
	eventPublisher       *kafka.EventPublisher
	notificationService  *service.NotificationService
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

func WithTelemetry(telemetry interface{}) Option {
	return func(s *Server) {
		s.telemetry = telemetry
	}
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
			KeyPrefix: "notification",
		}
		redisCache, err := cache.NewRedisCache(cacheConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize Redis cache", "error", err)
		} else {
			s.cache = redisCache
		}
	}

	// Initialize Kafka (optional)
	if len(s.config.Kafka.Brokers) > 0 {
		kafkaConfig := &kafka.Config{
			Brokers: s.config.Kafka.Brokers,
			Topic:   "notification-events",
		}
		publisher, err := kafka.NewEventPublisher(kafkaConfig)
		if err != nil {
			s.logger.Warn("Failed to initialize Kafka", "error", err)
		} else {
			s.eventPublisher = publisher
		}
	}

	// Initialize repository
	notificationRepo := postgres.NewNotificationRepository(db)

	// Initialize service
	s.notificationService = service.NewNotificationService(
		notificationRepo,
		s.eventPublisher,
		s.cache,
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
	
	// Add auth middleware
	authMiddleware := middleware.NewAuthMiddleware([]byte(s.config.Auth.JWTSecret))
	router.Use(authMiddleware.Middleware)

	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")

	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	
	// Initialize handlers
	notificationHandler := handlers.NewNotificationHandler(s.notificationService, s.logger)
	notificationHandler.RegisterRoutes(apiRouter)

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

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", "error", err)
	}

	if s.db != nil {
		_ = s.db.Close()
	}
	if s.cache != nil {
		_ = s.cache.Close()
	}
	if s.eventPublisher != nil {
		_ = s.eventPublisher.Close()
	}

	return nil
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
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
