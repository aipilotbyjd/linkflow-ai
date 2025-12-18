package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/analytics/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/analytics/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
)

type AnalyticsService struct {
	repository repository.EventRepository
	cache      *cache.RedisCache
	logger     logger.Logger
}

func NewAnalyticsService(
	repository repository.EventRepository,
	cache *cache.RedisCache,
	logger logger.Logger,
) *AnalyticsService {
	return &AnalyticsService{
		repository: repository,
		cache:      cache,
		logger:     logger,
	}
}

type TrackEventCommand struct {
	UserID     string
	SessionID  string
	EventType  string
	EventName  string
	Properties map[string]interface{}
	IP         string
	UserAgent  string
}

func (s *AnalyticsService) TrackEvent(ctx context.Context, cmd TrackEventCommand) error {
	event := model.NewAnalyticsEvent(
		cmd.UserID,
		cmd.SessionID,
		model.EventType(cmd.EventType),
		cmd.EventName,
	)

	for key, value := range cmd.Properties {
		event.SetProperty(key, value)
	}

	event.SetIP(cmd.IP)
	event.SetUserAgent(cmd.UserAgent)

	if err := s.repository.Save(ctx, event); err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	// Update real-time metrics in cache
	go s.updateMetrics(context.Background(), event)

	s.logger.Debug("Event tracked",
		"event_id", event.ID(),
		"user_id", event.UserID(),
		"event_type", event.EventType(),
	)

	return nil
}

func (s *AnalyticsService) updateMetrics(ctx context.Context, event *model.AnalyticsEvent) {
	if s.cache == nil {
		return
	}

	// Increment counters
	today := time.Now().Format("2006-01-02")
	
	// Daily active users
	dauKey := fmt.Sprintf("metrics:dau:%s", today)
	s.cache.SetNX(ctx, fmt.Sprintf("%s:%s", dauKey, event.UserID()), true, 24*time.Hour)
	
	// Event counters
	eventKey := fmt.Sprintf("metrics:events:%s:%s", today, event.EventType())
	s.cache.Increment(ctx, eventKey)
}

type GetMetricsQuery struct {
	UserID    string
	StartDate time.Time
	EndDate   time.Time
	Metrics   []string
}

func (s *AnalyticsService) GetMetrics(ctx context.Context, query GetMetricsQuery) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})
	
	// Get metrics from repository
	events, err := s.repository.FindByDateRange(ctx, query.StartDate, query.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Calculate metrics
	metrics["total_events"] = len(events)
	metrics["date_range"] = map[string]interface{}{
		"start": query.StartDate,
		"end":   query.EndDate,
	}

	// Event type breakdown
	eventTypes := make(map[string]int)
	for _, event := range events {
		eventTypes[string(event.EventType())]++
	}
	metrics["event_types"] = eventTypes

	return metrics, nil
}

type GetUserAnalyticsQuery struct {
	UserID    string
	StartDate time.Time
	EndDate   time.Time
}

func (s *AnalyticsService) GetUserAnalytics(ctx context.Context, query GetUserAnalyticsQuery) (map[string]interface{}, error) {
	events, err := s.repository.FindByUserID(ctx, query.UserID, query.StartDate, query.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get user events: %w", err)
	}

	analytics := map[string]interface{}{
		"user_id":      query.UserID,
		"total_events": len(events),
		"date_range": map[string]interface{}{
			"start": query.StartDate,
			"end":   query.EndDate,
		},
	}

	// Calculate activity by day
	activityByDay := make(map[string]int)
	for _, event := range events {
		day := event.Timestamp().Format("2006-01-02")
		activityByDay[day]++
	}
	analytics["activity_by_day"] = activityByDay

	return analytics, nil
}
