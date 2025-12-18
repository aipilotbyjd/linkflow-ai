// Package health provides health check functionality for services
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Check represents a single health check
type Check struct {
	Name    string        `json:"name"`
	Status  Status        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency_ms"`
}

// Response is the health check response
type Response struct {
	Status    Status            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Service   string            `json:"service,omitempty"`
	Checks    map[string]*Check `json:"checks,omitempty"`
	Uptime    time.Duration     `json:"uptime_seconds,omitempty"`
}

// Checker is a function that performs a health check
type Checker func(ctx context.Context) error

// Handler manages health checks for a service
type Handler struct {
	mu        sync.RWMutex
	checks    map[string]Checker
	service   string
	version   string
	startTime time.Time
}

// NewHandler creates a new health handler
func NewHandler(service, version string) *Handler {
	return &Handler{
		checks:    make(map[string]Checker),
		service:   service,
		version:   version,
		startTime: time.Now(),
	}
}

// AddCheck registers a health check
func (h *Handler) AddCheck(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = checker
}

// RemoveCheck removes a health check
func (h *Handler) RemoveCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
}

// Check runs all health checks and returns the result
func (h *Handler) Check(ctx context.Context) *Response {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp := &Response{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   h.version,
		Service:   h.service,
		Checks:    make(map[string]*Check),
		Uptime:    time.Since(h.startTime),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, checker := range h.checks {
		wg.Add(1)
		go func(name string, checker Checker) {
			defer wg.Done()

			start := time.Now()
			err := checker(ctx)
			latency := time.Since(start)

			check := &Check{
				Name:    name,
				Latency: latency / time.Millisecond,
			}

			if err != nil {
				check.Status = StatusUnhealthy
				check.Message = err.Error()
			} else {
				check.Status = StatusHealthy
			}

			mu.Lock()
			resp.Checks[name] = check
			if check.Status == StatusUnhealthy {
				resp.Status = StatusUnhealthy
			}
			mu.Unlock()
		}(name, checker)
	}

	wg.Wait()
	return resp
}

// LivenessHandler returns an HTTP handler for liveness probe
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}

// ReadinessHandler returns an HTTP handler for readiness probe
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		resp := h.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if resp.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// HealthHandler returns an HTTP handler for detailed health check
func (h *Handler) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := h.Check(ctx)

		w.Header().Set("Content-Type", "application/json")
		if resp.Status == StatusHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// Common health checkers

// DatabaseChecker creates a database health checker
func DatabaseChecker(pingFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// RedisChecker creates a Redis health checker
func RedisChecker(pingFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// KafkaChecker creates a Kafka health checker
func KafkaChecker(pingFunc func(ctx context.Context) error) Checker {
	return func(ctx context.Context) error {
		return pingFunc(ctx)
	}
}

// HTTPChecker creates an HTTP endpoint health checker
func HTTPChecker(url string, timeout time.Duration) Checker {
	return func(ctx context.Context) error {
		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return &HealthError{
				Message: "unhealthy status code",
				Code:    resp.StatusCode,
			}
		}
		return nil
	}
}

// HealthError represents a health check error
type HealthError struct {
	Message string
	Code    int
}

func (e *HealthError) Error() string {
	return e.Message
}
