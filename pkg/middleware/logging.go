package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logger is a logging interface
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
}

// LoggingConfig holds logging middleware configuration
type LoggingConfig struct {
	Logger          Logger
	SkipPaths       []string
	LogRequestBody  bool
	LogResponseBody bool
	MaxBodySize     int
}

// Logging creates request logging middleware
func Logging(config *LoggingConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Skip logging for certain paths
			for _, path := range config.SkipPaths {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Get or generate request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer
			rw := newResponseWriter(w)

			// Log request body if enabled
			var requestBody string
			if config.LogRequestBody && r.Body != nil {
				bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, int64(config.MaxBodySize)))
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				requestBody = string(bodyBytes)
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log the request
			if config.Logger != nil {
				logFields := []interface{}{
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration_ms", duration.Milliseconds(),
					"size", rw.size,
					"remote_addr", r.RemoteAddr,
					"user_agent", r.UserAgent(),
				}

				if requestBody != "" {
					logFields = append(logFields, "request_body", requestBody)
				}

				if userID := GetUserID(r.Context()); userID != "" {
					logFields = append(logFields, "user_id", userID)
				}

				if rw.statusCode >= 500 {
					config.Logger.Error("HTTP request", logFields...)
				} else if rw.statusCode >= 400 {
					config.Logger.Info("HTTP request", logFields...)
				} else {
					config.Logger.Debug("HTTP request", logFields...)
				}
			}
		})
	}
}

// RequestID adds request ID to requests
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		ctx := r.Context()
		ctx = SetRequestID(ctx, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetRequestID sets request ID in context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextRequestID, requestID)
}

// GetRequestID gets request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(ContextRequestID).(string); ok {
		return requestID
	}
	return ""
}

// AccessLog creates simple access log middleware
func AccessLog(logger Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			next.ServeHTTP(rw, r)

			if logger != nil {
				logger.Info("access",
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration", time.Since(start).String(),
				)
			}
		})
	}
}
