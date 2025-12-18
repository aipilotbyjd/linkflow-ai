package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Value Objects
type WorkflowID string

func NewWorkflowID() WorkflowID {
	return WorkflowID(uuid.New().String())
}

func (id WorkflowID) String() string {
	return string(id)
}

func (id WorkflowID) Validate() error {
	if id == "" {
		return errors.New("workflow ID cannot be empty")
	}
	_, err := uuid.Parse(string(id))
	return err
}

type WorkflowStatus string

const (
	WorkflowStatusDraft    WorkflowStatus = "draft"
	WorkflowStatusActive   WorkflowStatus = "active"
	WorkflowStatusInactive WorkflowStatus = "inactive"
	WorkflowStatusArchived WorkflowStatus = "archived"
)

// Node types
type NodeType string

const (
	NodeTypeTrigger   NodeType = "trigger"
	NodeTypeAction    NodeType = "action"
	NodeTypeCondition NodeType = "condition"
	NodeTypeLoop      NodeType = "loop"
	NodeTypeOutput    NodeType = "output"
)

// Node represents a workflow node
type Node struct {
	ID          string                 `json:"id"`
	Type        NodeType               `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Position    Position               `json:"position"`
}

// Position represents node position in UI
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Connection represents a connection between nodes
type Connection struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	SourcePort   string `json:"sourcePort"`
	TargetPort   string `json:"targetPort"`
}

// Settings for workflow
type Settings struct {
	MaxExecutionTime int                    `json:"maxExecutionTime"`
	RetryPolicy      RetryPolicy            `json:"retryPolicy"`
	ErrorHandling    ErrorHandlingStrategy  `json:"errorHandling"`
	Metadata         map[string]interface{} `json:"metadata"`
}

type RetryPolicy struct {
	MaxAttempts int           `json:"maxAttempts"`
	BackoffType string        `json:"backoffType"`
	Delay       time.Duration `json:"delay"`
}

type ErrorHandlingStrategy string

const (
	ErrorHandlingStop     ErrorHandlingStrategy = "stop"
	ErrorHandlingContinue ErrorHandlingStrategy = "continue"
	ErrorHandlingRetry    ErrorHandlingStrategy = "retry"
)

// Workflow aggregate root
type Workflow struct {
	id          WorkflowID
	version     int
	events      []DomainEvent
	
	userID      string
	name        string
	description string
	status      WorkflowStatus
	nodes       []Node
	connections []Connection
	settings    Settings
	tags        []string
	createdAt   time.Time
	updatedAt   time.Time
	
	// Business invariants
	maxNodes int
}

// Factory method
func NewWorkflow(userID, name, description string) (*Workflow, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if name == "" {
		return nil, errors.New("workflow name is required")
	}
	
	w := &Workflow{
		id:          NewWorkflowID(),
		version:     0,
		userID:      userID,
		name:        name,
		description: description,
		status:      WorkflowStatusDraft,
		nodes:       make([]Node, 0),
		connections: make([]Connection, 0),
		settings:    DefaultSettings(),
		tags:        make([]string, 0),
		createdAt:   time.Now(),
		updatedAt:   time.Now(),
		maxNodes:    100,
	}
	
	w.addEvent(WorkflowCreatedEvent{
		WorkflowID:  w.id,
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   w.createdAt,
	})
	
	return w, nil
}

// DefaultSettings returns default workflow settings
func DefaultSettings() Settings {
	return Settings{
		MaxExecutionTime: 3600, // 1 hour
		RetryPolicy: RetryPolicy{
			MaxAttempts: 3,
			BackoffType: "exponential",
			Delay:       5 * time.Second,
		},
		ErrorHandling: ErrorHandlingStop,
		Metadata:      make(map[string]interface{}),
	}
}

// Domain methods
func (w *Workflow) ID() WorkflowID {
	return w.id
}

func (w *Workflow) UserID() string {
	return w.userID
}

func (w *Workflow) Name() string {
	return w.name
}

func (w *Workflow) Description() string {
	return w.description
}

func (w *Workflow) Status() WorkflowStatus {
	return w.status
}

func (w *Workflow) Nodes() []Node {
	return w.nodes
}

func (w *Workflow) Connections() []Connection {
	return w.connections
}

func (w *Workflow) Settings() Settings {
	return w.settings
}

func (w *Workflow) Version() int {
	return w.version
}

func (w *Workflow) CreatedAt() time.Time {
	return w.createdAt
}

func (w *Workflow) UpdatedAt() time.Time {
	return w.updatedAt
}

// Activate activates the workflow
func (w *Workflow) Activate() error {
	if w.status != WorkflowStatusDraft && w.status != WorkflowStatusInactive {
		return errors.New("workflow can only be activated from draft or inactive status")
	}
	
	if len(w.nodes) == 0 {
		return errors.New("workflow must have at least one node")
	}
	
	if err := w.validateConnections(); err != nil {
		return fmt.Errorf("invalid connections: %w", err)
	}
	
	// Check for trigger node
	hasTrigger := false
	for _, node := range w.nodes {
		if node.Type == NodeTypeTrigger {
			hasTrigger = true
			break
		}
	}
	
	if !hasTrigger {
		return errors.New("workflow must have at least one trigger node")
	}
	
	w.status = WorkflowStatusActive
	w.updatedAt = time.Now()
	
	w.addEvent(WorkflowActivatedEvent{
		WorkflowID:  w.id,
		ActivatedAt: w.updatedAt,
	})
	
	return nil
}

// Deactivate deactivates the workflow
func (w *Workflow) Deactivate() error {
	if w.status != WorkflowStatusActive {
		return errors.New("only active workflows can be deactivated")
	}
	
	w.status = WorkflowStatusInactive
	w.updatedAt = time.Now()
	
	w.addEvent(WorkflowDeactivatedEvent{
		WorkflowID:    w.id,
		DeactivatedAt: w.updatedAt,
	})
	
	return nil
}

// Archive archives the workflow
func (w *Workflow) Archive() error {
	if w.status == WorkflowStatusArchived {
		return errors.New("workflow is already archived")
	}
	
	w.status = WorkflowStatusArchived
	w.updatedAt = time.Now()
	
	w.addEvent(WorkflowArchivedEvent{
		WorkflowID:  w.id,
		ArchivedAt:  w.updatedAt,
	})
	
	return nil
}

// AddNode adds a node to the workflow
func (w *Workflow) AddNode(node Node) error {
	if len(w.nodes) >= w.maxNodes {
		return fmt.Errorf("workflow cannot have more than %d nodes", w.maxNodes)
	}
	
	if w.status == WorkflowStatusArchived {
		return errors.New("cannot modify archived workflow")
	}
	
	// Check for duplicate node ID
	for _, existing := range w.nodes {
		if existing.ID == node.ID {
			return errors.New("node with this ID already exists")
		}
	}
	
	// Validate node ID
	if node.ID == "" {
		node.ID = uuid.New().String()
	}
	
	w.nodes = append(w.nodes, node)
	w.updatedAt = time.Now()
	
	// Deactivate workflow when structure changes
	if w.status == WorkflowStatusActive {
		w.status = WorkflowStatusDraft
	}
	
	w.addEvent(NodeAddedEvent{
		WorkflowID: w.id,
		Node:       node,
		AddedAt:    w.updatedAt,
	})
	
	return nil
}

// RemoveNode removes a node from the workflow
func (w *Workflow) RemoveNode(nodeID string) error {
	if w.status == WorkflowStatusArchived {
		return errors.New("cannot modify archived workflow")
	}
	
	nodeIndex := -1
	for i, node := range w.nodes {
		if node.ID == nodeID {
			nodeIndex = i
			break
		}
	}
	
	if nodeIndex == -1 {
		return errors.New("node not found")
	}
	
	// Remove node
	w.nodes = append(w.nodes[:nodeIndex], w.nodes[nodeIndex+1:]...)
	
	// Remove connections related to this node
	var newConnections []Connection
	for _, conn := range w.connections {
		if conn.SourceNodeID != nodeID && conn.TargetNodeID != nodeID {
			newConnections = append(newConnections, conn)
		}
	}
	w.connections = newConnections
	
	w.updatedAt = time.Now()
	
	// Deactivate workflow when structure changes
	if w.status == WorkflowStatusActive {
		w.status = WorkflowStatusDraft
	}
	
	w.addEvent(NodeRemovedEvent{
		WorkflowID: w.id,
		NodeID:     nodeID,
		RemovedAt:  w.updatedAt,
	})
	
	return nil
}

// AddConnection adds a connection between nodes
func (w *Workflow) AddConnection(connection Connection) error {
	if w.status == WorkflowStatusArchived {
		return errors.New("cannot modify archived workflow")
	}
	
	// Validate nodes exist
	sourceExists := false
	targetExists := false
	
	for _, node := range w.nodes {
		if node.ID == connection.SourceNodeID {
			sourceExists = true
		}
		if node.ID == connection.TargetNodeID {
			targetExists = true
		}
	}
	
	if !sourceExists {
		return fmt.Errorf("source node %s not found", connection.SourceNodeID)
	}
	if !targetExists {
		return fmt.Errorf("target node %s not found", connection.TargetNodeID)
	}
	
	// Check for duplicate connection
	for _, existing := range w.connections {
		if existing.SourceNodeID == connection.SourceNodeID &&
		   existing.TargetNodeID == connection.TargetNodeID &&
		   existing.SourcePort == connection.SourcePort &&
		   existing.TargetPort == connection.TargetPort {
			return errors.New("connection already exists")
		}
	}
	
	// Generate ID if not provided
	if connection.ID == "" {
		connection.ID = uuid.New().String()
	}
	
	w.connections = append(w.connections, connection)
	w.updatedAt = time.Now()
	
	// Deactivate workflow when structure changes
	if w.status == WorkflowStatusActive {
		w.status = WorkflowStatusDraft
	}
	
	w.addEvent(ConnectionAddedEvent{
		WorkflowID:  w.id,
		Connection:  connection,
		AddedAt:     w.updatedAt,
	})
	
	return nil
}

// UpdateSettings updates workflow settings
func (w *Workflow) UpdateSettings(settings Settings) error {
	if w.status == WorkflowStatusArchived {
		return errors.New("cannot modify archived workflow")
	}
	
	w.settings = settings
	w.updatedAt = time.Now()
	
	w.addEvent(WorkflowSettingsUpdatedEvent{
		WorkflowID: w.id,
		Settings:   settings,
		UpdatedAt:  w.updatedAt,
	})
	
	return nil
}

// validateConnections validates all connections
func (w *Workflow) validateConnections() error {
	nodeMap := make(map[string]bool)
	for _, node := range w.nodes {
		nodeMap[node.ID] = true
	}
	
	for _, conn := range w.connections {
		if !nodeMap[conn.SourceNodeID] {
			return fmt.Errorf("source node %s not found", conn.SourceNodeID)
		}
		if !nodeMap[conn.TargetNodeID] {
			return fmt.Errorf("target node %s not found", conn.TargetNodeID)
		}
	}
	
	// Check for cycles
	if w.hasCycle() {
		return errors.New("workflow contains a cycle")
	}
	
	return nil
}

// hasCycle detects cycles in the workflow
func (w *Workflow) hasCycle() bool {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, conn := range w.connections {
		adj[conn.SourceNodeID] = append(adj[conn.SourceNodeID], conn.TargetNodeID)
	}
	
	// DFS to detect cycle
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	
	var hasCycleDFS func(node string) bool
	hasCycleDFS = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		
		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if hasCycleDFS(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}
		
		recStack[node] = false
		return false
	}
	
	for _, node := range w.nodes {
		if !visited[node.ID] {
			if hasCycleDFS(node.ID) {
				return true
			}
		}
	}
	
	return false
}

// Event handling
func (w *Workflow) addEvent(event DomainEvent) {
	w.events = append(w.events, event)
	w.version++
}

func (w *Workflow) GetUncommittedEvents() []DomainEvent {
	return w.events
}

func (w *Workflow) MarkEventsAsCommitted() {
	w.events = []DomainEvent{}
}

// ReconstructWorkflow reconstructs a workflow from persisted state
func ReconstructWorkflow(
	id WorkflowID,
	userID string,
	name string,
	description string,
	status WorkflowStatus,
	nodes []Node,
	connections []Connection,
	settings Settings,
	version int,
	createdAt time.Time,
	updatedAt time.Time,
) *Workflow {
	return &Workflow{
		id:          id,
		version:     version,
		userID:      userID,
		name:        name,
		description: description,
		status:      status,
		nodes:       nodes,
		connections: connections,
		settings:    settings,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		maxNodes:    100,
		events:      []DomainEvent{},
	}
}
