package postgres

import (
	"context"
	"fmt"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/webhook/domain/repository"
)

type WebhookRepository struct {
	db *database.DB
}

func NewWebhookRepository(db *database.DB) repository.WebhookRepository {
	return &WebhookRepository{db: db}
}

func (r *WebhookRepository) Save(ctx context.Context, webhook *model.Webhook) error {
	query := `
		INSERT INTO webhooks (
			id, user_id, workflow_id, name, endpoint_url, secret, method, status,
			trigger_count, success_count, failure_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		webhook.ID().String(),
		webhook.UserID(),
		webhook.WorkflowID(),
		webhook.Name(),
		webhook.EndpointURL(),
		webhook.Secret(),
		string(webhook.Method()),
		string(webhook.Status()),
		webhook.TriggerCount(),
		webhook.SuccessCount(),
		webhook.FailureCount(),
		webhook.CreatedAt(),
		webhook.UpdatedAt(),
	)

	return err
}

func (r *WebhookRepository) Update(ctx context.Context, webhook *model.Webhook) error {
	query := `
		UPDATE webhooks SET
			endpoint_url = $2, status = $3, trigger_count = $4,
			success_count = $5, failure_count = $6, updated_at = $7
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		webhook.ID().String(),
		webhook.EndpointURL(),
		string(webhook.Status()),
		webhook.TriggerCount(),
		webhook.SuccessCount(),
		webhook.FailureCount(),
		webhook.UpdatedAt(),
	)

	return err
}

func (r *WebhookRepository) FindByID(ctx context.Context, id model.WebhookID) (*model.Webhook, error) {
	// Simplified implementation - would need full implementation
	return nil, repository.ErrNotFound
}

func (r *WebhookRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Webhook, error) {
	return nil, nil
}

func (r *WebhookRepository) FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Webhook, error) {
	return nil, nil
}

func (r *WebhookRepository) FindByURL(ctx context.Context, url string) (*model.Webhook, error) {
	return nil, repository.ErrNotFound
}

func (r *WebhookRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (r *WebhookRepository) Delete(ctx context.Context, id model.WebhookID) error {
	query := `DELETE FROM webhooks WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}
