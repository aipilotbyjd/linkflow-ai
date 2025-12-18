package repository

import (
	"context"
	"errors"

	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/model"
)

var (
	ErrNotFound = errors.New("webhook not found")
	ErrDuplicateURL = errors.New("webhook URL already exists")
)

type WebhookRepository interface {
	Save(ctx context.Context, webhook *model.Webhook) error
	Update(ctx context.Context, webhook *model.Webhook) error
	FindByID(ctx context.Context, id model.WebhookID) (*model.Webhook, error)
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Webhook, error)
	FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Webhook, error)
	FindByURL(ctx context.Context, url string) (*model.Webhook, error)
	CountByUserID(ctx context.Context, userID string) (int64, error)
	Delete(ctx context.Context, id model.WebhookID) error
}
