package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

// notificationRow represents a database row
type notificationRow struct {
	ID             string
	UserID         string
	OrganizationID sql.NullString
	Type           string
	Channels       []byte
	Title          string
	Message        string
	Data           []byte
	Priority       string
	Status         string
	DeliveryStatus []byte
	ReadAt         sql.NullTime
	ActionURL      sql.NullString
	ActionLabel    sql.NullString
	ExpiresAt      sql.NullTime
	Metadata       []byte
	TemplateID     sql.NullString
	TemplateData   []byte
	GroupID        sql.NullString
	RetryCount     int
	MaxRetries     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Version        int
}

func (r *NotificationRepository) Save(ctx context.Context, notification *model.Notification) error {
	channelsJSON, _ := json.Marshal(notification.Channels())
	dataJSON, _ := json.Marshal(notification.Data())
	metadataJSON, _ := json.Marshal(map[string]interface{}{})
	deliveryStatusJSON, _ := json.Marshal(map[string]interface{}{})
	templateDataJSON, _ := json.Marshal(map[string]interface{}{})

	query := `
		INSERT INTO notifications (
			id, user_id, type, channels, title, message, data, priority, status,
			delivery_status, metadata, template_data, retry_count, max_retries,
			created_at, updated_at, version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`

	_, err := r.db.ExecContext(ctx, query,
		notification.ID().String(),
		notification.UserID(),
		string(notification.Type()),
		channelsJSON,
		notification.Title(),
		notification.Message(),
		dataJSON,
		string(notification.Priority()),
		string(notification.Status()),
		deliveryStatusJSON,
		metadataJSON,
		templateDataJSON,
		0,
		3,
		notification.CreatedAt(),
		notification.UpdatedAt(),
		notification.Version(),
	)

	return err
}

func (r *NotificationRepository) Update(ctx context.Context, notification *model.Notification) error {
	query := `
		UPDATE notifications SET
			status = $2, 
			priority = $3,
			updated_at = NOW(),
			version = version + 1
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		notification.ID().String(),
		string(notification.Status()),
		string(notification.Priority()),
	)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
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

func (r *NotificationRepository) FindByID(ctx context.Context, id model.NotificationID) (*model.Notification, error) {
	query := `
		SELECT id, user_id, type, channels, title, message, data, priority, status,
			delivery_status, read_at, action_url, action_label, expires_at, metadata,
			template_id, template_data, group_id, retry_count, max_retries,
			created_at, updated_at, version
		FROM notifications
		WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, id.String())
	return r.scanNotification(row)
}

func (r *NotificationRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error) {
	query := `
		SELECT id, user_id, type, channels, title, message, data, priority, status,
			delivery_status, read_at, action_url, action_label, expires_at, metadata,
			template_id, template_data, group_id, retry_count, max_retries,
			created_at, updated_at, version
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

func (r *NotificationRepository) FindUnread(ctx context.Context, userID string, offset, limit int) ([]*model.Notification, error) {
	query := `
		SELECT id, user_id, type, channels, title, message, data, priority, status,
			delivery_status, read_at, action_url, action_label, expires_at, metadata,
			template_id, template_data, group_id, retry_count, max_retries,
			created_at, updated_at, version
		FROM notifications
		WHERE user_id = $1 AND status != 'read'
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query unread notifications: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

func (r *NotificationRepository) FindByStatus(ctx context.Context, status model.NotificationStatus, offset, limit int) ([]*model.Notification, error) {
	query := `
		SELECT id, user_id, type, channels, title, message, data, priority, status,
			delivery_status, read_at, action_url, action_label, expires_at, metadata,
			template_id, template_data, group_id, retry_count, max_retries,
			created_at, updated_at, version
		FROM notifications
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, string(status), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications by status: %w", err)
	}
	defer rows.Close()

	return r.scanNotifications(rows)
}

func (r *NotificationRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	return count, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND status != 'read'`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	return count, nil
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
	query := `DELETE FROM notifications WHERE expires_at IS NOT NULL AND expires_at < NOW()`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// scanNotification scans a single row into a Notification
func (r *NotificationRepository) scanNotification(row *sql.Row) (*model.Notification, error) {
	var nr notificationRow
	var channelsJSON, dataJSON, deliveryStatusJSON, metadataJSON, templateDataJSON []byte

	err := row.Scan(
		&nr.ID, &nr.UserID, &nr.Type, &channelsJSON, &nr.Title, &nr.Message,
		&dataJSON, &nr.Priority, &nr.Status, &deliveryStatusJSON, &nr.ReadAt,
		&nr.ActionURL, &nr.ActionLabel, &nr.ExpiresAt, &metadataJSON,
		&nr.TemplateID, &templateDataJSON, &nr.GroupID, &nr.RetryCount,
		&nr.MaxRetries, &nr.CreatedAt, &nr.UpdatedAt, &nr.Version,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan notification: %w", err)
	}

	// Parse JSON fields
	var channels []model.NotificationChannel
	if len(channelsJSON) > 0 {
		json.Unmarshal(channelsJSON, &channels)
	}
	if len(channels) == 0 {
		channels = []model.NotificationChannel{model.ChannelInApp}
	}

	notification, err := model.NewNotification(
		nr.UserID,
		model.NotificationType(nr.Type),
		nr.Title,
		nr.Message,
		channels,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	notification.SetPriority(model.NotificationPriority(nr.Priority))

	if nr.ActionURL.Valid && nr.ActionLabel.Valid {
		notification.SetAction(nr.ActionURL.String, nr.ActionLabel.String)
	}

	if nr.ExpiresAt.Valid {
		notification.SetExpiration(nr.ExpiresAt.Time)
	}

	if nr.GroupID.Valid {
		notification.SetGroup(nr.GroupID.String)
	}

	// Mark as read if status is read
	if nr.Status == string(model.StatusRead) {
		notification.MarkAsRead()
	}

	return notification, nil
}

// scanNotifications scans multiple rows into Notifications
func (r *NotificationRepository) scanNotifications(rows *sql.Rows) ([]*model.Notification, error) {
	var notifications []*model.Notification

	for rows.Next() {
		var nr notificationRow
		var channelsJSON, dataJSON, deliveryStatusJSON, metadataJSON, templateDataJSON []byte

		err := rows.Scan(
			&nr.ID, &nr.UserID, &nr.Type, &channelsJSON, &nr.Title, &nr.Message,
			&dataJSON, &nr.Priority, &nr.Status, &deliveryStatusJSON, &nr.ReadAt,
			&nr.ActionURL, &nr.ActionLabel, &nr.ExpiresAt, &metadataJSON,
			&nr.TemplateID, &templateDataJSON, &nr.GroupID, &nr.RetryCount,
			&nr.MaxRetries, &nr.CreatedAt, &nr.UpdatedAt, &nr.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification row: %w", err)
		}

		var channels []model.NotificationChannel
		if len(channelsJSON) > 0 {
			json.Unmarshal(channelsJSON, &channels)
		}
		if len(channels) == 0 {
			channels = []model.NotificationChannel{model.ChannelInApp}
		}

		notification, err := model.NewNotification(
			nr.UserID,
			model.NotificationType(nr.Type),
			nr.Title,
			nr.Message,
			channels,
		)
		if err != nil {
			continue
		}

		notification.SetPriority(model.NotificationPriority(nr.Priority))

		if nr.ActionURL.Valid && nr.ActionLabel.Valid {
			notification.SetAction(nr.ActionURL.String, nr.ActionLabel.String)
		}

		if nr.ExpiresAt.Valid {
			notification.SetExpiration(nr.ExpiresAt.Time)
		}

		if nr.GroupID.Valid {
			notification.SetGroup(nr.GroupID.String)
		}

		if nr.Status == string(model.StatusRead) {
			notification.MarkAsRead()
		}

		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notification rows: %w", err)
	}

	return notifications, nil
}
