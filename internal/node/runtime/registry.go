// Package runtime provides node execution runtime
package runtime

import (
	"context"
	"fmt"
	"sync"
)

// NodeExecutor is the interface that all node executors must implement
type NodeExecutor interface {
	// Execute runs the node with given input and returns output
	Execute(ctx context.Context, input *ExecutionInput) (*ExecutionOutput, error)
	
	// Validate validates the node configuration
	Validate(config map[string]interface{}) error
	
	// GetType returns the node type identifier
	GetType() string
	
	// GetMetadata returns node metadata for UI
	GetMetadata() NodeMetadata
}

// ExecutionInput represents input to a node execution
type ExecutionInput struct {
	NodeID      string
	NodeConfig  map[string]interface{}
	InputData   map[string]interface{}
	Credentials map[string]interface{}
	Context     *ExecutionContext
}

// ExecutionOutput represents output from a node execution
type ExecutionOutput struct {
	Data    map[string]interface{}
	Binary  map[string][]byte
	Error   error
	Logs    []LogEntry
	Metrics ExecutionMetrics
}

// ExecutionContext provides context during execution
type ExecutionContext struct {
	ExecutionID string
	WorkflowID  string
	UserID      string
	WorkspaceID string
	Variables   map[string]interface{}
	Env         map[string]string
	Mode        string // manual, webhook, schedule, api
}

// LogEntry represents a log entry during execution
type LogEntry struct {
	Level     string // debug, info, warn, error
	Message   string
	Timestamp int64
	NodeID    string
}

// ExecutionMetrics represents execution metrics
type ExecutionMetrics struct {
	StartTime    int64
	EndTime      int64
	DurationMs   int64
	ItemsIn      int
	ItemsOut     int
	BytesRead    int64
	BytesWritten int64
}

// NodeMetadata contains metadata about a node type
type NodeMetadata struct {
	Type        string
	Name        string
	Description string
	Category    string
	Icon        string
	Color       string
	Version     string
	Inputs      []PortDefinition
	Outputs     []PortDefinition
	Properties  []PropertyDefinition
	IsTrigger   bool
	IsPremium   bool
}

// PortDefinition defines an input or output port
type PortDefinition struct {
	Name        string
	Type        string
	Required    bool
	Multiple    bool
	Description string
}

// PropertyDefinition defines a configuration property
type PropertyDefinition struct {
	Name         string
	Type         string // string, number, boolean, select, json, code, credential
	Required     bool
	Default      interface{}
	Description  string
	Options      []PropertyOption
	Placeholder  string
	DisplayOrder int
}

// PropertyOption for select properties
type PropertyOption struct {
	Label string
	Value interface{}
}

// Registry holds all registered node executors
type Registry struct {
	mu       sync.RWMutex
	nodes    map[string]NodeExecutor
	triggers map[string]TriggerExecutor
}

// TriggerExecutor is the interface for trigger nodes
type TriggerExecutor interface {
	NodeExecutor
	
	// Start starts the trigger (for polling/webhook setup)
	Start(ctx context.Context, config map[string]interface{}, callback TriggerCallback) error
	
	// Stop stops the trigger
	Stop(ctx context.Context) error
	
	// GetTriggerType returns the trigger type
	GetTriggerType() TriggerType
}

// TriggerType represents the type of trigger
type TriggerType string

const (
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypePolling  TriggerType = "polling"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeEvent    TriggerType = "event"
)

// TriggerCallback is called when a trigger fires
type TriggerCallback func(data map[string]interface{}) error

// Global registry instance
var globalRegistry = NewRegistry()

// NewRegistry creates a new node registry
func NewRegistry() *Registry {
	return &Registry{
		nodes:    make(map[string]NodeExecutor),
		triggers: make(map[string]TriggerExecutor),
	}
}

// Register registers a node executor
func (r *Registry) Register(executor NodeExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	nodeType := executor.GetType()
	if _, exists := r.nodes[nodeType]; exists {
		return fmt.Errorf("node type '%s' already registered", nodeType)
	}
	
	r.nodes[nodeType] = executor
	
	// Also register as trigger if applicable
	if trigger, ok := executor.(TriggerExecutor); ok {
		r.triggers[nodeType] = trigger
	}
	
	return nil
}

// Get returns a node executor by type
func (r *Registry) Get(nodeType string) (NodeExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	executor, exists := r.nodes[nodeType]
	if !exists {
		return nil, fmt.Errorf("node type '%s' not found", nodeType)
	}
	
	return executor, nil
}

// GetTrigger returns a trigger executor by type
func (r *Registry) GetTrigger(nodeType string) (TriggerExecutor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	trigger, exists := r.triggers[nodeType]
	if !exists {
		return nil, fmt.Errorf("trigger type '%s' not found", nodeType)
	}
	
	return trigger, nil
}

// List returns all registered node types
func (r *Registry) List() []NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]NodeMetadata, 0, len(r.nodes))
	for _, executor := range r.nodes {
		result = append(result, executor.GetMetadata())
	}
	return result
}

// ListByCategory returns nodes filtered by category
func (r *Registry) ListByCategory(category string) []NodeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result []NodeMetadata
	for _, executor := range r.nodes {
		meta := executor.GetMetadata()
		if meta.Category == category {
			result = append(result, meta)
		}
	}
	return result
}

// Global registry functions

// Register registers a node executor in the global registry
func Register(executor NodeExecutor) error {
	return globalRegistry.Register(executor)
}

// Get returns a node executor from the global registry
func Get(nodeType string) (NodeExecutor, error) {
	return globalRegistry.Get(nodeType)
}

// GetTrigger returns a trigger executor from the global registry
func GetTrigger(nodeType string) (TriggerExecutor, error) {
	return globalRegistry.GetTrigger(nodeType)
}

// List returns all registered nodes from the global registry
func List() []NodeMetadata {
	return globalRegistry.List()
}

// ListByCategory returns nodes by category from the global registry
func ListByCategory(category string) []NodeMetadata {
	return globalRegistry.ListByCategory(category)
}
