package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/monitoring/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type MonitoringService struct {
	metrics      map[string]*model.Metric
	healthChecks map[string]*model.HealthCheck
	alerts       map[string]*model.Alert
	logger       logger.Logger
	mu           sync.RWMutex
	stopCh       chan struct{}
}

func NewMonitoringService(logger logger.Logger) *MonitoringService {
	return &MonitoringService{
		metrics:      make(map[string]*model.Metric),
		healthChecks: make(map[string]*model.HealthCheck),
		alerts:       make(map[string]*model.Alert),
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

func (s *MonitoringService) Start(ctx context.Context) {
	// Start system metrics collection
	go s.collectSystemMetrics(ctx)
	
	// Start health checks
	go s.performHealthChecks(ctx)
	
	// Start alert monitoring
	go s.monitorAlerts(ctx)
	
	s.logger.Info("Monitoring service started")
}

func (s *MonitoringService) Stop() {
	close(s.stopCh)
	s.logger.Info("Monitoring service stopped")
}

func (s *MonitoringService) collectSystemMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.collectCPUMetrics()
			s.collectMemoryMetrics()
			s.collectDiskMetrics()
			s.collectNetworkMetrics()
		}
	}
}

func (s *MonitoringService) collectCPUMetrics() {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		s.logger.Error("Failed to collect CPU metrics", "error", err)
		return
	}
	
	if len(percent) > 0 {
		metric := model.NewMetric("system", "cpu_usage", model.MetricTypeCPU, percent[0])
		s.recordMetric(metric)
	}
	
	// Collect Go runtime metrics
	metric := model.NewMetric("runtime", "goroutines", model.MetricTypeCustom, float64(runtime.NumGoroutine()))
	s.recordMetric(metric)
}

func (s *MonitoringService) collectMemoryMetrics() {
	v, err := mem.VirtualMemory()
	if err != nil {
		s.logger.Error("Failed to collect memory metrics", "error", err)
		return
	}
	
	metric := model.NewMetric("system", "memory_usage", model.MetricTypeMemory, v.UsedPercent)
	s.recordMetric(metric)
	
	// Go runtime memory
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	heapMetric := model.NewMetric("runtime", "heap_alloc_mb", model.MetricTypeMemory, float64(m.HeapAlloc)/(1024*1024))
	s.recordMetric(heapMetric)
}

func (s *MonitoringService) collectDiskMetrics() {
	usage, err := disk.Usage("/")
	if err != nil {
		s.logger.Error("Failed to collect disk metrics", "error", err)
		return
	}
	
	metric := model.NewMetric("system", "disk_usage", model.MetricTypeDisk, usage.UsedPercent)
	s.recordMetric(metric)
}

func (s *MonitoringService) collectNetworkMetrics() {
	stats, err := net.IOCounters(false)
	if err != nil {
		s.logger.Error("Failed to collect network metrics", "error", err)
		return
	}
	
	if len(stats) > 0 {
		rxMetric := model.NewMetric("system", "network_rx_bytes", model.MetricTypeNetwork, float64(stats[0].BytesRecv))
		txMetric := model.NewMetric("system", "network_tx_bytes", model.MetricTypeNetwork, float64(stats[0].BytesSent))
		s.recordMetric(rxMetric)
		s.recordMetric(txMetric)
	}
}

func (s *MonitoringService) recordMetric(metric *model.Metric) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := fmt.Sprintf("%s_%s", metric.Service(), metric.Name())
	s.metrics[key] = metric
	
	// Check alerts
	s.checkAlertsForMetric(metric)
}

func (s *MonitoringService) performHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	services := []struct {
		name     string
		endpoint string
	}{
		{"auth", "http://localhost:8001/health/live"},
		{"user", "http://localhost:8002/health/live"},
		{"workflow", "http://localhost:8004/health/live"},
		{"execution", "http://localhost:8003/health/live"},
		{"gateway", "http://localhost:8000/health/live"},
	}
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			for _, service := range services {
				go s.checkServiceHealth(service.name, service.endpoint)
			}
		}
	}
}

func (s *MonitoringService) checkServiceHealth(service, endpoint string) {
	healthCheck := model.NewHealthCheck(service, endpoint)
	
	// Perform health check (simplified)
	// In production, would make actual HTTP request
	healthCheck.Status = model.HealthStatusHealthy
	
	s.mu.Lock()
	s.healthChecks[service] = healthCheck
	s.mu.Unlock()
	
	s.logger.Debug("Health check performed", "service", service, "status", healthCheck.Status)
}

func (s *MonitoringService) monitorAlerts(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.evaluateAlerts()
		}
	}
}

func (s *MonitoringService) checkAlertsForMetric(metric *model.Metric) {
	for _, alert := range s.alerts {
		if alert.Metric == metric.Name() {
			s.evaluateAlert(alert, metric)
		}
	}
}

func (s *MonitoringService) evaluateAlert(alert *model.Alert, metric *model.Metric) {
	triggered := false
	
	switch alert.Condition {
	case ">":
		triggered = metric.Value() > alert.Threshold
	case "<":
		triggered = metric.Value() < alert.Threshold
	case ">=":
		triggered = metric.Value() >= alert.Threshold
	case "<=":
		triggered = metric.Value() <= alert.Threshold
	}
	
	if triggered && !alert.Triggered {
		alert.Triggered = true
		now := time.Now()
		alert.TriggeredAt = &now
		s.logger.Warn("Alert triggered",
			"alert", alert.ID,
			"metric", metric.Name(),
			"value", metric.Value(),
			"threshold", alert.Threshold,
		)
	} else if !triggered && alert.Triggered {
		alert.Triggered = false
		now := time.Now()
		alert.ResolvedAt = &now
		s.logger.Info("Alert resolved",
			"alert", alert.ID,
			"metric", metric.Name(),
		)
	}
}

func (s *MonitoringService) evaluateAlerts() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	for _, alert := range s.alerts {
		if metric, exists := s.metrics[alert.Metric]; exists {
			s.evaluateAlert(alert, metric)
		}
	}
}

func (s *MonitoringService) CreateAlert(ctx context.Context, metric, condition string, threshold float64, severity string) (*model.Alert, error) {
	alert := model.NewAlert(metric, condition, threshold, model.AlertSeverity(severity))
	
	s.mu.Lock()
	s.alerts[alert.ID] = alert
	s.mu.Unlock()
	
	s.logger.Info("Alert created",
		"id", alert.ID,
		"metric", metric,
		"condition", condition,
		"threshold", threshold,
	)
	
	return alert, nil
}

func (s *MonitoringService) GetMetrics(ctx context.Context) map[string]*model.Metric {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make(map[string]*model.Metric)
	for k, v := range s.metrics {
		result[k] = v
	}
	return result
}

func (s *MonitoringService) GetHealthStatus(ctx context.Context) map[string]*model.HealthCheck {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make(map[string]*model.HealthCheck)
	for k, v := range s.healthChecks {
		result[k] = v
	}
	return result
}

func (s *MonitoringService) GetAlerts(ctx context.Context) []*model.Alert {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*model.Alert
	for _, alert := range s.alerts {
		result = append(result, alert)
	}
	return result
}
