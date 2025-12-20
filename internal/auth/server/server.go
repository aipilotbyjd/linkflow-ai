package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/http/middleware"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type Server struct {
	config     *config.Config
	logger     logger.Logger
	telemetry  interface{}
	engine     *gin.Engine
	httpServer *http.Server
	db         *database.GormDB
}

type Option func(*Server)

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) { s.config = cfg }
}

func WithLogger(log logger.Logger) Option {
	return func(s *Server) { s.logger = log }
}

func WithTelemetry(tel interface{}) Option {
	return func(s *Server) { s.telemetry = tel }
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
	if s.config.Service.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewGorm(s.config.Database)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	s.db = db

	authService := service.NewAuthService(
		service.Config{
			JWTSecret:             s.config.Auth.JWTSecret,
			JWTIssuer:             "linkflow-ai",
			AccessTokenExpiry:     s.config.Auth.JWTExpiry,
			RefreshTokenExpiry:    7 * 24 * time.Hour,
			PasswordResetExpiry:   time.Hour,
			MaxLoginAttempts:      5,
			LockoutDuration:       15 * time.Minute,
			BcryptCost:            12,
			RequireEmailVerify:    false,
			AllowSignup:           true,
			PasswordMinLength:     8,
			PasswordRequireUpper:  true,
			PasswordRequireLower:  true,
			PasswordRequireNumber: true,
			PasswordRequireSymbol: false,
		},
		postgres.NewUserRepository(db.DB),
		postgres.NewTokenRepository(db.DB),
		postgres.NewAPIKeyRepository(db.DB),
		postgres.NewOAuthRepository(db.DB),
		nil,
		nil,
	)

	s.setupRouter(authService)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler:      s.engine,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
		IdleTimeout:  s.config.HTTP.IdleTimeout,
	}

	return nil
}

func (s *Server) setupRouter(authService *service.AuthService) {
	s.engine = gin.New()

	s.engine.Use(
		gin.Recovery(),
		middleware.Logger(s.logger),
		middleware.CORS(),
		middleware.SecurityHeaders(),
	)

	s.engine.GET("/health/live", s.healthLive)
	s.engine.GET("/health/ready", s.healthReady)

	authHandler := handlers.NewAuthHandler(authService)
	authHandler.RegisterRoutes(s.engine.Group("/api/v1"))
}

func (s *Server) healthLive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

func (s *Server) healthReady(c *gin.Context) {
	if err := s.db.HealthCheck(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
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
		if err := s.db.Close(); err != nil {
			s.logger.Error("Database close error", "error", err)
		}
	}

	return nil
}
