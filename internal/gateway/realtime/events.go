// Package realtime provides real-time event broadcasting
package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of real-time event
type EventType string

const (
	// Execution events
	EventExecutionStarted   EventType = "execution.started"
	EventExecutionCompleted EventType = "execution.completed"
	EventExecutionFailed    EventType = "execution.failed"
	EventExecutionCancelled EventType = "execution.cancelled"
	EventExecutionPaused    EventType = "execution.paused"
	EventExecutionResumed   EventType = "execution.resumed"
	
	// Node events
	EventNodeStarted   EventType = "node.started"
	EventNodeCompleted EventType = "node.completed"
	EventNodeFailed    EventType = "node.failed"
	EventNodeSkipped   EventType = "node.skipped"
	
	// Workflow events
	EventWorkflowCreated   EventType = "workflow.created"
	EventWorkflowUpdated   EventType = "workflow.updated"
	EventWorkflowDeleted   EventType = "workflow.deleted"
	EventWorkflowActivated EventType = "workflow.activated"
	EventWorkflowDeactivated EventType = "workflow.deactivated"
	
	// Credential events
	EventCredentialCreated EventType = "credential.created"
	EventCredentialUpdated EventType = "credential.updated"
	EventCredentialDeleted EventType = "credential.deleted"
	
	// Integration events
	EventIntegrationConnected    EventType = "integration.connected"
	EventIntegrationDisconnected EventType = "integration.disconnected"
	EventIntegrationError        EventType = "integration.error"
	
	// System events
	EventSystemAlert   EventType = "system.alert"
	EventSystemStatus  EventType = "system.status"
	EventSystemMaintenance EventType = "system.maintenance"
)

// Event represents a real-time event
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Channel     string                 `json:"channel"`
	UserID      string                 `json:"userId,omitempty"`
	WorkspaceID string                 `json:"workspaceId,omitempty"`
	ResourceID  string                 `json:"resourceId,omitempty"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, data map[string]interface{}) *Event {
	return &Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ExecutionEvent represents an execution-related event
type ExecutionEvent struct {
	ExecutionID  string                 `json:"executionId"`
	WorkflowID   string                 `json:"workflowId"`
	WorkflowName string                 `json:"workflowName"`
	Status       string                 `json:"status"`
	Mode         string                 `json:"mode"`
	StartedAt    *time.Time             `json:"startedAt,omitempty"`
	CompletedAt  *time.Time             `json:"completedAt,omitempty"`
	DurationMs   int64                  `json:"durationMs,omitempty"`
	Error        string                 `json:"error,omitempty"`
	CurrentNode  string                 `json:"currentNode,omitempty"`
	Progress     float64                `json:"progress,omitempty"`
	NodeCount    int                    `json:"nodeCount,omitempty"`
	CompletedNodes int                  `json:"completedNodes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NodeEvent represents a node execution event
type NodeEvent struct {
	ExecutionID string                 `json:"executionId"`
	WorkflowID  string                 `json:"workflowId"`
	NodeID      string                 `json:"nodeId"`
	NodeName    string                 `json:"nodeName"`
	NodeType    string                 `json:"nodeType"`
	Status      string                 `json:"status"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	DurationMs  int64                  `json:"durationMs,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retryCount,omitempty"`
}

// WorkflowEvent represents a workflow event
type WorkflowEvent struct {
	WorkflowID   string                 `json:"workflowId"`
	WorkflowName string                 `json:"workflowName"`
	Action       string                 `json:"action"`
	Version      int                    `json:"version,omitempty"`
	Active       bool                   `json:"active"`
	ModifiedBy   string                 `json:"modifiedBy,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EventBroadcaster broadcasts events to connected clients
type EventBroadcaster struct {
	hub         Hub
	subscribers map[string][]chan *Event
	mu          sync.RWMutex
}

// Hub interface for WebSocket hub
type Hub interface {
	Broadcast(channel, event string, data interface{})
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster(hub Hub) *EventBroadcaster {
	return &EventBroadcaster{
		hub:         hub,
		subscribers: make(map[string][]chan *Event),
	}
}

// Broadcast broadcasts an event
func (b *EventBroadcaster) Broadcast(event *Event) {
	if b.hub == nil {
		return
	}

	// Determine channel
	channel := event.Channel
	if channel == "" {
		channel = b.getChannelForEvent(event)
	}

	// Add event metadata
	data := map[string]interface{}{
		"id":        event.ID,
		"type":      event.Type,
		"data":      event.Data,
		"timestamp": event.Timestamp,
	}

	b.hub.Broadcast(channel, string(event.Type), data)

	// Also broadcast to subscribers
	b.notifySubscribers(channel, event)
}

func (b *EventBroadcaster) getChannelForEvent(event *Event) string {
	switch {
	case event.WorkspaceID != "":
		return fmt.Sprintf("workspace:%s", event.WorkspaceID)
	case event.UserID != "":
		return fmt.Sprintf("user:%s", event.UserID)
	default:
		return "system"
	}
}

func (b *EventBroadcaster) notifySubscribers(channel string, event *Event) {
	b.mu.RLock()
	subs := b.subscribers[channel]
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// Subscribe subscribes to events on a channel
func (b *EventBroadcaster) Subscribe(channel string) <-chan *Event {
	ch := make(chan *Event, 100)
	
	b.mu.Lock()
	b.subscribers[channel] = append(b.subscribers[channel], ch)
	b.mu.Unlock()

	return ch
}

// Unsubscribe removes a subscription
func (b *EventBroadcaster) Unsubscribe(channel string, ch <-chan *Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[channel]
	for i, sub := range subs {
		if sub == ch {
			b.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

// BroadcastExecutionStarted broadcasts execution started event
func (b *EventBroadcaster) BroadcastExecutionStarted(exec *ExecutionEvent) {
	event := NewEvent(EventExecutionStarted, map[string]interface{}{
		"execution": exec,
	})
	event.ResourceID = exec.ExecutionID
	b.Broadcast(event)
}

// BroadcastExecutionCompleted broadcasts execution completed event
func (b *EventBroadcaster) BroadcastExecutionCompleted(exec *ExecutionEvent) {
	event := NewEvent(EventExecutionCompleted, map[string]interface{}{
		"execution": exec,
	})
	event.ResourceID = exec.ExecutionID
	b.Broadcast(event)
}

// BroadcastExecutionFailed broadcasts execution failed event
func (b *EventBroadcaster) BroadcastExecutionFailed(exec *ExecutionEvent) {
	event := NewEvent(EventExecutionFailed, map[string]interface{}{
		"execution": exec,
	})
	event.ResourceID = exec.ExecutionID
	b.Broadcast(event)
}

// BroadcastNodeStarted broadcasts node started event
func (b *EventBroadcaster) BroadcastNodeStarted(node *NodeEvent) {
	event := NewEvent(EventNodeStarted, map[string]interface{}{
		"node": node,
	})
	event.ResourceID = node.ExecutionID
	b.Broadcast(event)
}

// BroadcastNodeCompleted broadcasts node completed event
func (b *EventBroadcaster) BroadcastNodeCompleted(node *NodeEvent) {
	event := NewEvent(EventNodeCompleted, map[string]interface{}{
		"node": node,
	})
	event.ResourceID = node.ExecutionID
	b.Broadcast(event)
}

// BroadcastNodeFailed broadcasts node failed event
func (b *EventBroadcaster) BroadcastNodeFailed(node *NodeEvent) {
	event := NewEvent(EventNodeFailed, map[string]interface{}{
		"node": node,
	})
	event.ResourceID = node.ExecutionID
	b.Broadcast(event)
}

// BroadcastWorkflowEvent broadcasts a workflow event
func (b *EventBroadcaster) BroadcastWorkflowEvent(eventType EventType, workflow *WorkflowEvent) {
	event := NewEvent(eventType, map[string]interface{}{
		"workflow": workflow,
	})
	event.ResourceID = workflow.WorkflowID
	b.Broadcast(event)
}

// ExecutionTracker tracks execution progress for real-time updates
type ExecutionTracker struct {
	broadcaster *EventBroadcaster
	executions  map[string]*TrackedExecution
	mu          sync.RWMutex
}

// TrackedExecution represents a tracked execution
type TrackedExecution struct {
	ExecutionID    string
	WorkflowID     string
	WorkflowName   string
	TotalNodes     int
	CompletedNodes int
	Status         string
	StartedAt      time.Time
	Nodes          map[string]*TrackedNode
}

// TrackedNode represents a tracked node
type TrackedNode struct {
	NodeID    string
	NodeName  string
	NodeType  string
	Status    string
	StartedAt *time.Time
	Duration  time.Duration
}

// NewExecutionTracker creates a new execution tracker
func NewExecutionTracker(broadcaster *EventBroadcaster) *ExecutionTracker {
	return &ExecutionTracker{
		broadcaster: broadcaster,
		executions:  make(map[string]*TrackedExecution),
	}
}

// StartExecution starts tracking an execution
func (t *ExecutionTracker) StartExecution(executionID, workflowID, workflowName string, totalNodes int) {
	t.mu.Lock()
	t.executions[executionID] = &TrackedExecution{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		TotalNodes:   totalNodes,
		Status:       "running",
		StartedAt:    time.Now(),
		Nodes:        make(map[string]*TrackedNode),
	}
	t.mu.Unlock()

	startedAt := time.Now()
	t.broadcaster.BroadcastExecutionStarted(&ExecutionEvent{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		Status:       "running",
		StartedAt:    &startedAt,
		NodeCount:    totalNodes,
	})
}

// StartNode marks a node as started
func (t *ExecutionTracker) StartNode(executionID, nodeID, nodeName, nodeType string) {
	t.mu.Lock()
	exec, exists := t.executions[executionID]
	if exists {
		now := time.Now()
		exec.Nodes[nodeID] = &TrackedNode{
			NodeID:    nodeID,
			NodeName:  nodeName,
			NodeType:  nodeType,
			Status:    "running",
			StartedAt: &now,
		}
	}
	t.mu.Unlock()

	if exists {
		startedAt := time.Now()
		t.broadcaster.BroadcastNodeStarted(&NodeEvent{
			ExecutionID: executionID,
			WorkflowID:  exec.WorkflowID,
			NodeID:      nodeID,
			NodeName:    nodeName,
			NodeType:    nodeType,
			Status:      "running",
			StartedAt:   &startedAt,
		})
	}
}

// CompleteNode marks a node as completed
func (t *ExecutionTracker) CompleteNode(executionID, nodeID string, output map[string]interface{}) {
	t.mu.Lock()
	exec, exists := t.executions[executionID]
	var node *TrackedNode
	if exists {
		node = exec.Nodes[nodeID]
		if node != nil {
			node.Status = "completed"
			if node.StartedAt != nil {
				node.Duration = time.Since(*node.StartedAt)
			}
		}
		exec.CompletedNodes++
	}
	t.mu.Unlock()

	if exists && node != nil {
		completedAt := time.Now()
		t.broadcaster.BroadcastNodeCompleted(&NodeEvent{
			ExecutionID: executionID,
			WorkflowID:  exec.WorkflowID,
			NodeID:      nodeID,
			NodeName:    node.NodeName,
			NodeType:    node.NodeType,
			Status:      "completed",
			CompletedAt: &completedAt,
			DurationMs:  node.Duration.Milliseconds(),
			Output:      output,
		})
	}
}

// FailNode marks a node as failed
func (t *ExecutionTracker) FailNode(executionID, nodeID string, err error) {
	t.mu.Lock()
	exec, exists := t.executions[executionID]
	var node *TrackedNode
	if exists {
		node = exec.Nodes[nodeID]
		if node != nil {
			node.Status = "failed"
			if node.StartedAt != nil {
				node.Duration = time.Since(*node.StartedAt)
			}
		}
	}
	t.mu.Unlock()

	if exists && node != nil {
		completedAt := time.Now()
		t.broadcaster.BroadcastNodeFailed(&NodeEvent{
			ExecutionID: executionID,
			WorkflowID:  exec.WorkflowID,
			NodeID:      nodeID,
			NodeName:    node.NodeName,
			NodeType:    node.NodeType,
			Status:      "failed",
			CompletedAt: &completedAt,
			DurationMs:  node.Duration.Milliseconds(),
			Error:       err.Error(),
		})
	}
}

// CompleteExecution marks execution as completed
func (t *ExecutionTracker) CompleteExecution(executionID string, outputs map[string]interface{}) {
	t.mu.Lock()
	exec, exists := t.executions[executionID]
	if exists {
		exec.Status = "completed"
	}
	t.mu.Unlock()

	if exists {
		completedAt := time.Now()
		startedAt := exec.StartedAt
		t.broadcaster.BroadcastExecutionCompleted(&ExecutionEvent{
			ExecutionID:    executionID,
			WorkflowID:     exec.WorkflowID,
			WorkflowName:   exec.WorkflowName,
			Status:         "completed",
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			DurationMs:     completedAt.Sub(startedAt).Milliseconds(),
			NodeCount:      exec.TotalNodes,
			CompletedNodes: exec.CompletedNodes,
			Progress:       100.0,
		})

		// Cleanup
		t.mu.Lock()
		delete(t.executions, executionID)
		t.mu.Unlock()
	}
}

// FailExecution marks execution as failed
func (t *ExecutionTracker) FailExecution(executionID string, err error) {
	t.mu.Lock()
	exec, exists := t.executions[executionID]
	if exists {
		exec.Status = "failed"
	}
	t.mu.Unlock()

	if exists {
		completedAt := time.Now()
		startedAt := exec.StartedAt
		progress := float64(exec.CompletedNodes) / float64(exec.TotalNodes) * 100

		t.broadcaster.BroadcastExecutionFailed(&ExecutionEvent{
			ExecutionID:    executionID,
			WorkflowID:     exec.WorkflowID,
			WorkflowName:   exec.WorkflowName,
			Status:         "failed",
			StartedAt:      &startedAt,
			CompletedAt:    &completedAt,
			DurationMs:     completedAt.Sub(startedAt).Milliseconds(),
			Error:          err.Error(),
			NodeCount:      exec.TotalNodes,
			CompletedNodes: exec.CompletedNodes,
			Progress:       progress,
		})

		// Cleanup
		t.mu.Lock()
		delete(t.executions, executionID)
		t.mu.Unlock()
	}
}

// GetProgress returns current execution progress
func (t *ExecutionTracker) GetProgress(executionID string) (float64, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	exec, exists := t.executions[executionID]
	if !exists {
		return 0, false
	}

	if exec.TotalNodes == 0 {
		return 0, true
	}

	return float64(exec.CompletedNodes) / float64(exec.TotalNodes) * 100, true
}

// EventStore stores events for replay
type EventStore interface {
	Store(ctx context.Context, event *Event) error
	GetByExecution(ctx context.Context, executionID string) ([]*Event, error)
	GetByWorkflow(ctx context.Context, workflowID string, limit int) ([]*Event, error)
	GetRecent(ctx context.Context, limit int) ([]*Event, error)
}

// InMemoryEventStore implements EventStore in memory
type InMemoryEventStore struct {
	events []*Event
	mu     sync.RWMutex
	maxSize int
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(maxSize int) *InMemoryEventStore {
	return &InMemoryEventStore{
		events:  make([]*Event, 0),
		maxSize: maxSize,
	}
}

func (s *InMemoryEventStore) Store(ctx context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	// Trim if exceeds max size
	if len(s.events) > s.maxSize {
		s.events = s.events[len(s.events)-s.maxSize:]
	}

	return nil
}

func (s *InMemoryEventStore) GetByExecution(ctx context.Context, executionID string) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Event
	for _, e := range s.events {
		if e.ResourceID == executionID {
			result = append(result, e)
		}
	}

	return result, nil
}

func (s *InMemoryEventStore) GetByWorkflow(ctx context.Context, workflowID string, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Event
	for i := len(s.events) - 1; i >= 0 && len(result) < limit; i-- {
		e := s.events[i]
		if data, ok := e.Data["workflow"].(map[string]interface{}); ok {
			if data["workflowId"] == workflowID {
				result = append(result, e)
			}
		}
	}

	return result, nil
}

func (s *InMemoryEventStore) GetRecent(ctx context.Context, limit int) ([]*Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := len(s.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*Event, len(s.events)-start)
	copy(result, s.events[start:])

	return result, nil
}

// EventJSON returns event as JSON bytes
func (e *Event) JSON() []byte {
	data, _ := json.Marshal(e)
	return data
}
