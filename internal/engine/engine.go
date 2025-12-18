// Package engine provides the workflow execution engine
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/expression"
)

// Engine orchestrates workflow execution
type Engine struct {
	parser      *expression.Parser
	executions  map[string]*ExecutionState
	mu          sync.RWMutex
	maxParallel int
}

// ExecutionState tracks the state of a workflow execution
type ExecutionState struct {
	ID           string
	WorkflowID   string
	Status       string
	StartedAt    time.Time
	CompletedAt  *time.Time
	CurrentNode  string
	NodeOutputs  map[string]map[string]interface{}
	Error        error
	Logs         []runtime.LogEntry
	cancel       context.CancelFunc
}

// WorkflowDefinition represents a workflow to execute
type WorkflowDefinition struct {
	ID          string
	Name        string
	Nodes       []NodeDefinition
	Connections []Connection
	Settings    WorkflowSettings
}

// NodeDefinition represents a node in the workflow
type NodeDefinition struct {
	ID         string
	Type       string
	Name       string
	Config     map[string]interface{}
	Position   Position
	Credential string
}

// Connection represents a connection between nodes
type Connection struct {
	SourceNodeID string
	SourcePort   string
	TargetNodeID string
	TargetPort   string
}

// Position represents node position
type Position struct {
	X float64
	Y float64
}

// WorkflowSettings represents workflow settings
type WorkflowSettings struct {
	MaxExecutionTime int
	RetryOnFail      bool
	MaxRetries       int
	ErrorHandling    string
}

// ExecutionOptions represents execution options
type ExecutionOptions struct {
	Mode         string // manual, webhook, schedule, api
	TriggerData  map[string]interface{}
	Credentials  map[string]map[string]interface{}
	Variables    map[string]interface{}
	Environment  map[string]string
	UserID       string
	WorkspaceID  string
}

// NewEngine creates a new workflow engine
func NewEngine() *Engine {
	return &Engine{
		parser:      expression.NewParser(),
		executions:  make(map[string]*ExecutionState),
		maxParallel: 10,
	}
}

// Execute executes a workflow
func (e *Engine) Execute(ctx context.Context, workflow *WorkflowDefinition, options *ExecutionOptions) (*ExecutionState, error) {
	// Create execution state
	executionID := uuid.New().String()
	execCtx, cancel := context.WithCancel(ctx)
	
	state := &ExecutionState{
		ID:          executionID,
		WorkflowID:  workflow.ID,
		Status:      "running",
		StartedAt:   time.Now(),
		NodeOutputs: make(map[string]map[string]interface{}),
		Logs:        []runtime.LogEntry{},
		cancel:      cancel,
	}
	
	e.mu.Lock()
	e.executions[executionID] = state
	e.mu.Unlock()
	
	// Find trigger node
	var triggerNode *NodeDefinition
	for i := range workflow.Nodes {
		node := &workflow.Nodes[i]
		meta, err := e.getNodeMetadata(node.Type)
		if err == nil && meta.IsTrigger {
			triggerNode = node
			break
		}
	}
	
	if triggerNode == nil {
		state.Status = "failed"
		state.Error = fmt.Errorf("workflow has no trigger node")
		return state, state.Error
	}
	
	// Set trigger data as initial output
	if options.TriggerData != nil {
		state.NodeOutputs[triggerNode.ID] = options.TriggerData
	}
	
	// Build execution graph
	graph := e.buildExecutionGraph(workflow)
	
	// Execute starting from trigger
	err := e.executeFromNode(execCtx, workflow, state, triggerNode.ID, graph, options)
	
	if err != nil {
		state.Status = "failed"
		state.Error = err
	} else {
		state.Status = "completed"
	}
	
	now := time.Now()
	state.CompletedAt = &now
	
	return state, err
}

func (e *Engine) buildExecutionGraph(workflow *WorkflowDefinition) map[string][]string {
	// Build adjacency list: node -> next nodes
	graph := make(map[string][]string)
	
	for _, conn := range workflow.Connections {
		graph[conn.SourceNodeID] = append(graph[conn.SourceNodeID], conn.TargetNodeID)
	}
	
	return graph
}

func (e *Engine) executeFromNode(
	ctx context.Context,
	workflow *WorkflowDefinition,
	state *ExecutionState,
	nodeID string,
	graph map[string][]string,
	options *ExecutionOptions,
) error {
	// Find node definition
	var nodeDef *NodeDefinition
	for i := range workflow.Nodes {
		if workflow.Nodes[i].ID == nodeID {
			nodeDef = &workflow.Nodes[i]
			break
		}
	}
	
	if nodeDef == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	state.CurrentNode = nodeID
	
	// Get node executor
	executor, err := runtime.Get(nodeDef.Type)
	if err != nil {
		return fmt.Errorf("executor for node type %s not found: %w", nodeDef.Type, err)
	}
	
	// Build input from previous nodes
	inputData := e.buildNodeInput(workflow, state, nodeID)
	
	// Build execution context
	execCtx := &runtime.ExecutionContext{
		ExecutionID: state.ID,
		WorkflowID:  workflow.ID,
		UserID:      options.UserID,
		WorkspaceID: options.WorkspaceID,
		Variables:   options.Variables,
		Env:         options.Environment,
		Mode:        options.Mode,
	}
	
	// Get credentials if specified
	var credentials map[string]interface{}
	if nodeDef.Credential != "" && options.Credentials != nil {
		credentials = options.Credentials[nodeDef.Credential]
	}
	
	// Evaluate expressions in config
	evaluatedConfig, err := e.evaluateConfig(nodeDef.Config, state.NodeOutputs, inputData, options)
	if err != nil {
		return fmt.Errorf("failed to evaluate config: %w", err)
	}
	
	// Execute node
	input := &runtime.ExecutionInput{
		NodeID:      nodeID,
		NodeConfig:  evaluatedConfig,
		InputData:   inputData,
		Credentials: credentials,
		Context:     execCtx,
	}
	
	state.Logs = append(state.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Executing node: %s (%s)", nodeDef.Name, nodeDef.Type),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    nodeID,
	})
	
	output, err := executor.Execute(ctx, input)
	if err != nil {
		return fmt.Errorf("node %s execution failed: %w", nodeID, err)
	}
	
	if output.Error != nil {
		// Handle error based on settings
		if workflow.Settings.ErrorHandling == "stop" {
			return fmt.Errorf("node %s error: %w", nodeID, output.Error)
		}
		// Continue on error
		state.Logs = append(state.Logs, runtime.LogEntry{
			Level:     "warn",
			Message:   fmt.Sprintf("Node %s error (continuing): %v", nodeID, output.Error),
			Timestamp: time.Now().UnixMilli(),
			NodeID:    nodeID,
		})
	}
	
	// Store output
	state.NodeOutputs[nodeID] = output.Data
	state.Logs = append(state.Logs, output.Logs...)
	
	// Determine next nodes
	nextNodes := graph[nodeID]
	
	// Handle branching (IF/Switch)
	if outputPort, ok := output.Data["_output"].(string); ok {
		// Find connections from this specific port
		nextNodes = []string{}
		for _, conn := range workflow.Connections {
			if conn.SourceNodeID == nodeID && conn.SourcePort == outputPort {
				nextNodes = append(nextNodes, conn.TargetNodeID)
			}
		}
	}
	
	// Execute next nodes
	for _, nextNodeID := range nextNodes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := e.executeFromNode(ctx, workflow, state, nextNodeID, graph, options); err != nil {
				return err
			}
		}
	}
	
	return nil
}

func (e *Engine) buildNodeInput(workflow *WorkflowDefinition, state *ExecutionState, nodeID string) map[string]interface{} {
	input := make(map[string]interface{})
	
	// Find incoming connections
	for _, conn := range workflow.Connections {
		if conn.TargetNodeID == nodeID {
			if sourceOutput, ok := state.NodeOutputs[conn.SourceNodeID]; ok {
				// Merge data from source
				for k, v := range sourceOutput {
					if k != "_output" && k != "_loopState" {
						input[k] = v
					}
				}
				
				// Also set input by port name
				input[conn.TargetPort] = sourceOutput
			}
		}
	}
	
	return input
}

func (e *Engine) evaluateConfig(
	config map[string]interface{},
	nodeOutputs map[string]map[string]interface{},
	inputData map[string]interface{},
	options *ExecutionOptions,
) (map[string]interface{}, error) {
	// Create expression context
	ctx := expression.NewContext()
	ctx.SetInput(inputData)
	ctx.Env = options.Environment
	ctx.Variables = options.Variables
	
	// Add node outputs to context
	for nodeID, output := range nodeOutputs {
		ctx.SetNodeOutput(nodeID, output)
	}
	
	// Evaluate all string values that might contain expressions
	return e.parser.EvaluateTemplate(config, ctx)
}

func (e *Engine) getNodeMetadata(nodeType string) (runtime.NodeMetadata, error) {
	executor, err := runtime.Get(nodeType)
	if err != nil {
		return runtime.NodeMetadata{}, err
	}
	return executor.GetMetadata(), nil
}

// CancelExecution cancels a running execution
func (e *Engine) CancelExecution(executionID string) error {
	e.mu.RLock()
	state, exists := e.executions[executionID]
	e.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}
	
	if state.cancel != nil {
		state.cancel()
	}
	
	state.Status = "cancelled"
	now := time.Now()
	state.CompletedAt = &now
	
	return nil
}

// GetExecution returns an execution state
func (e *Engine) GetExecution(executionID string) (*ExecutionState, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	state, exists := e.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}
	
	return state, nil
}

// ListExecutions lists recent executions
func (e *Engine) ListExecutions(workflowID string, limit int) []*ExecutionState {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	var executions []*ExecutionState
	for _, state := range e.executions {
		if workflowID == "" || state.WorkflowID == workflowID {
			executions = append(executions, state)
		}
	}
	
	// Sort by start time (newest first)
	// In production, this would be done at the database level
	
	if len(executions) > limit {
		executions = executions[:limit]
	}
	
	return executions
}

// CleanupOldExecutions removes old completed executions
func (e *Engine) CleanupOldExecutions(maxAge time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for id, state := range e.executions {
		if state.CompletedAt != nil && state.CompletedAt.Before(cutoff) {
			delete(e.executions, id)
		}
	}
}
