package model

import "time"

// DomainEvent interface for all domain events
type DomainEvent interface {
	EventType() string
	AggregateID() string
	OccurredAt() time.Time
}

// Base event
type baseEvent struct {
	aggregateID string
	occurredAt  time.Time
}

func (e baseEvent) AggregateID() string {
	return e.aggregateID
}

func (e baseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

// WorkflowCreatedEvent is raised when a workflow is created
type WorkflowCreatedEvent struct {
	WorkflowID  WorkflowID
	UserID      string
	Name        string
	Description string
	CreatedAt   time.Time
}

func (e WorkflowCreatedEvent) EventType() string {
	return "workflow.created"
}

func (e WorkflowCreatedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e WorkflowCreatedEvent) OccurredAt() time.Time {
	return e.CreatedAt
}

// WorkflowActivatedEvent is raised when a workflow is activated
type WorkflowActivatedEvent struct {
	WorkflowID  WorkflowID
	ActivatedAt time.Time
}

func (e WorkflowActivatedEvent) EventType() string {
	return "workflow.activated"
}

func (e WorkflowActivatedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e WorkflowActivatedEvent) OccurredAt() time.Time {
	return e.ActivatedAt
}

// WorkflowDeactivatedEvent is raised when a workflow is deactivated
type WorkflowDeactivatedEvent struct {
	WorkflowID    WorkflowID
	DeactivatedAt time.Time
}

func (e WorkflowDeactivatedEvent) EventType() string {
	return "workflow.deactivated"
}

func (e WorkflowDeactivatedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e WorkflowDeactivatedEvent) OccurredAt() time.Time {
	return e.DeactivatedAt
}

// WorkflowArchivedEvent is raised when a workflow is archived
type WorkflowArchivedEvent struct {
	WorkflowID WorkflowID
	ArchivedAt time.Time
}

func (e WorkflowArchivedEvent) EventType() string {
	return "workflow.archived"
}

func (e WorkflowArchivedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e WorkflowArchivedEvent) OccurredAt() time.Time {
	return e.ArchivedAt
}

// NodeAddedEvent is raised when a node is added
type NodeAddedEvent struct {
	WorkflowID WorkflowID
	Node       Node
	AddedAt    time.Time
}

func (e NodeAddedEvent) EventType() string {
	return "workflow.node_added"
}

func (e NodeAddedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e NodeAddedEvent) OccurredAt() time.Time {
	return e.AddedAt
}

// NodeRemovedEvent is raised when a node is removed
type NodeRemovedEvent struct {
	WorkflowID WorkflowID
	NodeID     string
	RemovedAt  time.Time
}

func (e NodeRemovedEvent) EventType() string {
	return "workflow.node_removed"
}

func (e NodeRemovedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e NodeRemovedEvent) OccurredAt() time.Time {
	return e.RemovedAt
}

// ConnectionAddedEvent is raised when a connection is added
type ConnectionAddedEvent struct {
	WorkflowID WorkflowID
	Connection Connection
	AddedAt    time.Time
}

func (e ConnectionAddedEvent) EventType() string {
	return "workflow.connection_added"
}

func (e ConnectionAddedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e ConnectionAddedEvent) OccurredAt() time.Time {
	return e.AddedAt
}

// WorkflowSettingsUpdatedEvent is raised when settings are updated
type WorkflowSettingsUpdatedEvent struct {
	WorkflowID WorkflowID
	Settings   Settings
	UpdatedAt  time.Time
}

func (e WorkflowSettingsUpdatedEvent) EventType() string {
	return "workflow.settings_updated"
}

func (e WorkflowSettingsUpdatedEvent) AggregateID() string {
	return e.WorkflowID.String()
}

func (e WorkflowSettingsUpdatedEvent) OccurredAt() time.Time {
	return e.UpdatedAt
}
