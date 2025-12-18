package telemetry

import (
	"context"
	"fmt"
	"net/http"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

// Telemetry holds telemetry components
type Telemetry struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	metrics  *prometheus.Registry
}

// Config for telemetry
type Config struct {
	ServiceName    string
	JaegerEndpoint string
	MetricsEnabled bool
	TracingEnabled bool
}

// New creates new telemetry instance
func New(cfg Config) (*Telemetry, error) {
	t := &Telemetry{
		metrics: prometheus.NewRegistry(),
	}
	
	// Setup tracing if enabled
	if cfg.TracingEnabled {
		provider, err := initTracer(cfg.ServiceName, cfg.JaegerEndpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize tracer: %w", err)
		}
		t.provider = provider
		t.tracer = otel.Tracer(cfg.ServiceName)
	}
	
	// Register default metrics
	if cfg.MetricsEnabled {
		prometheus.DefaultRegisterer = t.metrics
		t.metrics.MustRegister(prometheus.NewGoCollector())
		t.metrics.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	}
	
	return t, nil
}

// initTracer initializes Jaeger tracer
func initTracer(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(endpoint),
		),
	)
	if err != nil {
		return nil, err
	}
	
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	
	otel.SetTracerProvider(tp)
	
	return tp, nil
}

// Tracer returns the tracer
func (t *Telemetry) Tracer() trace.Tracer {
	return t.tracer
}

// MetricsHandler returns HTTP handler for metrics
func (t *Telemetry) MetricsHandler() http.Handler {
	return promhttp.HandlerFor(t.metrics, promhttp.HandlerOpts{})
}

// Close shuts down telemetry
func (t *Telemetry) Close() error {
	if t.provider != nil {
		return t.provider.Shutdown(context.Background())
	}
	return nil
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for tracing
func (t *Telemetry) UnaryServerInterceptor() interface{} {
	// Placeholder for actual implementation
	return nil
}

// StreamServerInterceptor returns a gRPC stream server interceptor for tracing  
func (t *Telemetry) StreamServerInterceptor() interface{} {
	// Placeholder for actual implementation
	return nil
}
