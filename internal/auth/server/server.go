package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/http/handlers"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/adapters/repository/postgres"
	"github.com/linkflow-ai/linkflow-ai/internal/auth/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

// Server represents the auth service server
type Server struct {
	config      *config.Config
	logger      logger.Logger
	telemetry   interface{}
	httpServer  *http.Server
	db          *database.DB
	authService *service.AuthService
	authHandler *handlers.AuthHandler
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
func WithLogger(log logger.Logger) Option {
	return func(s *Server) {
		s.logger = log
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
	
	// Initialize repositories
	tokenRepo := postgres.NewTokenRepository(db.DB)
	apiKeyRepo := postgres.NewAPIKeyRepository(db.DB)
	oauthRepo := postgres.NewOAuthRepository(db.DB)
	pgUserRepo := postgres.NewUserRepository(db.DB)
	
	// Wrap user repository to adapt types
	userRepo := &userRepoAdapter{repo: pgUserRepo}
	
	// Initialize auth service with configuration
	authConfig := service.Config{
		JWTSecret:             s.config.Auth.JWTSecret,
		JWTIssuer:             "linkflow-ai",
		AccessTokenExpiry:     s.config.Auth.JWTExpiry,
		RefreshTokenExpiry:    7 * 24 * time.Hour,
		PasswordResetExpiry:   1 * time.Hour,
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
	}
	
	s.authService = service.NewAuthService(
		authConfig,
		userRepo,
		tokenRepo,
		apiKeyRepo,
		oauthRepo,
		nil, // Email service - can be injected later
		nil, // Audit logger - can be injected later
	)
	
	// Initialize handler
	s.authHandler = handlers.NewAuthHandler(s.authService)
	
	// Setup HTTP server
	s.setupHTTPServer()
	
	return nil
}

func (s *Server) setupHTTPServer() {
	router := mux.NewRouter()
	
	// Add middleware
	router.Use(s.loggingMiddleware)
	router.Use(s.recoveryMiddleware)
	router.Use(s.securityHeadersMiddleware)
	
	// Health checks
	router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")
	
	// Register auth handler routes (uses actual service implementation)
	s.authHandler.RegisterRoutes(&routerAdapter{router: router})
	
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
		Handler:      router,
		ReadTimeout:  s.config.HTTP.ReadTimeout,
		WriteTimeout: s.config.HTTP.WriteTimeout,
		IdleTimeout:  s.config.HTTP.IdleTimeout,
	}
}

// securityHeadersMiddleware adds security headers to responses
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

// Auth route handlers (placeholder implementations)
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement full login with service
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"accessToken":  "placeholder-token",
		"refreshToken": "placeholder-refresh",
		"expiresAt":    time.Now().Add(s.config.Auth.JWTExpiry),
		"tokenType":    "Bearer",
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "Registration endpoint - implement with user service",
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"accessToken":  "new-placeholder-token",
		"refreshToken": "new-placeholder-refresh",
		"expiresAt":    time.Now().Add(s.config.Auth.JWTExpiry),
		"tokenType":    "Bearer",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "If an account exists with this email, you will receive a password reset link",
	})
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Password has been reset successfully",
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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
	
	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("Database close error", "error", err)
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
		
		s.logger.Debug("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		
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

// userRepoAdapter adapts postgres.UserRepository to service.UserRepository
type userRepoAdapter struct {
	repo *postgres.UserRepository
}

func (a *userRepoAdapter) FindByID(ctx context.Context, id string) (*service.User, error) {
	u, err := a.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return a.toServiceUser(u), nil
}

func (a *userRepoAdapter) FindByEmail(ctx context.Context, email string) (*service.User, error) {
	u, err := a.repo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return a.toServiceUser(u), nil
}

func (a *userRepoAdapter) Create(ctx context.Context, user *service.User) error {
	return a.repo.Create(ctx, a.toPostgresUser(user))
}

func (a *userRepoAdapter) Update(ctx context.Context, user *service.User) error {
	return a.repo.Update(ctx, a.toPostgresUser(user))
}

func (a *userRepoAdapter) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return a.repo.UpdatePassword(ctx, userID, passwordHash)
}

func (a *userRepoAdapter) VerifyEmail(ctx context.Context, userID string) error {
	return a.repo.VerifyEmail(ctx, userID)
}

func (a *userRepoAdapter) toServiceUser(u *postgres.AuthUser) *service.User {
	return &service.User{
		ID:             u.ID,
		Email:          u.Email,
		PasswordHash:   u.PasswordHash,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		EmailVerified:  u.EmailVerified,
		Status:         u.Status,
		FailedAttempts: u.FailedAttempts,
		LockedUntil:    u.LockedUntil,
		MFAEnabled:     u.MFAEnabled,
		MFASecret:      u.MFASecret,
		Roles:          u.Roles,
		CreatedAt:      u.CreatedAt,
	}
}

func (a *userRepoAdapter) toPostgresUser(u *service.User) *postgres.AuthUser {
	return &postgres.AuthUser{
		ID:             u.ID,
		Email:          u.Email,
		PasswordHash:   u.PasswordHash,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		EmailVerified:  u.EmailVerified,
		Status:         u.Status,
		FailedAttempts: u.FailedAttempts,
		LockedUntil:    u.LockedUntil,
		MFAEnabled:     u.MFAEnabled,
		MFASecret:      u.MFASecret,
		Roles:          u.Roles,
		CreatedAt:      u.CreatedAt,
	}
}

// routerAdapter adapts mux.Router to handlers.Router interface
type routerAdapter struct {
	router *mux.Router
}

func (a *routerAdapter) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	a.router.HandleFunc(pattern, handler)
}
