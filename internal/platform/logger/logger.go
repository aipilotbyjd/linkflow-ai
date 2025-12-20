package logger

import (
	"context"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
)

// Logger interface for structured logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	WithFields(fields map[string]interface{}) Logger
	WithContext(ctx context.Context) Logger
}

// ZapLogger wraps zap.Logger
type ZapLogger struct {
	logger *zap.SugaredLogger
	fields map[string]interface{}
}

// New creates a new logger instance
func New(cfg config.LoggerConfig) Logger {
	var zapConfig zap.Config

	if cfg.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Set output paths (default to stdout if not specified)
	if cfg.OutputPath == "" || cfg.OutputPath == "stdout" {
		zapConfig.OutputPaths = []string{"stdout"}
	} else {
		zapConfig.OutputPaths = []string{cfg.OutputPath}
	}

	// Build logger
	logger, err := zapConfig.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		panic(err)
	}

	return &ZapLogger{
		logger: logger.Sugar(),
		fields: make(map[string]interface{}),
	}
}

// Debug logs a debug message
func (l *ZapLogger) Debug(msg string, fields ...interface{}) {
	l.logger.With(l.flattenFields()...).Debugw(msg, fields...)
}

// Info logs an info message
func (l *ZapLogger) Info(msg string, fields ...interface{}) {
	l.logger.With(l.flattenFields()...).Infow(msg, fields...)
}

// Warn logs a warning message
func (l *ZapLogger) Warn(msg string, fields ...interface{}) {
	l.logger.With(l.flattenFields()...).Warnw(msg, fields...)
}

// Error logs an error message
func (l *ZapLogger) Error(msg string, fields ...interface{}) {
	l.logger.With(l.flattenFields()...).Errorw(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *ZapLogger) Fatal(msg string, fields ...interface{}) {
	l.logger.With(l.flattenFields()...).Fatalw(msg, fields...)
	os.Exit(1)
}

// WithFields returns a new logger with additional fields
func (l *ZapLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}
	// Add new fields
	for k, v := range fields {
		newFields[k] = v
	}

	return &ZapLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// WithContext returns a new logger with context values
func (l *ZapLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})

	// Extract common context values
	if requestID := ctx.Value("requestID"); requestID != nil {
		fields["request_id"] = requestID
	}
	if userID := ctx.Value("userID"); userID != nil {
		fields["user_id"] = userID
	}
	if correlationID := ctx.Value("correlationID"); correlationID != nil {
		fields["correlation_id"] = correlationID
	}
	if traceID := ctx.Value("traceID"); traceID != nil {
		fields["trace_id"] = traceID
	}

	return l.WithFields(fields)
}

// flattenFields converts map to slice for zap
func (l *ZapLogger) flattenFields() []interface{} {
	fields := make([]interface{}, 0, len(l.fields)*2)
	for k, v := range l.fields {
		fields = append(fields, k, v)
	}
	return fields
}

// Middleware for HTTP logging
func HTTPMiddleware(logger Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
			}

			// Log request
			logger.WithFields(map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_addr": r.RemoteAddr,
				"user_agent": r.UserAgent(),
			}).Debug("HTTP request started")

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log response
			duration := time.Since(start)
			logger.WithFields(map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"bytes":      wrapped.bytesWritten,
			}).Info("HTTP request completed")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}
