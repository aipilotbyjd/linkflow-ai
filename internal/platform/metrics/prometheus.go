package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestSize      *prometheus.HistogramVec
	HTTPResponseSize     *prometheus.HistogramVec
	HTTPActiveRequests   *prometheus.GaugeVec

	// Workflow metrics
	WorkflowsTotal       *prometheus.GaugeVec
	WorkflowsCreated     *prometheus.CounterVec
	WorkflowsActivated   *prometheus.CounterVec
	WorkflowsDeactivated *prometheus.CounterVec

	// Execution metrics
	ExecutionsTotal      *prometheus.CounterVec
	ExecutionsCompleted  *prometheus.CounterVec
	ExecutionsFailed     *prometheus.CounterVec
	ExecutionDuration    *prometheus.HistogramVec
	ExecutionsInProgress *prometheus.GaugeVec

	// Node execution metrics
	NodeExecutionsTotal    *prometheus.CounterVec
	NodeExecutionDuration  *prometheus.HistogramVec

	// Database metrics
	DBConnectionsOpen    *prometheus.GaugeVec
	DBConnectionsInUse   *prometheus.GaugeVec
	DBQueryDuration      *prometheus.HistogramVec
	DBQueryErrors        *prometheus.CounterVec

	// Cache metrics
	CacheHits            *prometheus.CounterVec
	CacheMisses          *prometheus.CounterVec
	CacheEvictions       *prometheus.CounterVec
	CacheSize            *prometheus.GaugeVec

	// Kafka metrics
	KafkaMessagesProduced *prometheus.CounterVec
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaConsumerLag      *prometheus.GaugeVec

	// Authentication metrics
	AuthAttemptsTotal    *prometheus.CounterVec
	AuthFailures         *prometheus.CounterVec
	ActiveSessions       *prometheus.GaugeVec

	// Business metrics
	UsersTotal           *prometheus.GaugeVec
	OrganizationsTotal   *prometheus.GaugeVec
	APIKeysTotal         *prometheus.GaugeVec

	// System metrics
	SystemCPUUsage       *prometheus.GaugeVec
	SystemMemoryUsage    *prometheus.GaugeVec
	SystemGoroutines     prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"method", "path"},
		),
		HTTPRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_size_bytes",
				Help:      "HTTP request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),
		HTTPActiveRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_active_requests",
				Help:      "Number of active HTTP requests",
			},
			[]string{"method"},
		),

		// Workflow metrics
		WorkflowsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "workflows_total",
				Help:      "Total number of workflows",
			},
			[]string{"status"},
		),
		WorkflowsCreated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_created_total",
				Help:      "Total number of workflows created",
			},
			[]string{"user_id"},
		),
		WorkflowsActivated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_activated_total",
				Help:      "Total number of workflows activated",
			},
			[]string{},
		),
		WorkflowsDeactivated: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "workflows_deactivated_total",
				Help:      "Total number of workflows deactivated",
			},
			[]string{},
		),

		// Execution metrics
		ExecutionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "executions_total",
				Help:      "Total number of executions",
			},
			[]string{"workflow_id", "trigger"},
		),
		ExecutionsCompleted: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "executions_completed_total",
				Help:      "Total number of completed executions",
			},
			[]string{"workflow_id"},
		),
		ExecutionsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "executions_failed_total",
				Help:      "Total number of failed executions",
			},
			[]string{"workflow_id", "error_code"},
		),
		ExecutionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "execution_duration_seconds",
				Help:      "Execution duration in seconds",
				Buckets:   []float64{.1, .5, 1, 2, 5, 10, 30, 60, 120, 300, 600},
			},
			[]string{"workflow_id"},
		),
		ExecutionsInProgress: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "executions_in_progress",
				Help:      "Number of executions currently in progress",
			},
			[]string{"workflow_id"},
		),

		// Node execution metrics
		NodeExecutionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "node_executions_total",
				Help:      "Total number of node executions",
			},
			[]string{"node_type", "status"},
		),
		NodeExecutionDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "node_execution_duration_seconds",
				Help:      "Node execution duration in seconds",
				Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"node_type"},
		),

		// Database metrics
		DBConnectionsOpen: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Number of open database connections",
			},
			[]string{"database"},
		),
		DBConnectionsInUse: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_in_use",
				Help:      "Number of database connections in use",
			},
			[]string{"database"},
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation"},
		),
		DBQueryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "db_query_errors_total",
				Help:      "Total number of database query errors",
			},
			[]string{"operation", "error_type"},
		),

		// Cache metrics
		CacheHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_name"},
		),
		CacheMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_name"},
		),
		CacheEvictions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_evictions_total",
				Help:      "Total number of cache evictions",
			},
			[]string{"cache_name"},
		),
		CacheSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "cache_size_bytes",
				Help:      "Current cache size in bytes",
			},
			[]string{"cache_name"},
		),

		// Kafka metrics
		KafkaMessagesProduced: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_messages_produced_total",
				Help:      "Total number of Kafka messages produced",
			},
			[]string{"topic"},
		),
		KafkaMessagesConsumed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_messages_consumed_total",
				Help:      "Total number of Kafka messages consumed",
			},
			[]string{"topic", "consumer_group"},
		),
		KafkaConsumerLag: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "kafka_consumer_lag",
				Help:      "Kafka consumer lag",
			},
			[]string{"topic", "partition", "consumer_group"},
		),

		// Authentication metrics
		AuthAttemptsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_attempts_total",
				Help:      "Total number of authentication attempts",
			},
			[]string{"method"},
		),
		AuthFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_failures_total",
				Help:      "Total number of authentication failures",
			},
			[]string{"method", "reason"},
		),
		ActiveSessions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_sessions",
				Help:      "Number of active user sessions",
			},
			[]string{},
		),

		// Business metrics
		UsersTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "users_total",
				Help:      "Total number of users",
			},
			[]string{"status"},
		),
		OrganizationsTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "organizations_total",
				Help:      "Total number of organizations",
			},
			[]string{},
		),
		APIKeysTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "api_keys_total",
				Help:      "Total number of API keys",
			},
			[]string{"status"},
		),

		// System metrics
		SystemCPUUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_cpu_usage_percent",
				Help:      "System CPU usage percentage",
			},
			[]string{},
		),
		SystemMemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_memory_usage_percent",
				Help:      "System memory usage percentage",
			},
			[]string{},
		),
		SystemGoroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "system_goroutines",
				Help:      "Number of goroutines",
			},
		),
	}

	// Register all metrics
	m.Register()

	return m
}

// Register registers all metrics with Prometheus
func (m *Metrics) Register() {
	prometheus.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.HTTPActiveRequests,
		m.WorkflowsTotal,
		m.WorkflowsCreated,
		m.WorkflowsActivated,
		m.WorkflowsDeactivated,
		m.ExecutionsTotal,
		m.ExecutionsCompleted,
		m.ExecutionsFailed,
		m.ExecutionDuration,
		m.ExecutionsInProgress,
		m.NodeExecutionsTotal,
		m.NodeExecutionDuration,
		m.DBConnectionsOpen,
		m.DBConnectionsInUse,
		m.DBQueryDuration,
		m.DBQueryErrors,
		m.CacheHits,
		m.CacheMisses,
		m.CacheEvictions,
		m.CacheSize,
		m.KafkaMessagesProduced,
		m.KafkaMessagesConsumed,
		m.KafkaConsumerLag,
		m.AuthAttemptsTotal,
		m.AuthFailures,
		m.ActiveSessions,
		m.UsersTotal,
		m.OrganizationsTotal,
		m.APIKeysTotal,
		m.SystemCPUUsage,
		m.SystemMemoryUsage,
		m.SystemGoroutines,
	)
}

// Handler returns the Prometheus HTTP handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

// HTTPMetricsMiddleware returns middleware that collects HTTP metrics
func (m *Metrics) HTTPMetricsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Increment active requests
			m.HTTPActiveRequests.WithLabelValues(r.Method).Inc()
			defer m.HTTPActiveRequests.WithLabelValues(r.Method).Dec()
			
			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			// Record request size
			if r.ContentLength > 0 {
				m.HTTPRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
			}
			
			// Call next handler
			next.ServeHTTP(wrapped, r)
			
			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(wrapped.statusCode)
			
			m.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
			
			if wrapped.size > 0 {
				m.HTTPResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(wrapped.size))
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}
