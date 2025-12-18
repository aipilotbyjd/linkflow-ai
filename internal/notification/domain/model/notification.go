package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type NotificationID string

func NewNotificationID() NotificationID {
	return NotificationID(uuid.New().String())
}

func (id NotificationID) String() string {
	return string(id)
}

type NotificationType string

const (
	NotificationTypeInfo     NotificationType = "info"
	NotificationTypeSuccess  NotificationType = "success"
	NotificationTypeWarning  NotificationType = "warning"
	NotificationTypeError    NotificationType = "error"
	NotificationTypeWorkflow NotificationType = "workflow"
	NotificationTypeSystem   NotificationType = "system"
	NotificationTypeAlert    NotificationType = "alert"
)

type NotificationChannel string

const (
	ChannelInApp    NotificationChannel = "in_app"
	ChannelEmail    NotificationChannel = "email"
	ChannelSMS      NotificationChannel = "sms"
	ChannelSlack    NotificationChannel = "slack"
	ChannelWebhook  NotificationChannel = "webhook"
	ChannelPush     NotificationChannel = "push"
	ChannelDiscord  NotificationChannel = "discord"
	ChannelTeams    NotificationChannel = "teams"
)

type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "low"
	PriorityNormal NotificationPriority = "normal"
	PriorityHigh   NotificationPriority = "high"
	PriorityUrgent NotificationPriority = "urgent"
)

type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusSent      NotificationStatus = "sent"
	StatusDelivered NotificationStatus = "delivered"
	StatusRead      NotificationStatus = "read"
	StatusFailed    NotificationStatus = "failed"
	StatusExpired   NotificationStatus = "expired"
)

type DeliveryStatus struct {
	Channel     NotificationChannel `json:"channel"`
	Status      NotificationStatus  `json:"status"`
	SentAt      *time.Time          `json:"sentAt,omitempty"`
	DeliveredAt *time.Time          `json:"deliveredAt,omitempty"`
	Error       string              `json:"error,omitempty"`
	RetryCount  int                 `json:"retryCount"`
}

type Notification struct {
	id              NotificationID
	userID          string
	organizationID  string
	notifType       NotificationType
	channels        []NotificationChannel
	title           string
	message         string
	data            map[string]interface{}
	priority        NotificationPriority
	status          NotificationStatus
	deliveryStatus  map[NotificationChannel]*DeliveryStatus
	readAt          *time.Time
	actionURL       string
	actionLabel     string
	expiresAt       *time.Time
	metadata        map[string]interface{}
	templateID      string
	templateData    map[string]interface{}
	groupID         string
	retryCount      int
	maxRetries      int
	createdAt       time.Time
	updatedAt       time.Time
	version         int
}

func NewNotification(
	userID string,
	notifType NotificationType,
	title string,
	message string,
	channels []NotificationChannel,
) (*Notification, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if title == "" {
		return nil, errors.New("title is required")
	}
	if len(channels) == 0 {
		channels = []NotificationChannel{ChannelInApp}
	}

	now := time.Now()
	notification := &Notification{
		id:             NewNotificationID(),
		userID:         userID,
		notifType:      notifType,
		channels:       channels,
		title:          title,
		message:        message,
		data:           make(map[string]interface{}),
		priority:       PriorityNormal,
		status:         StatusPending,
		deliveryStatus: make(map[NotificationChannel]*DeliveryStatus),
		metadata:       make(map[string]interface{}),
		templateData:   make(map[string]interface{}),
		maxRetries:     3,
		createdAt:      now,
		updatedAt:      now,
		version:        0,
	}

	// Initialize delivery status for each channel
	for _, channel := range channels {
		notification.deliveryStatus[channel] = &DeliveryStatus{
			Channel:    channel,
			Status:     StatusPending,
			RetryCount: 0,
		}
	}

	return notification, nil
}

// Getters
func (n *Notification) ID() NotificationID                   { return n.id }
func (n *Notification) UserID() string                       { return n.userID }
func (n *Notification) Type() NotificationType               { return n.notifType }
func (n *Notification) Title() string                        { return n.title }
func (n *Notification) Message() string                      { return n.message }
func (n *Notification) Channels() []NotificationChannel      { return n.channels }
func (n *Notification) Priority() NotificationPriority       { return n.priority }
func (n *Notification) Status() NotificationStatus           { return n.status }
func (n *Notification) Data() map[string]interface{}         { return n.data }
func (n *Notification) CreatedAt() time.Time                 { return n.createdAt }
func (n *Notification) UpdatedAt() time.Time                 { return n.updatedAt }
func (n *Notification) Version() int                         { return n.version }

func (n *Notification) SetOrganizationID(orgID string) {
	n.organizationID = orgID
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) SetPriority(priority NotificationPriority) {
	n.priority = priority
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) SetAction(url, label string) {
	n.actionURL = url
	n.actionLabel = label
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) SetExpiration(expiresAt time.Time) {
	n.expiresAt = &expiresAt
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) SetTemplate(templateID string, data map[string]interface{}) {
	n.templateID = templateID
	n.templateData = data
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) SetGroup(groupID string) {
	n.groupID = groupID
	n.updatedAt = time.Now()
	n.version++
}

func (n *Notification) MarkAsRead() error {
	if n.status == StatusRead {
		return errors.New("notification already read")
	}
	
	now := time.Now()
	n.readAt = &now
	n.status = StatusRead
	n.updatedAt = now
	n.version++
	return nil
}

func (n *Notification) MarkChannelSent(channel NotificationChannel) error {
	delivery, exists := n.deliveryStatus[channel]
	if !exists {
		return errors.New("channel not found")
	}
	
	now := time.Now()
	delivery.Status = StatusSent
	delivery.SentAt = &now
	
	// Update overall status if all channels are sent
	allSent := true
	for _, d := range n.deliveryStatus {
		if d.Status == StatusPending {
			allSent = false
			break
		}
	}
	if allSent {
		n.status = StatusSent
	}
	
	n.updatedAt = now
	n.version++
	return nil
}

func (n *Notification) MarkChannelDelivered(channel NotificationChannel) error {
	delivery, exists := n.deliveryStatus[channel]
	if !exists {
		return errors.New("channel not found")
	}
	
	now := time.Now()
	delivery.Status = StatusDelivered
	delivery.DeliveredAt = &now
	
	// Update overall status if all channels are delivered
	allDelivered := true
	for _, d := range n.deliveryStatus {
		if d.Status != StatusDelivered && d.Status != StatusFailed {
			allDelivered = false
			break
		}
	}
	if allDelivered {
		n.status = StatusDelivered
	}
	
	n.updatedAt = now
	n.version++
	return nil
}

func (n *Notification) MarkChannelFailed(channel NotificationChannel, errorMsg string) error {
	delivery, exists := n.deliveryStatus[channel]
	if !exists {
		return errors.New("channel not found")
	}
	
	delivery.Status = StatusFailed
	delivery.Error = errorMsg
	delivery.RetryCount++
	
	// Check if all channels have failed
	allFailed := true
	for _, d := range n.deliveryStatus {
		if d.Status != StatusFailed {
			allFailed = false
			break
		}
	}
	if allFailed {
		n.status = StatusFailed
	}
	
	n.updatedAt = time.Now()
	n.version++
	return nil
}

func (n *Notification) ShouldRetry(channel NotificationChannel) bool {
	delivery, exists := n.deliveryStatus[channel]
	if !exists {
		return false
	}
	
	return delivery.Status == StatusFailed && delivery.RetryCount < n.maxRetries
}

func (n *Notification) IsExpired() bool {
	if n.expiresAt == nil {
		return false
	}
	return time.Now().After(*n.expiresAt)
}
