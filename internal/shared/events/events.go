package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event represents a domain event
type Event struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregateId"`
	AggregateType string                 `json:"aggregateType"`
	EventType     string                 `json:"eventType"`
	EventVersion  int                    `json:"eventVersion"`
	Timestamp     time.Time              `json:"timestamp"`
	UserID        string                 `json:"userId"`
	CorrelationID string                 `json:"correlationId"`
	CausationID   string                 `json:"causationId"`
	Metadata      map[string]interface{} `json:"metadata"`
	Payload       json.RawMessage        `json:"payload"`
}

// NewEvent creates a new event
func NewEvent(aggregateID, aggregateType, eventType string, payload interface{}) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:            uuid.New().String(),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		EventVersion:  1,
		Timestamp:     time.Now(),
		Metadata:      make(map[string]interface{}),
		Payload:       payloadBytes,
	}, nil
}

// Auth Events
type UserRegistered struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserLoggedIn struct {
	UserID    string    `json:"userId"`
	Email     string    `json:"email"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Timestamp time.Time `json:"timestamp"`
}

type UserLoggedOut struct {
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

type PasswordChanged struct {
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
}

// Workflow Events
type WorkflowCreated struct {
	WorkflowID  string    `json:"workflowId"`
	UserID      string    `json:"userId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type WorkflowUpdated struct {
	WorkflowID string                 `json:"workflowId"`
	Changes    map[string]interface{} `json:"changes"`
	UpdatedBy  string                 `json:"updatedBy"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type WorkflowDeleted struct {
	WorkflowID string    `json:"workflowId"`
	DeletedBy  string    `json:"deletedBy"`
	DeletedAt  time.Time `json:"deletedAt"`
}

type WorkflowActivated struct {
	WorkflowID  string    `json:"workflowId"`
	ActivatedBy string    `json:"activatedBy"`
	ActivatedAt time.Time `json:"activatedAt"`
}

type WorkflowDeactivated struct {
	WorkflowID    string    `json:"workflowId"`
	DeactivatedBy string    `json:"deactivatedBy"`
	DeactivatedAt time.Time `json:"deactivatedAt"`
}

// Execution Events
type ExecutionStarted struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	UserID      string                 `json:"userId"`
	TriggerType string                 `json:"triggerType"`
	InputData   map[string]interface{} `json:"inputData"`
	StartedAt   time.Time              `json:"startedAt"`
}

type ExecutionCompleted struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"`
	OutputData  map[string]interface{} `json:"outputData"`
	Duration    time.Duration          `json:"duration"`
	CompletedAt time.Time              `json:"completedAt"`
}

type ExecutionFailed struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	Error       string                 `json:"error"`
	FailedNode  string                 `json:"failedNode"`
	FailedAt    time.Time              `json:"failedAt"`
	ErrorDetails map[string]interface{} `json:"errorDetails"`
}

type NodeExecutionStarted struct {
	ExecutionID string    `json:"executionId"`
	NodeID      string    `json:"nodeId"`
	NodeType    string    `json:"nodeType"`
	StartedAt   time.Time `json:"startedAt"`
}

type NodeExecutionCompleted struct {
	ExecutionID string                 `json:"executionId"`
	NodeID      string                 `json:"nodeId"`
	OutputData  map[string]interface{} `json:"outputData"`
	Duration    time.Duration          `json:"duration"`
	CompletedAt time.Time              `json:"completedAt"`
}

// Schedule Events
type ScheduleCreated struct {
	ScheduleID   string    `json:"scheduleId"`
	WorkflowID   string    `json:"workflowId"`
	CronExpression string  `json:"cronExpression"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ScheduleTriggered struct {
	ScheduleID  string    `json:"scheduleId"`
	WorkflowID  string    `json:"workflowId"`
	ExecutionID string    `json:"executionId"`
	TriggeredAt time.Time `json:"triggeredAt"`
}

// Webhook Events
type WebhookReceived struct {
	WebhookID   string                 `json:"webhookId"`
	WorkflowID  string                 `json:"workflowId"`
	Method      string                 `json:"method"`
	Headers     map[string]string      `json:"headers"`
	Body        json.RawMessage        `json:"body"`
	ReceivedAt  time.Time              `json:"receivedAt"`
}

type WebhookProcessed struct {
	WebhookID   string        `json:"webhookId"`
	ExecutionID string        `json:"executionId"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	ProcessedAt time.Time     `json:"processedAt"`
}

// Notification Events
type NotificationSent struct {
	NotificationID string                 `json:"notificationId"`
	UserID         string                 `json:"userId"`
	Type           string                 `json:"type"`
	Channel        string                 `json:"channel"`
	Content        map[string]interface{} `json:"content"`
	SentAt         time.Time              `json:"sentAt"`
}

type NotificationFailed struct {
	NotificationID string    `json:"notificationId"`
	UserID         string    `json:"userId"`
	Type           string    `json:"type"`
	Channel        string    `json:"channel"`
	Error          string    `json:"error"`
	FailedAt       time.Time `json:"failedAt"`
}

// Helper functions
func GetEventType(event interface{}) string {
	switch event.(type) {
	// Auth events
	case UserRegistered, *UserRegistered:
		return "auth.user.registered"
	case UserLoggedIn, *UserLoggedIn:
		return "auth.user.logged_in"
	case UserLoggedOut, *UserLoggedOut:
		return "auth.user.logged_out"
	case PasswordChanged, *PasswordChanged:
		return "auth.password.changed"
		
	// Workflow events
	case WorkflowCreated, *WorkflowCreated:
		return "workflow.created"
	case WorkflowUpdated, *WorkflowUpdated:
		return "workflow.updated"
	case WorkflowDeleted, *WorkflowDeleted:
		return "workflow.deleted"
	case WorkflowActivated, *WorkflowActivated:
		return "workflow.activated"
	case WorkflowDeactivated, *WorkflowDeactivated:
		return "workflow.deactivated"
		
	// Execution events
	case ExecutionStarted, *ExecutionStarted:
		return "execution.started"
	case ExecutionCompleted, *ExecutionCompleted:
		return "execution.completed"
	case ExecutionFailed, *ExecutionFailed:
		return "execution.failed"
	case NodeExecutionStarted, *NodeExecutionStarted:
		return "execution.node.started"
	case NodeExecutionCompleted, *NodeExecutionCompleted:
		return "execution.node.completed"
		
	// Schedule events
	case ScheduleCreated, *ScheduleCreated:
		return "schedule.created"
	case ScheduleTriggered, *ScheduleTriggered:
		return "schedule.triggered"
		
	// Webhook events
	case WebhookReceived, *WebhookReceived:
		return "webhook.received"
	case WebhookProcessed, *WebhookProcessed:
		return "webhook.processed"
		
	// Notification events
	case NotificationSent, *NotificationSent:
		return "notification.sent"
	case NotificationFailed, *NotificationFailed:
		return "notification.failed"
		
	default:
		return "unknown"
	}
}
