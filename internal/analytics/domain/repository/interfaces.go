package repository

import (
	"context"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/analytics/domain/model"
)

type EventRepository interface {
	Save(ctx context.Context, event *model.AnalyticsEvent) error
	FindByUserID(ctx context.Context, userID string, start, end time.Time) ([]*model.AnalyticsEvent, error)
	FindByDateRange(ctx context.Context, start, end time.Time) ([]*model.AnalyticsEvent, error)
	GetAggregates(ctx context.Context, start, end time.Time, groupBy string) (map[string]interface{}, error)
}
