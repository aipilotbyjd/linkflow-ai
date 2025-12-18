// Package engine provides retry and error handling
package engine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	JitterFactor    float64
	RetryableErrors []error
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
	}
}

// RetryFunc is a function that can be retried
type RetryFunc func(ctx context.Context, attempt int) error

// Retry executes a function with retry logic
func Retry(ctx context.Context, config *RetryConfig, fn RetryFunc) error {
	if config == nil {
		config = DefaultRetryConfig()
	}

	var lastErr error
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn(ctx, attempt)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryable(err, config.RetryableErrors) {
			return err
		}

		// Don't sleep after last attempt
		if attempt < config.MaxAttempts {
			delay := calculateDelay(config, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRetryable(err error, retryableErrors []error) bool {
	if len(retryableErrors) == 0 {
		// If no specific errors, retry all
		return true
	}

	for _, re := range retryableErrors {
		if errors.Is(err, re) {
			return true
		}
	}

	return false
}

func calculateDelay(config *RetryConfig, attempt int) time.Duration {
	// Exponential backoff
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))

	// Add jitter
	if config.JitterFactor > 0 {
		jitter := delay * config.JitterFactor
		delay = delay + (rand.Float64()*2-1)*jitter
	}

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// ErrorHandler handles execution errors
type ErrorHandler struct {
	policy      ErrorPolicy
	onError     ErrorCallback
	fallback    FallbackFunc
	maxErrors   int
	errorCount  int
	errorWindow time.Duration
	errors      []errorRecord
}

// ErrorPolicy defines how to handle errors
type ErrorPolicy string

const (
	ErrorPolicyStop          ErrorPolicy = "stop"           // Stop execution on error
	ErrorPolicyContinue      ErrorPolicy = "continue"       // Continue execution on error
	ErrorPolicyRetry         ErrorPolicy = "retry"          // Retry failed node
	ErrorPolicyFallback      ErrorPolicy = "fallback"       // Execute fallback
	ErrorPolicyCircuitBreaker ErrorPolicy = "circuit_breaker" // Use circuit breaker
)

// ErrorCallback is called when an error occurs
type ErrorCallback func(err error, context *ErrorContext)

// FallbackFunc is executed when primary fails
type FallbackFunc func(ctx context.Context, err error) (map[string]interface{}, error)

// ErrorContext provides context about the error
type ErrorContext struct {
	ExecutionID string
	WorkflowID  string
	NodeID      string
	NodeType    string
	Attempt     int
	Error       error
	Timestamp   time.Time
}

type errorRecord struct {
	err       error
	timestamp time.Time
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(policy ErrorPolicy) *ErrorHandler {
	return &ErrorHandler{
		policy:      policy,
		maxErrors:   10,
		errorWindow: 5 * time.Minute,
		errors:      make([]errorRecord, 0),
	}
}

// WithCallback sets the error callback
func (h *ErrorHandler) WithCallback(callback ErrorCallback) *ErrorHandler {
	h.onError = callback
	return h
}

// WithFallback sets the fallback function
func (h *ErrorHandler) WithFallback(fallback FallbackFunc) *ErrorHandler {
	h.fallback = fallback
	return h
}

// WithMaxErrors sets max errors before circuit opens
func (h *ErrorHandler) WithMaxErrors(max int) *ErrorHandler {
	h.maxErrors = max
	return h
}

// Handle handles an error
func (h *ErrorHandler) Handle(ctx context.Context, errCtx *ErrorContext) (map[string]interface{}, error) {
	// Record error
	h.errors = append(h.errors, errorRecord{
		err:       errCtx.Error,
		timestamp: errCtx.Timestamp,
	})

	// Clean old errors
	h.cleanOldErrors()

	// Call callback if set
	if h.onError != nil {
		h.onError(errCtx.Error, errCtx)
	}

	// Handle based on policy
	switch h.policy {
	case ErrorPolicyStop:
		return nil, errCtx.Error

	case ErrorPolicyContinue:
		// Return empty output and continue
		return map[string]interface{}{
			"_error": errCtx.Error.Error(),
		}, nil

	case ErrorPolicyRetry:
		// Let the caller handle retry
		return nil, &RetryableError{Err: errCtx.Error}

	case ErrorPolicyFallback:
		if h.fallback != nil {
			return h.fallback(ctx, errCtx.Error)
		}
		return nil, errCtx.Error

	case ErrorPolicyCircuitBreaker:
		if h.isCircuitOpen() {
			return nil, &CircuitOpenError{}
		}
		return nil, errCtx.Error

	default:
		return nil, errCtx.Error
	}
}

func (h *ErrorHandler) cleanOldErrors() {
	cutoff := time.Now().Add(-h.errorWindow)
	var newErrors []errorRecord
	for _, e := range h.errors {
		if e.timestamp.After(cutoff) {
			newErrors = append(newErrors, e)
		}
	}
	h.errors = newErrors
}

func (h *ErrorHandler) isCircuitOpen() bool {
	h.cleanOldErrors()
	return len(h.errors) >= h.maxErrors
}

// GetErrorCount returns recent error count
func (h *ErrorHandler) GetErrorCount() int {
	h.cleanOldErrors()
	return len(h.errors)
}

// RetryableError indicates an error that can be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// CircuitOpenError indicates circuit breaker is open
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name        string
	state       CircuitState
	failCount   int
	successCount int
	lastFailure time.Time
	
	maxFailures    int
	resetTimeout   time.Duration
	halfOpenMax    int
}

// CircuitState represents circuit state
type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"
	CircuitStateOpen     CircuitState = "open"
	CircuitStateHalfOpen CircuitState = "half_open"
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures  int
	ResetTimeout time.Duration
	HalfOpenMax  int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	if config == nil {
		config = &CircuitBreakerConfig{
			MaxFailures:  5,
			ResetTimeout: 30 * time.Second,
			HalfOpenMax:  1,
		}
	}

	return &CircuitBreaker{
		name:         name,
		state:        CircuitStateClosed,
		maxFailures:  config.MaxFailures,
		resetTimeout: config.ResetTimeout,
		halfOpenMax:  config.HalfOpenMax,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.canExecute() {
		return &CircuitOpenError{}
	}

	err := fn()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	switch cb.state {
	case CircuitStateClosed:
		return true
	case CircuitStateOpen:
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = CircuitStateHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case CircuitStateHalfOpen:
		return cb.successCount < cb.halfOpenMax
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failCount++
	cb.lastFailure = time.Now()

	if cb.state == CircuitStateHalfOpen {
		cb.state = CircuitStateOpen
		return
	}

	if cb.failCount >= cb.maxFailures {
		cb.state = CircuitStateOpen
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.successCount++

	if cb.state == CircuitStateHalfOpen && cb.successCount >= cb.halfOpenMax {
		cb.state = CircuitStateClosed
		cb.failCount = 0
	}
}

// State returns the current state
func (cb *CircuitBreaker) State() CircuitState {
	return cb.state
}

// Reset resets the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.state = CircuitStateClosed
	cb.failCount = 0
	cb.successCount = 0
}

// ExecutionError wraps execution errors with context
type ExecutionError struct {
	ExecutionID string
	WorkflowID  string
	NodeID      string
	NodeType    string
	Message     string
	Cause       error
	Timestamp   time.Time
	Recoverable bool
}

func (e *ExecutionError) Error() string {
	if e.NodeID != "" {
		return fmt.Sprintf("execution error in node %s (%s): %s", e.NodeID, e.NodeType, e.Message)
	}
	return fmt.Sprintf("execution error: %s", e.Message)
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// NewExecutionError creates a new execution error
func NewExecutionError(executionID, workflowID, nodeID, nodeType, message string, cause error) *ExecutionError {
	return &ExecutionError{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Message:     message,
		Cause:       cause,
		Timestamp:   time.Now(),
		Recoverable: true,
	}
}
