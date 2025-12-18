package postgres

import (
	"context"
	"fmt"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
)

type NotificationRepository struct {
	db *database.DB
}

func NewNotificationRepository(db *database.DB) repository.NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Save(ctx context.Context, notification *model.Notification) error {
	query := `
		INSERT INTO notifications (
			id, user_id, type, channel, title, message, priority, status,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		notification.ID().String(),
		notification.UserID(),
		string(notification.Type()),
		"in_app", // Default channel for now
		notification.Title(),
		notification.Message(),
		string(notification.Priority()),
		string(notification.Status()),
		notification.CreatedAt(),
	)

	return err
}

func (r *NotificationRepository) Update(ctx context.Context, notification *model.Notification) error {
	query := `
		UPDATE notifications SET
			status = $2, updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		notification.ID().String(),
		string(notification.Status()),
	)

	return err
}

func (r *NotificationRepository) FindByID(ctx context.Context, id model.NotificationID) (*model.Notification, error) {
	// Simplified implementation
	return nil, repository.ErrNotFound
}

func (r *NotificationRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error) {
	return nil, nil
}

func (r *NotificationRepository) FindUnread(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error) {
	return nil, nil
}

func (r *NotificationRepository) FindByStatus(ctx context.Context, status model.NotificationStatus, offset, limit int) ([]*model.Notification, error) {
	return nil, nil
}

func (r *NotificationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int64, error) {
	return 0, nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id model.NotificationID) error {
	query := `DELETE FROM notifications WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
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

func (r *NotificationRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM notifications WHERE expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, query)
	return err
}
