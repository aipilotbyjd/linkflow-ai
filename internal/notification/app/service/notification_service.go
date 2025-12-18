package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/notification/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrUnauthorized         = errors.New("unauthorized")
)

type NotificationService struct {
	repository     repository.NotificationRepository
	eventPublisher *kafka.EventPublisher
	cache          *cache.RedisCache
	logger         logger.Logger
	channels       map[model.NotificationChannel]ChannelSender
}

type ChannelSender interface {
	Send(ctx context.Context, notification *model.Notification) error
}

func NewNotificationService(
	repository repository.NotificationRepository,
	eventPublisher *kafka.EventPublisher,
	cache *cache.RedisCache,
	logger logger.Logger,
) *NotificationService {
	service := &NotificationService{
		repository:     repository,
		eventPublisher: eventPublisher,
		cache:          cache,
		logger:         logger,
		channels:       make(map[model.NotificationChannel]ChannelSender),
	}

	// Register default channel senders
	service.RegisterChannel(model.ChannelInApp, &InAppSender{})
	// Add more channel senders as needed

	return service
}

func (s *NotificationService) RegisterChannel(channel model.NotificationChannel, sender ChannelSender) {
	s.channels[channel] = sender
}

type CreateNotificationCommand struct {
	UserID         string
	OrganizationID string
	Type           string
	Channels       []string
	Title          string
	Message        string
	Data           map[string]interface{}
	Priority       string
	ActionURL      string
	ActionLabel    string
	TemplateID     string
	TemplateData   map[string]interface{}
	ExpiresAt      *time.Time
}

func (s *NotificationService) CreateNotification(ctx context.Context, cmd CreateNotificationCommand) (*model.Notification, error) {
	// Convert channels
	channels := make([]model.NotificationChannel, len(cmd.Channels))
	for i, ch := range cmd.Channels {
		channels[i] = model.NotificationChannel(ch)
	}

	// Create notification
	notification, err := model.NewNotification(
		cmd.UserID,
		model.NotificationType(cmd.Type),
		cmd.Title,
		cmd.Message,
		channels,
	)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if cmd.OrganizationID != "" {
		notification.SetOrganizationID(cmd.OrganizationID)
	}
	if cmd.Priority != "" {
		notification.SetPriority(model.NotificationPriority(cmd.Priority))
	}
	if cmd.ActionURL != "" {
		notification.SetAction(cmd.ActionURL, cmd.ActionLabel)
	}
	if cmd.TemplateID != "" {
		notification.SetTemplate(cmd.TemplateID, cmd.TemplateData)
	}
	if cmd.ExpiresAt != nil {
		notification.SetExpiration(*cmd.ExpiresAt)
	}

	// Save notification
	if err := s.repository.Save(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to save notification: %w", err)
	}

	// Send notification asynchronously
	go s.sendNotification(context.Background(), notification)

	// Publish event
	if s.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"notificationId": notification.ID().String(),
			"userId":         notification.UserID(),
			"type":           string(notification.Type()),
			"channels":       channels,
		})
		event := &events.Event{
			AggregateID:   notification.ID().String(),
			AggregateType: "Notification",
			Type:          events.NotificationSent, // created",
			UserID:        cmd.UserID,
			Timestamp:     time.Now(),
			Data:          json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Notification created", 
		"notification_id", notification.ID(),
		"user_id", notification.UserID(),
		"channels", channels,
	)

	return notification, nil
}

func (s *NotificationService) sendNotification(ctx context.Context, notification *model.Notification) {
	for _, channel := range notification.Channels() {
		sender, exists := s.channels[channel]
		if !exists {
			s.logger.Warn("No sender for channel", "channel", channel)
			_ = notification.MarkChannelFailed(channel, "channel not configured")
			continue
		}

		// Send through channel
		if err := sender.Send(ctx, notification); err != nil {
			s.logger.Error("Failed to send notification",
				"notification_id", notification.ID(),
				"channel", channel,
				"error", err,
			)
			_ = notification.MarkChannelFailed(channel, err.Error())

			// Retry if applicable
			if notification.ShouldRetry(channel) {
				go s.retryNotification(context.Background(), notification, channel)
			}
		} else {
			_ = notification.MarkChannelSent(channel)
			s.logger.Info("Notification sent",
				"notification_id", notification.ID(),
				"channel", channel,
			)
		}
	}

	// Update notification status
	_ = s.repository.Update(ctx, notification)
}

func (s *NotificationService) retryNotification(ctx context.Context, notification *model.Notification, channel model.NotificationChannel) {
	// Wait before retry
	time.Sleep(5 * time.Second)

	sender, exists := s.channels[channel]
	if !exists {
		return
	}

	if err := sender.Send(ctx, notification); err != nil {
		_ = notification.MarkChannelFailed(channel, err.Error())
	} else {
		_ = notification.MarkChannelSent(channel)
	}

	_ = s.repository.Update(ctx, notification)
}

func (s *NotificationService) GetNotification(ctx context.Context, notificationID model.NotificationID, userID string) (*model.Notification, error) {
	notification, err := s.repository.FindByID(ctx, notificationID)
	if err != nil {
		return nil, ErrNotificationNotFound
	}

	// Check authorization
	if notification.UserID() != userID {
		return nil, ErrUnauthorized
	}

	return notification, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID model.NotificationID, userID string) error {
	notification, err := s.repository.FindByID(ctx, notificationID)
	if err != nil {
		return ErrNotificationNotFound
	}

	// Check authorization
	if notification.UserID() != userID {
		return ErrUnauthorized
	}

	if err := notification.MarkAsRead(); err != nil {
		return err
	}

	return s.repository.Update(ctx, notification)
}

type ListNotificationsQuery struct {
	UserID   string
	Status   string
	Type     string
	Channel  string
	Priority string
	Offset   int
	Limit    int
}

func (s *NotificationService) ListNotifications(ctx context.Context, query ListNotificationsQuery) ([]*model.Notification, int64, error) {
	notifications, err := s.repository.FindByUserID(ctx, query.UserID, query.Offset, query.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}

	total, err := s.repository.CountByUserID(ctx, query.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	return notifications, total, nil
}

// InAppSender sends in-app notifications
type InAppSender struct{}

func (s *InAppSender) Send(ctx context.Context, notification *model.Notification) error {
	// In-app notifications are already saved to database
	// Just mark as delivered
	return notification.MarkChannelDelivered(model.ChannelInApp)
}
