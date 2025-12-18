package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
)

// RecoveryConfig holds recovery middleware configuration
type RecoveryConfig struct {
	Logger         Logger
	StackTrace     bool
	PrintStack     bool
	ErrorHandler   func(w http.ResponseWriter, r *http.Request, err interface{})
}

// DefaultRecoveryConfig returns default recovery configuration
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		StackTrace: true,
		PrintStack: false,
	}
}

// Recovery creates panic recovery middleware
func Recovery(config *RecoveryConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultRecoveryConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					var stack string
					if config.StackTrace {
						stack = string(debug.Stack())
					}

					// Log the panic
					if config.Logger != nil {
						config.Logger.Error("panic recovered",
							"error", err,
							"path", r.URL.Path,
							"method", r.Method,
							"stack", stack,
						)
					}

					// Print stack if configured
					if config.PrintStack {
						fmt.Printf("Panic: %v\n%s\n", err, stack)
					}

					// Handle error response
					if config.ErrorHandler != nil {
						config.ErrorHandler(w, r, err)
					} else {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusInternalServerError)
						json.NewEncoder(w).Encode(map[string]interface{}{
							"success": false,
							"error": map[string]string{
								"code":    "INTERNAL_ERROR",
								"message": "An unexpected error occurred",
							},
						})
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryWithLogger creates recovery middleware with a logger
func RecoveryWithLogger(logger Logger) func(http.Handler) http.Handler {
	return Recovery(&RecoveryConfig{
		Logger:     logger,
		StackTrace: true,
	})
}

// SimpleRecovery creates a simple recovery middleware
func SimpleRecovery(next http.Handler) http.Handler {
	return Recovery(nil)(next)
}
