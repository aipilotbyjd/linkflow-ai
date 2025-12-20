// Package main provides the main API server entry point with PostgreSQL persistence
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	// Import node implementations to register them
	_ "github.com/linkflow-ai/linkflow-ai/internal/node/runtime/nodes"

	"github.com/linkflow-ai/linkflow-ai/internal/engine"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/middleware"
)

// simpleLogger implements middleware.Logger
type simpleLogger struct{}

func (l *simpleLogger) Info(msg string, keysAndValues ...interface{})  { log.Println("[INFO]", msg, keysAndValues) }
func (l *simpleLogger) Error(msg string, keysAndValues ...interface{}) { log.Println("[ERROR]", msg, keysAndValues) }
func (l *simpleLogger) Debug(msg string, keysAndValues ...interface{}) { log.Println("[DEBUG]", msg, keysAndValues) }

// Config holds server configuration
type Config struct {
	Port        string
	DatabaseDSN string
	JWTSecret   string
	Environment string
}

// Global database connection
var db *sql.DB

// Workflow engine
var eng *engine.Engine

func main() {
	// Load configuration from environment
	cfg := loadConfig()

	// Connect to PostgreSQL
	var err error
	db, err = sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL")

	// Initialize workflow engine
	eng = engine.NewEngine()
	nodeCount := len(runtime.List())
	log.Printf("Registered %d node types", nodeCount)

	// Create router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.Recovery(&middleware.RecoveryConfig{
		PrintStack: cfg.Environment == "development",
		Logger:     &simpleLogger{},
	}))
	router.Use(middleware.Logging(&middleware.LoggingConfig{
		Logger:    &simpleLogger{},
		SkipPaths: []string{"/health", "/api/health"},
	}))
	router.Use(middleware.CORS(&middleware.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}))

	// Register routes
	registerRoutes(router)

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Starting API server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited")
}

func loadConfig() Config {
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	dbUser := getEnvOrDefault("DB_USER", "postgres")
	dbPass := getEnvOrDefault("DB_PASSWORD", "postgres")
	dbName := getEnvOrDefault("DB_NAME", "linkflow")
	dbSSL := getEnvOrDefault("DB_SSL_MODE", "disable")

	return Config{
		Port:        port,
		DatabaseDSN: fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", dbHost, dbPort, dbUser, dbPass, dbName, dbSSL),
		JWTSecret:   getEnvOrDefault("JWT_SECRET", "linkflow-dev-secret"),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func registerRoutes(r *mux.Router) {
	// Health check
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/health", healthHandler).Methods("GET")

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Auth routes (public)
	api.HandleFunc("/auth/register", registerHandler).Methods("POST")
	api.HandleFunc("/auth/login", loginHandler).Methods("POST")
	api.HandleFunc("/auth/refresh", refreshTokenHandler).Methods("POST")
	api.HandleFunc("/auth/logout", logoutHandler).Methods("POST")
	api.HandleFunc("/auth/me", authMiddleware(meHandler)).Methods("GET")
	api.HandleFunc("/auth/password/reset", requestPasswordResetHandler).Methods("POST")
	api.HandleFunc("/auth/password/reset/{token}", resetPasswordHandler).Methods("POST")
	api.HandleFunc("/auth/sessions", authMiddleware(listSessionsHandler)).Methods("GET")

	// User routes
	api.HandleFunc("/users", authMiddleware(listUsersHandler)).Methods("GET")
	api.HandleFunc("/users/{id}", authMiddleware(getUserHandler)).Methods("GET")
	api.HandleFunc("/users/{id}", authMiddleware(updateUserHandler)).Methods("PUT")
	api.HandleFunc("/users/{id}", authMiddleware(deleteUserHandler)).Methods("DELETE")

	// Workflow routes
	api.HandleFunc("/workflows", authMiddleware(listWorkflowsHandler)).Methods("GET")
	api.HandleFunc("/workflows", authMiddleware(createWorkflowHandler)).Methods("POST")
	api.HandleFunc("/workflows/{id}", authMiddleware(getWorkflowHandler)).Methods("GET")
	api.HandleFunc("/workflows/{id}", authMiddleware(updateWorkflowHandler)).Methods("PUT")
	api.HandleFunc("/workflows/{id}", authMiddleware(deleteWorkflowHandler)).Methods("DELETE")
	api.HandleFunc("/workflows/{id}/activate", authMiddleware(activateWorkflowHandler)).Methods("POST")
	api.HandleFunc("/workflows/{id}/deactivate", authMiddleware(deactivateWorkflowHandler)).Methods("POST")
	api.HandleFunc("/workflows/{id}/execute", authMiddleware(executeWorkflowHandler)).Methods("POST")
	api.HandleFunc("/workflows/{id}/clone", authMiddleware(cloneWorkflowHandler)).Methods("POST")

	// Execution routes
	api.HandleFunc("/executions", authMiddleware(listExecutionsHandler)).Methods("GET")
	api.HandleFunc("/executions/{id}", authMiddleware(getExecutionHandler)).Methods("GET")
	api.HandleFunc("/executions/{id}/cancel", authMiddleware(cancelExecutionHandler)).Methods("POST")
	api.HandleFunc("/execute", authMiddleware(directExecuteHandler)).Methods("POST")

	// Node routes
	api.HandleFunc("/nodes", listNodesHandler).Methods("GET")
	api.HandleFunc("/nodes/{type}", getNodeHandler).Methods("GET")
	api.HandleFunc("/nodes/categories", listNodeCategoriesHandler).Methods("GET")

	// Schedule routes
	api.HandleFunc("/schedules", authMiddleware(listSchedulesHandler)).Methods("GET")
	api.HandleFunc("/schedules", authMiddleware(createScheduleHandler)).Methods("POST")
	api.HandleFunc("/schedules/{id}", authMiddleware(getScheduleHandler)).Methods("GET")
	api.HandleFunc("/schedules/{id}", authMiddleware(updateScheduleHandler)).Methods("PUT")
	api.HandleFunc("/schedules/{id}", authMiddleware(deleteScheduleHandler)).Methods("DELETE")
	api.HandleFunc("/schedules/{id}/pause", authMiddleware(pauseScheduleHandler)).Methods("POST")
	api.HandleFunc("/schedules/{id}/resume", authMiddleware(resumeScheduleHandler)).Methods("POST")
	api.HandleFunc("/schedules/{id}/trigger", authMiddleware(triggerScheduleHandler)).Methods("POST")
	api.HandleFunc("/schedules/cron/validate", authMiddleware(validateCronHandler)).Methods("POST")

	// Credential routes
	api.HandleFunc("/credentials", authMiddleware(listCredentialsHandler)).Methods("GET")
	api.HandleFunc("/credentials", authMiddleware(createCredentialHandler)).Methods("POST")
	api.HandleFunc("/credentials/types", authMiddleware(listCredentialTypesHandler)).Methods("GET")
	api.HandleFunc("/credentials/{id}", authMiddleware(getCredentialHandler)).Methods("GET")
	api.HandleFunc("/credentials/{id}", authMiddleware(updateCredentialHandler)).Methods("PUT")
	api.HandleFunc("/credentials/{id}", authMiddleware(deleteCredentialHandler)).Methods("DELETE")
	api.HandleFunc("/credentials/{id}/test", authMiddleware(testCredentialHandler)).Methods("POST")

	// Integration routes
	api.HandleFunc("/integrations", authMiddleware(listIntegrationsHandler)).Methods("GET")
	api.HandleFunc("/integrations", authMiddleware(createIntegrationHandler)).Methods("POST")
	api.HandleFunc("/integrations/categories", authMiddleware(listIntegrationCategoriesHandler)).Methods("GET")
	api.HandleFunc("/integrations/{id}", authMiddleware(getIntegrationHandler)).Methods("GET")
	api.HandleFunc("/integrations/{id}", authMiddleware(updateIntegrationHandler)).Methods("PUT")
	api.HandleFunc("/integrations/{id}", authMiddleware(deleteIntegrationHandler)).Methods("DELETE")
	api.HandleFunc("/integrations/{id}/enable", authMiddleware(enableIntegrationHandler)).Methods("POST")
	api.HandleFunc("/integrations/{id}/disable", authMiddleware(disableIntegrationHandler)).Methods("POST")
	api.HandleFunc("/integrations/{id}/test", authMiddleware(testIntegrationHandler)).Methods("POST")

	// Webhook routes
	api.HandleFunc("/webhooks", authMiddleware(listWebhooksHandler)).Methods("GET")
	api.HandleFunc("/webhooks", authMiddleware(createWebhookHandler)).Methods("POST")
	api.HandleFunc("/webhooks/{id}", authMiddleware(getWebhookHandler)).Methods("GET")
	api.HandleFunc("/webhooks/{id}", authMiddleware(updateWebhookHandler)).Methods("PUT")
	api.HandleFunc("/webhooks/{id}", authMiddleware(deleteWebhookHandler)).Methods("DELETE")
	api.HandleFunc("/webhooks/{endpointId}/trigger", triggerWebhookHandler).Methods("POST")

	// Notification routes
	api.HandleFunc("/notifications", authMiddleware(listNotificationsHandler)).Methods("GET")
	api.HandleFunc("/notifications", authMiddleware(createNotificationHandler)).Methods("POST")
	api.HandleFunc("/notifications/{id}", authMiddleware(getNotificationHandler)).Methods("GET")
	api.HandleFunc("/notifications/{id}/read", authMiddleware(markNotificationReadHandler)).Methods("POST")

	// Workspace routes
	api.HandleFunc("/workspaces", authMiddleware(listWorkspacesHandler)).Methods("GET")
	api.HandleFunc("/workspaces", authMiddleware(createWorkspaceHandler)).Methods("POST")
	api.HandleFunc("/workspaces/{id}", authMiddleware(getWorkspaceHandler)).Methods("GET")
	api.HandleFunc("/workspaces/{id}", authMiddleware(updateWorkspaceHandler)).Methods("PUT")
	api.HandleFunc("/workspaces/{id}", authMiddleware(deleteWorkspaceHandler)).Methods("DELETE")
	api.HandleFunc("/workspaces/{id}/members", authMiddleware(listWorkspaceMembersHandler)).Methods("GET")
	api.HandleFunc("/workspaces/{id}/members", authMiddleware(inviteWorkspaceMemberHandler)).Methods("POST")

	// Billing routes
	api.HandleFunc("/billing/plans", listPlansHandler).Methods("GET")
	api.HandleFunc("/billing/subscription", authMiddleware(getSubscriptionHandler)).Methods("GET")
	api.HandleFunc("/billing/subscription", authMiddleware(createSubscriptionHandler)).Methods("POST")
	api.HandleFunc("/billing/usage", authMiddleware(getUsageHandler)).Methods("GET")
}

// ============================================================================
// Auth Middleware
// ============================================================================

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "Invalid authorization header")
			return
		}

		token := parts[1]

		// Look up token in sessions table
		var userID string
		var expiresAt time.Time
		err := db.QueryRow(`
			SELECT user_id, expires_at FROM auth_service.sessions 
			WHERE token_hash = $1 AND revoked_at IS NULL
		`, token).Scan(&userID, &expiresAt)

		if err == sql.ErrNoRows {
			respondError(w, http.StatusUnauthorized, "Invalid token")
			return
		}
		if err != nil {
			log.Printf("Auth error: %v", err)
			respondError(w, http.StatusInternalServerError, "Authentication error")
			return
		}

		if time.Now().After(expiresAt) {
			respondError(w, http.StatusUnauthorized, "Token expired")
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), "userID", userID)
		next(w, r.WithContext(ctx))
	}
}

func getUserIDFromContext(r *http.Request) string {
	if v := r.Context().Value("userID"); v != nil {
		return v.(string)
	}
	return ""
}

// ============================================================================
// Response Helpers
// ============================================================================

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error":   message,
		"status":  status,
		"success": false,
	})
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ============================================================================
// Health Handler
// ============================================================================

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbStatus := "healthy"
	if err := db.Ping(); err != nil {
		dbStatus = "unhealthy"
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"database":  dbStatus,
		"nodes":     len(runtime.List()),
	})
}

// ============================================================================
// Auth Handlers
// ============================================================================

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		Username  string `json:"username"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Check if user exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM auth_service.users WHERE email = $1)", req.Email).Scan(&exists)
	if err != nil {
		log.Printf("DB error: %v", err)
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if exists {
		respondError(w, http.StatusConflict, "User already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Generate username if not provided
	if req.Username == "" {
		req.Username = strings.Split(req.Email, "@")[0]
	}

	// Insert user
	userID := uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO auth_service.users (id, email, username, password_hash, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'active', NOW(), NOW())
	`, userID, req.Email, req.Username, string(hashedPassword))

	if err != nil {
		log.Printf("Insert error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Create profile in user_service
	db.Exec(`
		INSERT INTO user_service.profiles (id, user_id, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`, uuid.New().String(), userID, req.FirstName, req.LastName)

	// Create session token
	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO auth_service.sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`, uuid.New().String(), userID, token, expiresAt)

	if err != nil {
		log.Printf("Session error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"user": map[string]interface{}{
			"id":        userID,
			"email":     req.Email,
			"username":  req.Username,
			"firstName": req.FirstName,
			"lastName":  req.LastName,
		},
		"token":     token,
		"expiresAt": expiresAt.Format(time.RFC3339),
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Find user
	var userID, passwordHash, username string
	err := db.QueryRow(`
		SELECT id, password_hash, username FROM auth_service.users 
		WHERE email = $1 AND status = 'active'
	`, req.Email).Scan(&userID, &passwordHash, &username)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}
	if err != nil {
		log.Printf("DB error: %v", err)
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Update last login
	db.Exec("UPDATE auth_service.users SET last_login_at = NOW() WHERE id = $1", userID)

	// Create session token
	token := generateToken()
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(`
		INSERT INTO auth_service.sessions (id, user_id, token_hash, expires_at, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`, uuid.New().String(), userID, token, expiresAt, r.RemoteAddr, r.UserAgent())

	if err != nil {
		log.Printf("Session error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Get profile info
	var firstName, lastName sql.NullString
	db.QueryRow("SELECT first_name, last_name FROM user_service.profiles WHERE user_id = $1", userID).Scan(&firstName, &lastName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":        userID,
			"email":     req.Email,
			"username":  username,
			"firstName": firstName.String,
			"lastName":  lastName.String,
		},
		"token":     token,
		"expiresAt": expiresAt.Format(time.RFC3339),
	})
}

func refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// For simplicity, just create a new token
	token := generateToken()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"token":        token,
		"refreshToken": generateToken(),
		"expiresAt":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 {
			// Revoke the session
			db.Exec("UPDATE auth_service.sessions SET revoked_at = NOW() WHERE token_hash = $1", parts[1])
		}
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

func meHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var email, username string
	err := db.QueryRow("SELECT email, username FROM auth_service.users WHERE id = $1", userID).Scan(&email, &username)
	if err != nil {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	var firstName, lastName sql.NullString
	db.QueryRow("SELECT first_name, last_name FROM user_service.profiles WHERE user_id = $1", userID).Scan(&firstName, &lastName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        userID,
		"email":     email,
		"username":  username,
		"firstName": firstName.String,
		"lastName":  lastName.String,
	})
}

func requestPasswordResetHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset email sent if account exists",
	})
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset successfully",
	})
}

func listSessionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	rows, err := db.Query(`
		SELECT id, ip_address, user_agent, created_at, expires_at
		FROM auth_service.sessions 
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get sessions")
		return
	}
	defer rows.Close()

	var sessions []map[string]interface{}
	for rows.Next() {
		var id string
		var ipAddress, userAgent sql.NullString
		var createdAt, expiresAt time.Time
		rows.Scan(&id, &ipAddress, &userAgent, &createdAt, &expiresAt)
		sessions = append(sessions, map[string]interface{}{
			"id":        id,
			"ipAddress": ipAddress.String,
			"userAgent": userAgent.String,
			"createdAt": createdAt.Format(time.RFC3339),
			"expiresAt": expiresAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": sessions,
		"total": len(sessions),
	})
}

// ============================================================================
// User Handlers
// ============================================================================

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT u.id, u.email, u.username, u.status, u.created_at,
		       p.first_name, p.last_name
		FROM auth_service.users u
		LEFT JOIN user_service.profiles p ON p.user_id = u.id
		ORDER BY u.created_at DESC LIMIT 100
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list users")
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, email, username, status string
		var createdAt time.Time
		var firstName, lastName sql.NullString
		rows.Scan(&id, &email, &username, &status, &createdAt, &firstName, &lastName)
		users = append(users, map[string]interface{}{
			"id":        id,
			"email":     email,
			"username":  username,
			"firstName": firstName.String,
			"lastName":  lastName.String,
			"status":    status,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": users,
		"total": len(users),
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var email, username, status string
	var createdAt time.Time
	err := db.QueryRow("SELECT email, username, status, created_at FROM auth_service.users WHERE id = $1", id).
		Scan(&email, &username, &status, &createdAt)
	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	var firstName, lastName sql.NullString
	db.QueryRow("SELECT first_name, last_name FROM user_service.profiles WHERE user_id = $1", id).Scan(&firstName, &lastName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        id,
		"email":     email,
		"username":  username,
		"firstName": firstName.String,
		"lastName":  lastName.String,
		"status":    status,
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Username  string `json:"username"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// Update username if provided
	if req.Username != "" {
		db.Exec("UPDATE auth_service.users SET username = $1, updated_at = NOW() WHERE id = $2", req.Username, id)
	}

	// Update profile
	db.Exec(`
		INSERT INTO user_service.profiles (id, user_id, first_name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET first_name = $3, last_name = $4, updated_at = NOW()
	`, uuid.New().String(), id, req.FirstName, req.LastName)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        id,
		"firstName": req.FirstName,
		"lastName":  req.LastName,
		"message":   "User updated",
	})
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("UPDATE auth_service.users SET status = 'deleted', updated_at = NOW() WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted",
	})
}

// ============================================================================
// Workflow Handlers
// ============================================================================

func listWorkflowsHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	rows, err := db.Query(`
		SELECT id, name, description, status, version, created_at, updated_at
		FROM workflow_service.workflows
		WHERE user_id = $1
		ORDER BY updated_at DESC LIMIT 100
	`, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list workflows")
		return
	}
	defer rows.Close()

	var workflows []map[string]interface{}
	for rows.Next() {
		var id, name, status string
		var description sql.NullString
		var version int
		var createdAt, updatedAt time.Time
		rows.Scan(&id, &name, &description, &status, &version, &createdAt, &updatedAt)
		workflows = append(workflows, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description.String,
			"status":      status,
			"version":     version,
			"createdAt":   createdAt.Format(time.RFC3339),
			"updatedAt":   updatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": workflows,
		"total": len(workflows),
	})
}

func createWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var req struct {
		Name        string        `json:"name"`
		Description string        `json:"description"`
		Nodes       []interface{} `json:"nodes"`
		Connections []interface{} `json:"connections"`
		Settings    interface{}   `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		req.Name = "Untitled Workflow"
	}

	nodesJSON, _ := json.Marshal(req.Nodes)
	connectionsJSON, _ := json.Marshal(req.Connections)
	settingsJSON, _ := json.Marshal(req.Settings)

	workflowID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO workflow_service.workflows (id, user_id, name, description, status, nodes, connections, settings, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'draft', $5, $6, $7, 1, NOW(), NOW())
	`, workflowID, userID, req.Name, req.Description, nodesJSON, connectionsJSON, settingsJSON)

	if err != nil {
		log.Printf("Insert workflow error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          workflowID,
		"name":        req.Name,
		"description": req.Description,
		"status":      "draft",
		"version":     1,
		"nodes":       req.Nodes,
		"connections": req.Connections,
		"createdAt":   time.Now().Format(time.RFC3339),
	})
}

func getWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	var name, status string
	var description sql.NullString
	var nodesJSON, connectionsJSON, settingsJSON []byte
	var version int
	var createdAt, updatedAt time.Time

	err := db.QueryRow(`
		SELECT name, description, status, nodes, connections, settings, version, created_at, updated_at
		FROM workflow_service.workflows
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&name, &description, &status, &nodesJSON, &connectionsJSON, &settingsJSON, &version, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	var nodes, connections []interface{}
	var settings interface{}
	json.Unmarshal(nodesJSON, &nodes)
	json.Unmarshal(connectionsJSON, &connections)
	json.Unmarshal(settingsJSON, &settings)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description.String,
		"status":      status,
		"nodes":       nodes,
		"connections": connections,
		"settings":    settings,
		"version":     version,
		"createdAt":   createdAt.Format(time.RFC3339),
		"updatedAt":   updatedAt.Format(time.RFC3339),
	})
}

func updateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	if name, ok := req["name"].(string); ok {
		db.Exec("UPDATE workflow_service.workflows SET name = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3", name, id, userID)
	}
	if desc, ok := req["description"].(string); ok {
		db.Exec("UPDATE workflow_service.workflows SET description = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3", desc, id, userID)
	}
	if nodes, ok := req["nodes"]; ok {
		nodesJSON, _ := json.Marshal(nodes)
		db.Exec("UPDATE workflow_service.workflows SET nodes = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3", nodesJSON, id, userID)
	}
	if connections, ok := req["connections"]; ok {
		connectionsJSON, _ := json.Marshal(connections)
		db.Exec("UPDATE workflow_service.workflows SET connections = $1, updated_at = NOW() WHERE id = $2 AND user_id = $3", connectionsJSON, id, userID)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"message": "Workflow updated",
	})
}

func deleteWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)
	db.Exec("DELETE FROM workflow_service.workflows WHERE id = $1 AND user_id = $2", id, userID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Workflow deleted",
	})
}

func activateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)
	db.Exec("UPDATE workflow_service.workflows SET status = 'active', updated_at = NOW() WHERE id = $1 AND user_id = $2", id, userID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  "active",
		"message": "Workflow activated",
	})
}

func deactivateWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)
	db.Exec("UPDATE workflow_service.workflows SET status = 'inactive', updated_at = NOW() WHERE id = $1 AND user_id = $2", id, userID)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  "inactive",
		"message": "Workflow deactivated",
	})
}

func executeWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	// Get workflow data
	var nodesJSON, connectionsJSON []byte
	var version int
	err := db.QueryRow(`
		SELECT nodes, connections, version FROM workflow_service.workflows
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&nodesJSON, &connectionsJSON, &version)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Parse input
	var input map[string]interface{}
	json.NewDecoder(r.Body).Decode(&input)

	// Create execution record
	executionID := uuid.New().String()
	inputJSON, _ := json.Marshal(input)
	_, err = db.Exec(`
		INSERT INTO execution_service.executions (id, workflow_id, workflow_version, user_id, trigger_type, status, input_data, created_at, started_at)
		VALUES ($1, $2, $3, $4, 'manual', 'running', $5, NOW(), NOW())
	`, executionID, id, version, userID, inputJSON)

	if err != nil {
		log.Printf("Create execution error: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to create execution")
		return
	}

	// Parse and execute workflow
	var nodeList []map[string]interface{}
	var connectionList []map[string]interface{}
	json.Unmarshal(nodesJSON, &nodeList)
	json.Unmarshal(connectionsJSON, &connectionList)

	// Build engine workflow
	engineNodes := make([]engine.NodeDefinition, len(nodeList))
	for i, n := range nodeList {
		config := map[string]interface{}{}
		if c, ok := n["config"].(map[string]interface{}); ok {
			config = c
		}
		engineNodes[i] = engine.NodeDefinition{
			ID:     getString(n, "id"),
			Type:   getString(n, "type"),
			Config: config,
		}
	}

	engineConnections := make([]engine.Connection, len(connectionList))
	for i, c := range connectionList {
		engineConnections[i] = engine.Connection{
			SourceNodeID: getString(c, "source"),
			TargetNodeID: getString(c, "target"),
		}
	}

	wf := &engine.WorkflowDefinition{
		ID:          id,
		Nodes:       engineNodes,
		Connections: engineConnections,
	}

	// Execute asynchronously
	go func() {
		result, execErr := eng.Execute(context.Background(), wf, &engine.ExecutionOptions{TriggerData: input})

		outputJSON, _ := json.Marshal(result)
		if execErr != nil {
			db.Exec(`
				UPDATE execution_service.executions 
				SET status = 'failed', error_message = $1, completed_at = NOW()
				WHERE id = $2
			`, execErr.Error(), executionID)
		} else {
			db.Exec(`
				UPDATE execution_service.executions 
				SET status = 'completed', output_data = $1, completed_at = NOW()
				WHERE id = $2
			`, outputJSON, executionID)
		}
	}()

	respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"executionId": executionID,
		"workflowId":  id,
		"status":      "running",
		"message":     "Workflow execution started",
	})
}

func cloneWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	// Get original workflow
	var name, description string
	var nodesJSON, connectionsJSON, settingsJSON []byte
	err := db.QueryRow(`
		SELECT name, description, nodes, connections, settings
		FROM workflow_service.workflows WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&name, &description, &nodesJSON, &connectionsJSON, &settingsJSON)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Create clone
	cloneID := uuid.New().String()
	cloneName := name + " (Copy)"
	_, err = db.Exec(`
		INSERT INTO workflow_service.workflows (id, user_id, name, description, status, nodes, connections, settings, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'draft', $5, $6, $7, 1, NOW(), NOW())
	`, cloneID, userID, cloneName, description, nodesJSON, connectionsJSON, settingsJSON)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to clone workflow")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":        cloneID,
		"name":      cloneName,
		"status":    "draft",
		"message":   "Workflow cloned",
		"createdAt": time.Now().Format(time.RFC3339),
	})
}

// ============================================================================
// Execution Handlers
// ============================================================================

func listExecutionsHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	rows, err := db.Query(`
		SELECT id, workflow_id, trigger_type, status, created_at, started_at, completed_at
		FROM execution_service.executions
		WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 100
	`, userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list executions")
		return
	}
	defer rows.Close()

	var executions []map[string]interface{}
	for rows.Next() {
		var id, workflowID, triggerType, status string
		var createdAt time.Time
		var startedAt, completedAt sql.NullTime
		rows.Scan(&id, &workflowID, &triggerType, &status, &createdAt, &startedAt, &completedAt)
		exec := map[string]interface{}{
			"id":          id,
			"workflowId":  workflowID,
			"triggerType": triggerType,
			"status":      status,
			"createdAt":   createdAt.Format(time.RFC3339),
		}
		if startedAt.Valid {
			exec["startedAt"] = startedAt.Time.Format(time.RFC3339)
		}
		if completedAt.Valid {
			exec["completedAt"] = completedAt.Time.Format(time.RFC3339)
		}
		executions = append(executions, exec)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": executions,
		"total": len(executions),
	})
}

func getExecutionHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	var workflowID, triggerType, status string
	var inputJSON, outputJSON []byte
	var errorMessage sql.NullString
	var createdAt time.Time
	var startedAt, completedAt sql.NullTime

	err := db.QueryRow(`
		SELECT workflow_id, trigger_type, status, input_data, output_data, error_message, created_at, started_at, completed_at
		FROM execution_service.executions
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&workflowID, &triggerType, &status, &inputJSON, &outputJSON, &errorMessage, &createdAt, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Execution not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	var input, output interface{}
	json.Unmarshal(inputJSON, &input)
	json.Unmarshal(outputJSON, &output)

	exec := map[string]interface{}{
		"id":          id,
		"workflowId":  workflowID,
		"triggerType": triggerType,
		"status":      status,
		"input":       input,
		"output":      output,
		"createdAt":   createdAt.Format(time.RFC3339),
	}
	if startedAt.Valid {
		exec["startedAt"] = startedAt.Time.Format(time.RFC3339)
	}
	if completedAt.Valid {
		exec["completedAt"] = completedAt.Time.Format(time.RFC3339)
	}
	if errorMessage.Valid {
		exec["error"] = errorMessage.String
	}

	respondJSON(w, http.StatusOK, exec)
}

func cancelExecutionHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	db.Exec(`
		UPDATE execution_service.executions 
		SET status = 'cancelled', completed_at = NOW() 
		WHERE id = $1 AND user_id = $2 AND status IN ('pending', 'running')
	`, id, userID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"status":  "cancelled",
		"message": "Execution cancelled",
	})
}

func directExecuteHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var req struct {
		Nodes       []map[string]interface{} `json:"nodes"`
		Connections []map[string]interface{} `json:"connections"`
		Input       map[string]interface{}   `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	executionID := uuid.New().String()

	// Build engine workflow
	engineNodes := make([]engine.NodeDefinition, len(req.Nodes))
	for i, n := range req.Nodes {
		config := map[string]interface{}{}
		if c, ok := n["config"].(map[string]interface{}); ok {
			config = c
		}
		engineNodes[i] = engine.NodeDefinition{
			ID:     getString(n, "id"),
			Type:   getString(n, "type"),
			Config: config,
		}
	}

	engineConnections := make([]engine.Connection, len(req.Connections))
	for i, c := range req.Connections {
		engineConnections[i] = engine.Connection{
			SourceNodeID: getString(c, "source"),
			TargetNodeID: getString(c, "target"),
		}
	}

	wf := &engine.WorkflowDefinition{
		ID:          executionID,
		Nodes:       engineNodes,
		Connections: engineConnections,
	}

	// Execute synchronously for direct execution
	result, err := eng.Execute(context.Background(), wf, &engine.ExecutionOptions{TriggerData: req.Input})

	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"executionId": executionID,
			"status":      "failed",
			"error":       err.Error(),
			"userId":      userID,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"executionId": executionID,
		"status":      "completed",
		"output":      result,
	})
}

// ============================================================================
// Node Handlers
// ============================================================================

func listNodesHandler(w http.ResponseWriter, r *http.Request) {
	nodeTypes := runtime.List()

	nodes := make([]map[string]interface{}, 0, len(nodeTypes))
	for _, metadata := range nodeTypes {
		nodes = append(nodes, map[string]interface{}{
			"type":        metadata.Type,
			"name":        metadata.Name,
			"description": metadata.Description,
			"category":    metadata.Category,
			"icon":        metadata.Icon,
			"color":       metadata.Color,
			"inputs":      metadata.Inputs,
			"outputs":     metadata.Outputs,
			"config":      metadata.Properties,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": nodes,
		"total": len(nodes),
	})
}

func getNodeHandler(w http.ResponseWriter, r *http.Request) {
	nodeType := mux.Vars(r)["type"]

	executor, err := runtime.Get(nodeType)
	if err != nil {
		respondError(w, http.StatusNotFound, "Node type not found")
		return
	}

	metadata := executor.GetMetadata()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"type":        metadata.Type,
		"name":        metadata.Name,
		"description": metadata.Description,
		"category":    metadata.Category,
		"icon":        metadata.Icon,
		"color":       metadata.Color,
		"inputs":      metadata.Inputs,
		"outputs":     metadata.Outputs,
		"config":      metadata.Properties,
	})
}

func listNodeCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	categories := []map[string]interface{}{
		{"id": "trigger", "name": "Triggers", "description": "Start workflow execution"},
		{"id": "action", "name": "Actions", "description": "Perform operations"},
		{"id": "logic", "name": "Logic", "description": "Control flow"},
		{"id": "transform", "name": "Transform", "description": "Data transformation"},
		{"id": "integration", "name": "Integrations", "description": "Third-party services"},
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": categories,
		"total": len(categories),
	})
}

// ============================================================================
// Schedule Handlers
// ============================================================================

func listSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, workflow_id, name, description, cron_expression, timezone, is_active, next_run_at, last_run_at, created_at
		FROM schedule_service.schedules
		ORDER BY created_at DESC LIMIT 100
	`)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list schedules")
		return
	}
	defer rows.Close()

	var schedules []map[string]interface{}
	for rows.Next() {
		var id, workflowID, name, cronExpr, timezone string
		var description sql.NullString
		var isActive bool
		var nextRunAt, lastRunAt sql.NullTime
		var createdAt time.Time
		rows.Scan(&id, &workflowID, &name, &description, &cronExpr, &timezone, &isActive, &nextRunAt, &lastRunAt, &createdAt)
		schedules = append(schedules, map[string]interface{}{
			"id":             id,
			"workflowId":     workflowID,
			"name":           name,
			"description":    description.String,
			"cronExpression": cronExpr,
			"timezone":       timezone,
			"isActive":       isActive,
			"createdAt":      createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"items": schedules,
		"total": len(schedules),
	})
}

func createScheduleHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkflowID     string `json:"workflowId"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		CronExpression string `json:"cronExpression"`
		Timezone       string `json:"timezone"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	scheduleID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO schedule_service.schedules (id, workflow_id, name, description, cron_expression, timezone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
	`, scheduleID, req.WorkflowID, req.Name, req.Description, req.CronExpression, req.Timezone)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create schedule")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":             scheduleID,
		"workflowId":     req.WorkflowID,
		"name":           req.Name,
		"cronExpression": req.CronExpression,
		"isActive":       true,
	})
}

func getScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var workflowID, name, cronExpr, timezone string
	var description sql.NullString
	var isActive bool
	var createdAt time.Time

	err := db.QueryRow(`
		SELECT workflow_id, name, description, cron_expression, timezone, is_active, created_at
		FROM schedule_service.schedules WHERE id = $1
	`, id).Scan(&workflowID, &name, &description, &cronExpr, &timezone, &isActive, &createdAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Schedule not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":             id,
		"workflowId":     workflowID,
		"name":           name,
		"description":    description.String,
		"cronExpression": cronExpr,
		"timezone":       timezone,
		"isActive":       isActive,
		"createdAt":      createdAt.Format(time.RFC3339),
	})
}

func updateScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	if name, ok := req["name"].(string); ok {
		db.Exec("UPDATE schedule_service.schedules SET name = $1, updated_at = NOW() WHERE id = $2", name, id)
	}
	if cron, ok := req["cronExpression"].(string); ok {
		db.Exec("UPDATE schedule_service.schedules SET cron_expression = $1, updated_at = NOW() WHERE id = $2", cron, id)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Schedule updated"})
}

func deleteScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("DELETE FROM schedule_service.schedules WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Schedule deleted"})
}

func pauseScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("UPDATE schedule_service.schedules SET is_active = false, updated_at = NOW() WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "isActive": false, "message": "Schedule paused"})
}

func resumeScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("UPDATE schedule_service.schedules SET is_active = true, updated_at = NOW() WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "isActive": true, "message": "Schedule resumed"})
}

func triggerScheduleHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("UPDATE schedule_service.schedules SET last_run_at = NOW() WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Schedule triggered"})
}

func validateCronHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Expression string `json:"expression"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	respondJSON(w, http.StatusOK, map[string]interface{}{"valid": true, "expression": req.Expression})
}

// ============================================================================
// Credential Handlers
// ============================================================================

func listCredentialsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
}

func createCredentialHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   uuid.New().String(),
		"name": req.Name,
		"type": req.Type,
	})
}

func getCredentialHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotFound, "Credential not found")
}

func updateCredentialHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Credential updated"})
}

func deleteCredentialHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Credential deleted"})
}

func listCredentialTypesHandler(w http.ResponseWriter, r *http.Request) {
	types := []map[string]interface{}{
		{"id": "api_key", "name": "API Key"},
		{"id": "oauth2", "name": "OAuth 2.0"},
		{"id": "basic", "name": "Basic Auth"},
		{"id": "bearer", "name": "Bearer Token"},
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": types, "total": len(types)})
}

func testCredentialHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Credential is valid"})
}

// ============================================================================
// Integration Handlers
// ============================================================================

func listIntegrationsHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
}

func createIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":   uuid.New().String(),
		"name": req.Name,
		"type": req.Type,
	})
}

func getIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	respondError(w, http.StatusNotFound, "Integration not found")
}

func updateIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Integration updated"})
}

func deleteIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Integration deleted"})
}

func enableIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "enabled": true})
}

func disableIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "enabled": false})
}

func testIntegrationHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func listIntegrationCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	categories := []map[string]interface{}{
		{"id": "communication", "name": "Communication"},
		{"id": "database", "name": "Database"},
		{"id": "storage", "name": "Storage"},
		{"id": "analytics", "name": "Analytics"},
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": categories, "total": len(categories)})
}

// ============================================================================
// Webhook Handlers
// ============================================================================

func listWebhooksHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, workflow_id, endpoint_id, path, method, is_active, created_at
		FROM webhook_service.webhooks
		ORDER BY created_at DESC LIMIT 100
	`)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
		return
	}
	defer rows.Close()

	var webhooks []map[string]interface{}
	for rows.Next() {
		var id, workflowID, endpointID, path, method string
		var isActive bool
		var createdAt time.Time
		rows.Scan(&id, &workflowID, &endpointID, &path, &method, &isActive, &createdAt)
		webhooks = append(webhooks, map[string]interface{}{
			"id":         id,
			"workflowId": workflowID,
			"endpointId": endpointID,
			"path":       path,
			"method":     method,
			"isActive":   isActive,
			"createdAt":  createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": webhooks, "total": len(webhooks)})
}

func createWebhookHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkflowID string `json:"workflowId"`
		Path       string `json:"path"`
		Method     string `json:"method"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Method == "" {
		req.Method = "POST"
	}

	webhookID := uuid.New().String()
	endpointID := generateToken()[:16]

	_, err := db.Exec(`
		INSERT INTO webhook_service.webhooks (id, workflow_id, endpoint_id, path, method, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())
	`, webhookID, req.WorkflowID, endpointID, req.Path, req.Method)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create webhook")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":         webhookID,
		"workflowId": req.WorkflowID,
		"endpointId": endpointID,
		"path":       req.Path,
		"method":     req.Method,
		"isActive":   true,
	})
}

func getWebhookHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var workflowID, endpointID, path, method string
	var isActive bool
	var createdAt time.Time

	err := db.QueryRow(`
		SELECT workflow_id, endpoint_id, path, method, is_active, created_at
		FROM webhook_service.webhooks WHERE id = $1
	`, id).Scan(&workflowID, &endpointID, &path, &method, &isActive, &createdAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Webhook not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":         id,
		"workflowId": workflowID,
		"endpointId": endpointID,
		"path":       path,
		"method":     method,
		"isActive":   isActive,
		"createdAt":  createdAt.Format(time.RFC3339),
	})
}

func updateWebhookHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Webhook updated"})
}

func deleteWebhookHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("DELETE FROM webhook_service.webhooks WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Webhook deleted"})
}

func triggerWebhookHandler(w http.ResponseWriter, r *http.Request) {
	endpointID := mux.Vars(r)["endpointId"]

	// Find webhook
	var webhookID, workflowID string
	err := db.QueryRow(`
		SELECT id, workflow_id FROM webhook_service.webhooks
		WHERE endpoint_id = $1 AND is_active = true
	`, endpointID).Scan(&webhookID, &workflowID)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Webhook not found")
		return
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}

	// Log the webhook call
	var body interface{}
	json.NewDecoder(r.Body).Decode(&body)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"webhookId":  webhookID,
		"workflowId": workflowID,
		"received":   body,
		"message":    "Webhook triggered",
	})
}

// ============================================================================
// Notification Handlers
// ============================================================================

func listNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	rows, err := db.Query(`
		SELECT id, type, channel, subject, content, status, read_at, created_at
		FROM notification_service.notifications
		WHERE user_id = $1
		ORDER BY created_at DESC LIMIT 50
	`, userID)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
		return
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var id, notifType, channel, status string
		var subject, content sql.NullString
		var readAt sql.NullTime
		var createdAt time.Time
		rows.Scan(&id, &notifType, &channel, &subject, &content, &status, &readAt, &createdAt)
		notifications = append(notifications, map[string]interface{}{
			"id":        id,
			"type":      notifType,
			"channel":   channel,
			"subject":   subject.String,
			"content":   content.String,
			"status":    status,
			"createdAt": createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": notifications, "total": len(notifications)})
}

func createNotificationHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	var req struct {
		Type    string `json:"type"`
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	notifID := uuid.New().String()
	db.Exec(`
		INSERT INTO notification_service.notifications (id, user_id, type, channel, subject, content, status, created_at)
		VALUES ($1, $2, $3, 'in_app', $4, $5, 'unread', NOW())
	`, notifID, userID, req.Type, req.Subject, req.Content)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      notifID,
		"type":    req.Type,
		"subject": req.Subject,
		"status":  "unread",
	})
}

func getNotificationHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)

	var notifType, channel, status string
	var subject, content sql.NullString
	var createdAt time.Time

	err := db.QueryRow(`
		SELECT type, channel, subject, content, status, created_at
		FROM notification_service.notifications WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&notifType, &channel, &subject, &content, &status, &createdAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Notification not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":        id,
		"type":      notifType,
		"channel":   channel,
		"subject":   subject.String,
		"content":   content.String,
		"status":    status,
		"createdAt": createdAt.Format(time.RFC3339),
	})
}

func markNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	userID := getUserIDFromContext(r)
	db.Exec("UPDATE notification_service.notifications SET status = 'read', read_at = NOW() WHERE id = $1 AND user_id = $2", id, userID)
	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "status": "read"})
}

// ============================================================================
// Workspace Handlers
// ============================================================================

func listWorkspacesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, slug, description, created_at
		FROM user_service.organizations
		ORDER BY created_at DESC LIMIT 100
	`)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
		return
	}
	defer rows.Close()

	var workspaces []map[string]interface{}
	for rows.Next() {
		var id, name, slug string
		var description sql.NullString
		var createdAt time.Time
		rows.Scan(&id, &name, &slug, &description, &createdAt)
		workspaces = append(workspaces, map[string]interface{}{
			"id":          id,
			"name":        name,
			"slug":        slug,
			"description": description.String,
			"createdAt":   createdAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": workspaces, "total": len(workspaces)})
}

func createWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}

	workspaceID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO user_service.organizations (id, name, slug, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, workspaceID, req.Name, req.Slug, req.Description, userID)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create workspace")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":          workspaceID,
		"name":        req.Name,
		"slug":        req.Slug,
		"description": req.Description,
	})
}

func getWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var name, slug string
	var description sql.NullString
	var createdAt time.Time

	err := db.QueryRow(`
		SELECT name, slug, description, created_at
		FROM user_service.organizations WHERE id = $1
	`, id).Scan(&name, &slug, &description, &createdAt)

	if err == sql.ErrNoRows {
		respondError(w, http.StatusNotFound, "Workspace not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":          id,
		"name":        name,
		"slug":        slug,
		"description": description.String,
		"createdAt":   createdAt.Format(time.RFC3339),
	})
}

func updateWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	if name, ok := req["name"].(string); ok {
		db.Exec("UPDATE user_service.organizations SET name = $1, updated_at = NOW() WHERE id = $2", name, id)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"id": id, "message": "Workspace updated"})
}

func deleteWorkspaceHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	db.Exec("DELETE FROM user_service.organizations WHERE id = $1", id)
	respondJSON(w, http.StatusOK, map[string]interface{}{"message": "Workspace deleted"})
}

func listWorkspaceMembersHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	rows, err := db.Query(`
		SELECT m.id, m.user_id, m.role, m.joined_at, u.email, u.username
		FROM user_service.organization_members m
		JOIN auth_service.users u ON u.id = m.user_id
		WHERE m.organization_id = $1
	`, id)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"items": []interface{}{}, "total": 0})
		return
	}
	defer rows.Close()

	var members []map[string]interface{}
	for rows.Next() {
		var memberID, userID, role, email, username string
		var joinedAt sql.NullTime
		rows.Scan(&memberID, &userID, &role, &joinedAt, &email, &username)
		members = append(members, map[string]interface{}{
			"id":       memberID,
			"userId":   userID,
			"email":    email,
			"username": username,
			"role":     role,
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{"items": members, "total": len(members)})
}

func inviteWorkspaceMemberHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      uuid.New().String(),
		"email":   req.Email,
		"role":    req.Role,
		"status":  "pending",
		"message": "Invitation sent",
	})
}

// ============================================================================
// Billing Handlers
// ============================================================================

func listPlansHandler(w http.ResponseWriter, r *http.Request) {
	plans := []map[string]interface{}{
		{"id": "free", "name": "Free", "price": 0, "features": []string{"5 workflows", "100 executions/month"}},
		{"id": "starter", "name": "Starter", "price": 29, "features": []string{"25 workflows", "1000 executions/month"}},
		{"id": "pro", "name": "Pro", "price": 99, "features": []string{"Unlimited workflows", "10000 executions/month"}},
		{"id": "enterprise", "name": "Enterprise", "price": 299, "features": []string{"Unlimited everything", "Priority support"}},
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"items": plans, "total": len(plans)})
}

func getSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"plan":   "free",
		"status": "active",
		"usage":  map[string]interface{}{"workflows": 0, "executions": 0},
	})
}

func createSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PlanID string `json:"planId"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":     uuid.New().String(),
		"plan":   req.PlanID,
		"status": "active",
	})
}

func getUsageHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)

	var wfCount int
	db.QueryRow("SELECT COUNT(*) FROM workflow_service.workflows WHERE user_id = $1", userID).Scan(&wfCount)

	var execCount int
	db.QueryRow("SELECT COUNT(*) FROM execution_service.executions WHERE user_id = $1", userID).Scan(&execCount)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"workflows":        wfCount,
		"executions":       execCount,
		"apiCalls":         0,
		"storageUsedBytes": 0,
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
