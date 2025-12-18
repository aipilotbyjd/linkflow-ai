package model

import (
	"time"

	"github.com/google/uuid"
)

type MetricID string

func NewMetricID() MetricID {
	return MetricID(uuid.New().String())
}

type MetricType string

const (
	MetricTypeCPU       MetricType = "cpu"
	MetricTypeMemory    MetricType = "memory"
	MetricTypeDisk      MetricType = "disk"
	MetricTypeNetwork   MetricType = "network"
	MetricTypeLatency   MetricType = "latency"
	MetricTypeThroughput MetricType = "throughput"
	MetricTypeError     MetricType = "error"
	MetricTypeCustom    MetricType = "custom"
)

type Metric struct {
	id         MetricID
	service    string
	name       string
	metricType MetricType
	value      float64
	unit       string
	tags       map[string]string
	timestamp  time.Time
}

func NewMetric(service, name string, metricType MetricType, value float64) *Metric {
	return &Metric{
		id:         NewMetricID(),
		service:    service,
		name:       name,
		metricType: metricType,
		value:      value,
		tags:       make(map[string]string),
		timestamp:  time.Now(),
	}
}

func (m *Metric) ID() MetricID         { return m.id }
func (m *Metric) Service() string      { return m.service }
func (m *Metric) Name() string         { return m.name }
func (m *Metric) Type() MetricType     { return m.metricType }
func (m *Metric) Value() float64       { return m.value }
func (m *Metric) Timestamp() time.Time { return m.timestamp }

type HealthCheck struct {
	id            string
	service       string
	endpoint      string
	status        HealthStatus
	responseTime  time.Duration
	errorMessage  string
	lastCheckTime time.Time
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

func NewHealthCheck(service, endpoint string) *HealthCheck {
	return &HealthCheck{
		id:            uuid.New().String(),
		service:       service,
		endpoint:      endpoint,
		status:        HealthStatusHealthy,
		lastCheckTime: time.Now(),
	}
}

type Alert struct {
	id          string
	metric      string
	condition   string
	threshold   float64
	severity    AlertSeverity
	message     string
	triggered   bool
	triggeredAt *time.Time
	resolvedAt  *time.Time
}

type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

func NewAlert(metric, condition string, threshold float64, severity AlertSeverity) *Alert {
	return &Alert{
		id:        uuid.New().String(),
		metric:    metric,
		condition: condition,
		threshold: threshold,
		severity:  severity,
		triggered: false,
	}
}
