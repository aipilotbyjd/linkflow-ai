// Package features provides advanced workflow features
package features

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// SubWorkflowExecutor handles sub-workflow execution
type SubWorkflowExecutor struct {
	workflowRepo WorkflowRepository
	executor     WorkflowExecutor
	maxDepth     int
	mu           sync.RWMutex
}

// WorkflowRepository interface for fetching workflows
type WorkflowRepository interface {
	FindByID(ctx context.Context, id string) (*model.Workflow, error)
}

// WorkflowExecutor interface for executing workflows
type WorkflowExecutor interface {
	Execute(ctx context.Context, workflow *model.Workflow, input map[string]interface{}, options *ExecutionOptions) (*ExecutionResult, error)
}

// ExecutionOptions holds execution options
type ExecutionOptions struct {
	Mode        string
	UserID      string
	WorkspaceID string
	ParentID    string
	Depth       int
	Variables   map[string]interface{}
	Credentials map[string]map[string]interface{}
}

// ExecutionResult holds execution result
type ExecutionResult struct {
	ExecutionID string
	Status      string
	Output      map[string]interface{}
	Error       error
	DurationMs  int64
}

// NewSubWorkflowExecutor creates a new sub-workflow executor
func NewSubWorkflowExecutor(repo WorkflowRepository, executor WorkflowExecutor) *SubWorkflowExecutor {
	return &SubWorkflowExecutor{
		workflowRepo: repo,
		executor:     executor,
		maxDepth:     10, // Maximum nesting depth
	}
}

// SubWorkflowNode represents a sub-workflow node configuration
type SubWorkflowNode struct {
	WorkflowID    string                 `json:"workflowId"`
	InputMapping  map[string]string      `json:"inputMapping"`  // Maps parent data to sub-workflow input
	OutputMapping map[string]string      `json:"outputMapping"` // Maps sub-workflow output to parent data
	WaitForOutput bool                   `json:"waitForOutput"` // Whether to wait for completion
	Timeout       time.Duration          `json:"timeout"`
	OnError       string                 `json:"onError"` // stop, continue, fallback
	FallbackValue map[string]interface{} `json:"fallbackValue"`
}

// ExecuteSubWorkflow executes a sub-workflow
func (e *SubWorkflowExecutor) ExecuteSubWorkflow(
	ctx context.Context,
	config *SubWorkflowNode,
	input map[string]interface{},
	options *ExecutionOptions,
) (*ExecutionResult, error) {
	// Check depth limit
	if options.Depth >= e.maxDepth {
		return nil, fmt.Errorf("maximum sub-workflow nesting depth (%d) exceeded", e.maxDepth)
	}

	// Fetch sub-workflow
	workflow, err := e.workflowRepo.FindByID(ctx, config.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("sub-workflow not found: %w", err)
	}

	// Map input data
	mappedInput := e.mapInput(input, config.InputMapping)

	// Create execution options for sub-workflow
	subOptions := &ExecutionOptions{
		Mode:        "subworkflow",
		UserID:      options.UserID,
		WorkspaceID: options.WorkspaceID,
		ParentID:    options.ParentID,
		Depth:       options.Depth + 1,
		Variables:   options.Variables,
		Credentials: options.Credentials,
	}

	// Execute with timeout if specified
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	// Execute sub-workflow
	result, err := e.executor.Execute(ctx, workflow, mappedInput, subOptions)
	if err != nil {
		// Handle error based on config
		switch config.OnError {
		case "continue":
			return &ExecutionResult{
				Status: "failed",
				Output: config.FallbackValue,
				Error:  err,
			}, nil
		case "fallback":
			return &ExecutionResult{
				Status: "fallback",
				Output: config.FallbackValue,
			}, nil
		default:
			return nil, err
		}
	}

	// Map output data
	if config.OutputMapping != nil && len(config.OutputMapping) > 0 {
		result.Output = e.mapOutput(result.Output, config.OutputMapping)
	}

	return result, nil
}

func (e *SubWorkflowExecutor) mapInput(input map[string]interface{}, mapping map[string]string) map[string]interface{} {
	if mapping == nil || len(mapping) == 0 {
		return input
	}

	mapped := make(map[string]interface{})
	for targetKey, sourceKey := range mapping {
		if value, ok := input[sourceKey]; ok {
			mapped[targetKey] = value
		}
	}
	return mapped
}

func (e *SubWorkflowExecutor) mapOutput(output map[string]interface{}, mapping map[string]string) map[string]interface{} {
	mapped := make(map[string]interface{})
	for targetKey, sourceKey := range mapping {
		if value, ok := output[sourceKey]; ok {
			mapped[targetKey] = value
		}
	}
	return mapped
}

// SubWorkflowReference represents a reference to a sub-workflow
type SubWorkflowReference struct {
	ID           string
	WorkflowID   string
	ParentID     string
	Name         string
	Description  string
	InputSchema  map[string]interface{}
	OutputSchema map[string]interface{}
	CreatedAt    time.Time
}

// SubWorkflowRegistry manages sub-workflow references
type SubWorkflowRegistry struct {
	references map[string]*SubWorkflowReference
	mu         sync.RWMutex
}

// NewSubWorkflowRegistry creates a new sub-workflow registry
func NewSubWorkflowRegistry() *SubWorkflowRegistry {
	return &SubWorkflowRegistry{
		references: make(map[string]*SubWorkflowReference),
	}
}

// Register registers a workflow as a sub-workflow
func (r *SubWorkflowRegistry) Register(ref *SubWorkflowReference) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if ref.ID == "" {
		ref.ID = uuid.New().String()
	}
	ref.CreatedAt = time.Now()
	r.references[ref.ID] = ref
}

// Get retrieves a sub-workflow reference
func (r *SubWorkflowRegistry) Get(id string) (*SubWorkflowReference, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ref, ok := r.references[id]
	return ref, ok
}

// ListByParent lists sub-workflows for a parent workflow
func (r *SubWorkflowRegistry) ListByParent(parentID string) []*SubWorkflowReference {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var refs []*SubWorkflowReference
	for _, ref := range r.references {
		if ref.ParentID == parentID {
			refs = append(refs, ref)
		}
	}
	return refs
}

// ListAll lists all sub-workflow references
func (r *SubWorkflowRegistry) ListAll() []*SubWorkflowReference {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	refs := make([]*SubWorkflowReference, 0, len(r.references))
	for _, ref := range r.references {
		refs = append(refs, ref)
	}
	return refs
}

// Delete removes a sub-workflow reference
func (r *SubWorkflowRegistry) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.references, id)
}

// ValidateSubWorkflowChain validates that a sub-workflow chain doesn't have cycles
func ValidateSubWorkflowChain(repo WorkflowRepository, workflowID string, visited map[string]bool) error {
	if visited == nil {
		visited = make(map[string]bool)
	}
	
	if visited[workflowID] {
		return fmt.Errorf("circular sub-workflow reference detected: %s", workflowID)
	}
	
	visited[workflowID] = true
	
	// In a real implementation, this would fetch the workflow and check its sub-workflow nodes
	// For now, we just validate the visited map
	
	return nil
}
