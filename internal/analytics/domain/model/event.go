package model

import (
	"time"

	"github.com/google/uuid"
)

type EventID string

func NewEventID() EventID {
	return EventID(uuid.New().String())
}

type EventType string

const (
	EventTypePageView        EventType = "page_view"
	EventTypeWorkflowCreated EventType = "workflow_created"
	EventTypeWorkflowExecuted EventType = "workflow_executed"
	EventTypeUserAction      EventType = "user_action"
	EventTypeAPICall         EventType = "api_call"
	EventTypeError           EventType = "error"
)

type AnalyticsEvent struct {
	id         EventID
	userID     string
	sessionID  string
	eventType  EventType
	eventName  string
	properties map[string]interface{}
	metadata   map[string]interface{}
	timestamp  time.Time
	ip         string
	userAgent  string
}

func NewAnalyticsEvent(userID, sessionID string, eventType EventType, eventName string) *AnalyticsEvent {
	return &AnalyticsEvent{
		id:         NewEventID(),
		userID:     userID,
		sessionID:  sessionID,
		eventType:  eventType,
		eventName:  eventName,
		properties: make(map[string]interface{}),
		metadata:   make(map[string]interface{}),
		timestamp:  time.Now(),
	}
}

func (e *AnalyticsEvent) ID() EventID                     { return e.id }
func (e *AnalyticsEvent) UserID() string                  { return e.userID }
func (e *AnalyticsEvent) SessionID() string               { return e.sessionID }
func (e *AnalyticsEvent) EventType() EventType            { return e.eventType }
func (e *AnalyticsEvent) EventName() string               { return e.eventName }
func (e *AnalyticsEvent) Properties() map[string]interface{} { return e.properties }
func (e *AnalyticsEvent) Timestamp() time.Time            { return e.timestamp }

func (e *AnalyticsEvent) SetProperty(key string, value interface{}) {
	e.properties[key] = value
}

func (e *AnalyticsEvent) SetIP(ip string) {
	e.ip = ip
}

func (e *AnalyticsEvent) SetUserAgent(userAgent string) {
	e.userAgent = userAgent
}
