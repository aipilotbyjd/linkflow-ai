package repository

import (
	"context"
	"errors"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
)

var (
	ErrNotFound = errors.New("notification not found")
)

type NotificationRepository interface {
	Save(ctx context.Context, notification *model.Notification) error
	Update(ctx context.Context, notification *model.Notification) error
	FindByID(ctx context.Context, id model.NotificationID) (*model.Notification, error)
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error)
	FindUnread(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error)
	FindByStatus(ctx context.Context, status model.NotificationStatus, offset, limit int) ([]*model.Notification, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	CountUnread(ctx context.Context, userID string) (int64, error)
	Delete(ctx context.Context, id model.NotificationID) error
	DeleteExpired(ctx context.Context) error
}
