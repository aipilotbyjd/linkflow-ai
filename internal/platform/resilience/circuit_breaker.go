package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned when the circuit breaker is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the state of the circuit breaker
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	name            string
	state           State
	failures        int
	successes       int
	lastFailureTime time.Time
	lastStateChange time.Time

	// Configuration
	maxFailures     int
	timeout         time.Duration
	halfOpenSuccess int

	// Callbacks
	onStateChange func(name string, from, to State)
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Name            string
	MaxFailures     int
	Timeout         time.Duration
	HalfOpenSuccess int
	OnStateChange   func(name string, from, to State)
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:            name,
		MaxFailures:     5,
		Timeout:         30 * time.Second,
		HalfOpenSuccess: 3,
	}
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:            config.Name,
		state:           StateClosed,
		maxFailures:     config.MaxFailures,
		timeout:         config.Timeout,
		halfOpenSuccess: config.HalfOpenSuccess,
		onStateChange:   config.OnStateChange,
		lastStateChange: time.Now(),
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.canExecute() {
		return ErrCircuitOpen
	}

	// Execute the function
	err := fn()

	// Record the result
	cb.recordResult(err)

	return err
}

// ExecuteWithFallback runs the function with a fallback on circuit open
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, fn func() error, fallback func() error) error {
	if !cb.canExecute() {
		return fallback()
	}

	err := fn()
	cb.recordResult(err)

	if err != nil && cb.State() == StateOpen {
		return fallback()
	}

	return err
}

// canExecute checks if a request can be executed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of an execution
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}
}

// onFailure handles a failed execution
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()
	cb.successes = 0

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.maxFailures {
			cb.transitionTo(StateOpen)
		}
	case StateHalfOpen:
		cb.transitionTo(StateOpen)
	}
}

// onSuccess handles a successful execution
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenSuccess {
			cb.transitionTo(StateClosed)
		}
	}
}

// transitionTo changes the state of the circuit breaker
func (cb *CircuitBreaker) transitionTo(newState State) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters on state change
	if newState == StateClosed {
		cb.failures = 0
		cb.successes = 0
	} else if newState == StateHalfOpen {
		cb.successes = 0
	}

	// Call the callback if set
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Failures returns the current failure count
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}

// CircuitBreakerRegistry manages multiple circuit breakers
type CircuitBreakerRegistry struct {
	mu       sync.RWMutex
	breakers map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerRegistry creates a new registry
func NewCircuitBreakerRegistry(defaultConfig CircuitBreakerConfig) *CircuitBreakerRegistry {
	return &CircuitBreakerRegistry{
		breakers: make(map[string]*CircuitBreaker),
		config:   defaultConfig,
	}
}

// Get returns the circuit breaker for the given name, creating one if needed
func (r *CircuitBreakerRegistry) Get(name string) *CircuitBreaker {
	r.mu.RLock()
	cb, ok := r.breakers[name]
	r.mu.RUnlock()

	if ok {
		return cb
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok = r.breakers[name]; ok {
		return cb
	}

	// Create new circuit breaker
	config := r.config
	config.Name = name
	cb = NewCircuitBreaker(config)
	r.breakers[name] = cb

	return cb
}

// GetAll returns all registered circuit breakers
func (r *CircuitBreakerRegistry) GetAll() map[string]*CircuitBreaker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*CircuitBreaker, len(r.breakers))
	for k, v := range r.breakers {
		result[k] = v
	}
	return result
}

// Stats returns statistics for all circuit breakers
func (r *CircuitBreakerRegistry) Stats() map[string]CircuitBreakerStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]CircuitBreakerStats, len(r.breakers))
	for name, cb := range r.breakers {
		stats[name] = CircuitBreakerStats{
			Name:     name,
			State:    cb.State().String(),
			Failures: cb.Failures(),
		}
	}
	return stats
}

// CircuitBreakerStats holds statistics for a circuit breaker
type CircuitBreakerStats struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Failures int    `json:"failures"`
}

// RetryWithCircuitBreaker retries an operation with circuit breaker protection
func RetryWithCircuitBreaker(
	ctx context.Context,
	cb *CircuitBreaker,
	maxRetries int,
	backoff time.Duration,
	fn func() error,
) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := cb.Execute(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry if circuit is open
		if errors.Is(err, ErrCircuitOpen) {
			return err
		}

		// Wait before retrying
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff * time.Duration(i+1)):
			}
		}
	}

	return lastErr
}
