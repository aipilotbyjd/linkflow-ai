// Package events defines domain events for all services
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType defines the type of event
type EventType string

// Event types for all services
const (
	// Workflow events
	WorkflowCreated   EventType = "workflow.created"
	WorkflowUpdated   EventType = "workflow.updated"
	WorkflowDeleted   EventType = "workflow.deleted"
	WorkflowActivated EventType = "workflow.activated"
	WorkflowArchived  EventType = "workflow.archived"

	// Execution events
	ExecutionStarted   EventType = "execution.started"
	ExecutionCompleted EventType = "execution.completed"
	ExecutionFailed    EventType = "execution.failed"
	ExecutionCancelled EventType = "execution.cancelled"
	ExecutionRetried   EventType = "execution.retried"

	// Node events
	NodeExecutionStarted   EventType = "node.execution.started"
	NodeExecutionCompleted EventType = "node.execution.completed"
	NodeExecutionFailed    EventType = "node.execution.failed"

	// User events
	UserCreated      EventType = "user.created"
	UserUpdated      EventType = "user.updated"
	UserDeleted      EventType = "user.deleted"
	UserLoggedIn     EventType = "user.logged_in"
	UserLoggedOut    EventType = "user.logged_out"
	UserPasswordChanged EventType = "user.password_changed"

	// Auth events
	TokenGenerated EventType = "auth.token.generated"
	TokenRefreshed EventType = "auth.token.refreshed"
	TokenRevoked   EventType = "auth.token.revoked"
	MFAEnabled     EventType = "auth.mfa.enabled"
	MFADisabled    EventType = "auth.mfa.disabled"

	// Tenant events
	TenantCreated    EventType = "tenant.created"
	TenantUpdated    EventType = "tenant.updated"
	TenantSuspended  EventType = "tenant.suspended"
	TenantActivated  EventType = "tenant.activated"
	TenantPlanChanged EventType = "tenant.plan_changed"

	// Credential events
	CredentialCreated EventType = "credential.created"
	CredentialUpdated EventType = "credential.updated"
	CredentialDeleted EventType = "credential.deleted"
	CredentialUsed    EventType = "credential.used"

	// Schedule events
	ScheduleCreated   EventType = "schedule.created"
	ScheduleUpdated   EventType = "schedule.updated"
	ScheduleDeleted   EventType = "schedule.deleted"
	ScheduleTriggered EventType = "schedule.triggered"
	SchedulePaused    EventType = "schedule.paused"
	ScheduleResumed   EventType = "schedule.resumed"

	// Webhook events
	WebhookCreated   EventType = "webhook.created"
	WebhookReceived  EventType = "webhook.received"
	WebhookDelivered EventType = "webhook.delivered"
	WebhookFailed    EventType = "webhook.failed"

	// Notification events
	NotificationSent   EventType = "notification.sent"
	NotificationFailed EventType = "notification.failed"
	NotificationRead   EventType = "notification.read"

	// Integration events
	IntegrationConnected    EventType = "integration.connected"
	IntegrationDisconnected EventType = "integration.disconnected"
	IntegrationSynced       EventType = "integration.synced"

	// Storage events
	FileUploaded EventType = "storage.file.uploaded"
	FileDeleted  EventType = "storage.file.deleted"
	FileAccessed EventType = "storage.file.accessed"

	// Billing events
	InvoiceCreated EventType = "billing.invoice.created"
	PaymentReceived EventType = "billing.payment.received"
	PaymentFailed   EventType = "billing.payment.failed"
	SubscriptionChanged EventType = "billing.subscription.changed"
)

// Event represents a domain event
type Event struct {
	ID            string          `json:"id"`
	Type          EventType       `json:"type"`
	AggregateID   string          `json:"aggregateId"`
	AggregateType string          `json:"aggregateType"`
	TenantID      string          `json:"tenantId,omitempty"`
	UserID        string          `json:"userId,omitempty"`
	Timestamp     time.Time       `json:"timestamp"`
	Version       int             `json:"version"`
	Data          json.RawMessage `json:"data"`
	Metadata      Metadata        `json:"metadata"`
}

// Metadata contains event metadata
type Metadata struct {
	CorrelationID string            `json:"correlationId,omitempty"`
	CausationID   string            `json:"causationId,omitempty"`
	Source        string            `json:"source,omitempty"`
	TraceID       string            `json:"traceId,omitempty"`
	SpanID        string            `json:"spanId,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, aggregateID, aggregateType string, data interface{}) (*Event, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:            uuid.New().String(),
		Type:          eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Timestamp:     time.Now().UTC(),
		Version:       1,
		Data:          dataBytes,
		Metadata:      Metadata{},
	}, nil
}

// WithTenant sets the tenant ID
func (e *Event) WithTenant(tenantID string) *Event {
	e.TenantID = tenantID
	return e
}

// WithUser sets the user ID
func (e *Event) WithUser(userID string) *Event {
	e.UserID = userID
	return e
}

// WithCorrelation sets the correlation ID
func (e *Event) WithCorrelation(correlationID string) *Event {
	e.Metadata.CorrelationID = correlationID
	return e
}

// WithCausation sets the causation ID
func (e *Event) WithCausation(causationID string) *Event {
	e.Metadata.CausationID = causationID
	return e
}

// WithSource sets the source service
func (e *Event) WithSource(source string) *Event {
	e.Metadata.Source = source
	return e
}

// GetData unmarshals the event data into the provided type
func (e *Event) GetData(v interface{}) error {
	return json.Unmarshal(e.Data, v)
}

// Topic returns the Kafka topic for this event
func (e *Event) Topic() string {
	switch {
	case e.Type >= WorkflowCreated && e.Type <= WorkflowArchived:
		return "linkflow.workflow.events"
	case e.Type >= ExecutionStarted && e.Type <= ExecutionRetried:
		return "linkflow.execution.events"
	case e.Type >= NodeExecutionStarted && e.Type <= NodeExecutionFailed:
		return "linkflow.node.events"
	case e.Type >= UserCreated && e.Type <= UserPasswordChanged:
		return "linkflow.user.events"
	case e.Type >= TokenGenerated && e.Type <= MFADisabled:
		return "linkflow.auth.events"
	case e.Type >= TenantCreated && e.Type <= TenantPlanChanged:
		return "linkflow.tenant.events"
	case e.Type >= CredentialCreated && e.Type <= CredentialUsed:
		return "linkflow.credential.events"
	case e.Type >= ScheduleCreated && e.Type <= ScheduleResumed:
		return "linkflow.schedule.events"
	case e.Type >= WebhookCreated && e.Type <= WebhookFailed:
		return "linkflow.webhook.events"
	case e.Type >= NotificationSent && e.Type <= NotificationRead:
		return "linkflow.notification.events"
	case e.Type >= IntegrationConnected && e.Type <= IntegrationSynced:
		return "linkflow.integration.events"
	case e.Type >= FileUploaded && e.Type <= FileAccessed:
		return "linkflow.storage.events"
	case e.Type >= InvoiceCreated && e.Type <= SubscriptionChanged:
		return "linkflow.billing.events"
	default:
		return "linkflow.default.events"
	}
}

// Specific event data structures

// WorkflowCreatedData contains data for workflow created event
type WorkflowCreatedData struct {
	WorkflowID  string `json:"workflowId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy"`
}

// ExecutionStartedData contains data for execution started event
type ExecutionStartedData struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	TriggerType string                 `json:"triggerType"`
	InputData   map[string]interface{} `json:"inputData"`
}

// ExecutionCompletedData contains data for execution completed event
type ExecutionCompletedData struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	Status      string                 `json:"status"`
	Duration    int64                  `json:"duration"`
	OutputData  map[string]interface{} `json:"outputData"`
}

// ExecutionFailedData contains data for execution failed event
type ExecutionFailedData struct {
	ExecutionID string `json:"executionId"`
	WorkflowID  string `json:"workflowId"`
	Error       string `json:"error"`
	FailedNode  string `json:"failedNode,omitempty"`
}

// UserCreatedData contains data for user created event
type UserCreatedData struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// CredentialUsedData contains data for credential used event
type CredentialUsedData struct {
	CredentialID string `json:"credentialId"`
	ExecutionID  string `json:"executionId"`
	NodeID       string `json:"nodeId"`
}

// ScheduleTriggeredData contains data for schedule triggered event
type ScheduleTriggeredData struct {
	ScheduleID  string    `json:"scheduleId"`
	WorkflowID  string    `json:"workflowId"`
	ExecutionID string    `json:"executionId"`
	ScheduledAt time.Time `json:"scheduledAt"`
}

// NotificationSentData contains data for notification sent event
type NotificationSentData struct {
	NotificationID string `json:"notificationId"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Subject        string `json:"subject"`
}
