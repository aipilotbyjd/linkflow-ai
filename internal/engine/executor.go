// Package engine provides advanced workflow execution
package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/expression"
)

// AdvancedExecutor provides advanced workflow execution capabilities
type AdvancedExecutor struct {
	engine       *Engine
	parser       *expression.Parser
	pool         *WorkerPool
	queue        TaskQueue
	eventEmitter *EventEmitter
	mu           sync.RWMutex
}

// NewAdvancedExecutor creates a new advanced executor
func NewAdvancedExecutor(engine *Engine, pool *WorkerPool, queue TaskQueue) *AdvancedExecutor {
	return &AdvancedExecutor{
		engine:       engine,
		parser:       expression.NewParser(),
		pool:         pool,
		queue:        queue,
		eventEmitter: NewEventEmitter(),
	}
}

// ExecuteWorkflow executes a workflow with advanced features
func (e *AdvancedExecutor) ExecuteWorkflow(ctx context.Context, workflow *WorkflowDefinition, options *ExecutionOptions) (*ExecutionResult, error) {
	executionID := uuid.New().String()
	startTime := time.Now()

	result := &ExecutionResult{
		ExecutionID: executionID,
		WorkflowID:  workflow.ID,
		Status:      ExecutionStatusRunning,
		StartedAt:   startTime,
		NodeResults: make(map[string]*NodeResult),
		Logs:        []ExecutionLog{},
	}

	// Emit execution started event
	e.eventEmitter.Emit(ExecutionEvent{
		Type:        EventTypeExecutionStarted,
		ExecutionID: executionID,
		WorkflowID:  workflow.ID,
		Timestamp:   startTime,
	})

	// Build execution plan
	plan, err := e.buildExecutionPlan(workflow)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = err.Error()
		return result, err
	}

	// Execute plan
	err = e.executePlan(ctx, plan, workflow, options, result)
	
	endTime := time.Now()
	result.CompletedAt = &endTime
	result.DurationMs = endTime.Sub(startTime).Milliseconds()

	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = err.Error()
	} else {
		result.Status = ExecutionStatusCompleted
	}

	// Emit execution completed event
	e.eventEmitter.Emit(ExecutionEvent{
		Type:        EventTypeExecutionCompleted,
		ExecutionID: executionID,
		WorkflowID:  workflow.ID,
		Timestamp:   endTime,
		Data: map[string]interface{}{
			"status":     result.Status,
			"durationMs": result.DurationMs,
		},
	})

	return result, err
}

// ExecutionResult holds the complete result of a workflow execution
type ExecutionResult struct {
	ExecutionID string
	WorkflowID  string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt *time.Time
	DurationMs  int64
	NodeResults map[string]*NodeResult
	Outputs     map[string]interface{}
	Error       string
	Logs        []ExecutionLog
}

// NodeResult holds the result of a single node execution
type NodeResult struct {
	NodeID      string
	NodeType    string
	Status      ExecutionStatus
	StartedAt   time.Time
	CompletedAt *time.Time
	DurationMs  int64
	Input       map[string]interface{}
	Output      map[string]interface{}
	Error       string
	Retries     int
}

// ExecutionLog represents a log entry
type ExecutionLog struct {
	Timestamp time.Time
	Level     string
	NodeID    string
	Message   string
	Data      map[string]interface{}
}

// ExecutionStatus represents execution status
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
	ExecutionStatusPaused    ExecutionStatus = "paused"
)

// ExecutionPlan represents an execution plan
type ExecutionPlan struct {
	Stages     [][]*PlanNode // Nodes grouped by stage (parallel execution within stage)
	TriggerID  string
	TotalNodes int
}

// PlanNode represents a node in the execution plan
type PlanNode struct {
	ID           string
	Type         string
	Name         string
	Config       map[string]interface{}
	Dependencies []string
	Outputs      []string
	Stage        int
}

func (e *AdvancedExecutor) buildExecutionPlan(workflow *WorkflowDefinition) (*ExecutionPlan, error) {
	// Build dependency graph
	nodeMap := make(map[string]*NodeDefinition)
	for i := range workflow.Nodes {
		nodeMap[workflow.Nodes[i].ID] = &workflow.Nodes[i]
	}

	// Build adjacency list and reverse adjacency (dependencies)
	outputs := make(map[string][]string)    // node -> nodes it outputs to
	inputs := make(map[string][]string)     // node -> nodes it receives from
	inDegree := make(map[string]int)        // node -> number of inputs

	for _, node := range workflow.Nodes {
		outputs[node.ID] = []string{}
		inputs[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	for _, conn := range workflow.Connections {
		outputs[conn.SourceNodeID] = append(outputs[conn.SourceNodeID], conn.TargetNodeID)
		inputs[conn.TargetNodeID] = append(inputs[conn.TargetNodeID], conn.SourceNodeID)
		inDegree[conn.TargetNodeID]++
	}

	// Find trigger node (in-degree 0 and is trigger type)
	var triggerID string
	for _, node := range workflow.Nodes {
		if inDegree[node.ID] == 0 {
			meta, err := e.engine.getNodeMetadata(node.Type)
			if err == nil && meta.IsTrigger {
				triggerID = node.ID
				break
			}
		}
	}

	if triggerID == "" {
		return nil, fmt.Errorf("no trigger node found")
	}

	// Topological sort with stages (Kahn's algorithm with levels)
	stages := make([][]*PlanNode, 0)
	queue := []string{triggerID}
	remaining := make(map[string]int)
	for k, v := range inDegree {
		remaining[k] = v
	}

	processed := make(map[string]bool)

	for len(queue) > 0 {
		stage := make([]*PlanNode, 0)
		nextQueue := []string{}

		for _, nodeID := range queue {
			if processed[nodeID] {
				continue
			}

			node := nodeMap[nodeID]
			if node == nil {
				continue
			}

			stage = append(stage, &PlanNode{
				ID:           node.ID,
				Type:         node.Type,
				Name:         node.Name,
				Config:       node.Config,
				Dependencies: inputs[node.ID],
				Outputs:      outputs[node.ID],
				Stage:        len(stages),
			})
			processed[nodeID] = true

			// Decrease in-degree of dependents
			for _, outID := range outputs[nodeID] {
				remaining[outID]--
				if remaining[outID] == 0 {
					nextQueue = append(nextQueue, outID)
				}
			}
		}

		if len(stage) > 0 {
			stages = append(stages, stage)
		}
		queue = nextQueue
	}

	// Check if all nodes were processed
	if len(processed) != len(workflow.Nodes) {
		return nil, fmt.Errorf("workflow contains cycles or disconnected nodes")
	}

	return &ExecutionPlan{
		Stages:     stages,
		TriggerID:  triggerID,
		TotalNodes: len(workflow.Nodes),
	}, nil
}

func (e *AdvancedExecutor) executePlan(
	ctx context.Context,
	plan *ExecutionPlan,
	workflow *WorkflowDefinition,
	options *ExecutionOptions,
	result *ExecutionResult,
) error {
	nodeOutputs := make(map[string]map[string]interface{})

	// Set trigger data
	if options.TriggerData != nil {
		nodeOutputs[plan.TriggerID] = options.TriggerData
	}

	// Execute stages sequentially, nodes within stage in parallel
	for stageIdx, stage := range plan.Stages {
		result.Logs = append(result.Logs, ExecutionLog{
			Timestamp: time.Now(),
			Level:     "info",
			Message:   fmt.Sprintf("Starting stage %d with %d nodes", stageIdx, len(stage)),
		})

		// Execute nodes in parallel
		var wg sync.WaitGroup
		errChan := make(chan error, len(stage))
		resultChan := make(chan *NodeResult, len(stage))

		for _, planNode := range stage {
			wg.Add(1)
			go func(pn *PlanNode) {
				defer wg.Done()

				nodeResult := e.executeNode(ctx, pn, workflow, options, nodeOutputs)
				resultChan <- nodeResult

				if nodeResult.Error != "" {
					errChan <- fmt.Errorf("node %s failed: %s", pn.ID, nodeResult.Error)
				}
			}(planNode)
		}

		// Wait for all nodes in stage to complete
		wg.Wait()
		close(errChan)
		close(resultChan)

		// Collect results
		for nodeResult := range resultChan {
			result.NodeResults[nodeResult.NodeID] = nodeResult
			if nodeResult.Output != nil {
				nodeOutputs[nodeResult.NodeID] = nodeResult.Output
			}
		}

		// Check for errors
		for err := range errChan {
			if workflow.Settings.ErrorHandling == "stop" {
				return err
			}
			// Log error and continue
			result.Logs = append(result.Logs, ExecutionLog{
				Timestamp: time.Now(),
				Level:     "error",
				Message:   err.Error(),
			})
		}
	}

	// Collect final outputs from nodes with no outgoing connections
	result.Outputs = make(map[string]interface{})
	for _, stage := range plan.Stages {
		for _, node := range stage {
			if len(node.Outputs) == 0 {
				// This is an output node
				if output, ok := nodeOutputs[node.ID]; ok {
					result.Outputs[node.ID] = output
				}
			}
		}
	}

	return nil
}

func (e *AdvancedExecutor) executeNode(
	ctx context.Context,
	planNode *PlanNode,
	workflow *WorkflowDefinition,
	options *ExecutionOptions,
	nodeOutputs map[string]map[string]interface{},
) *NodeResult {
	startTime := time.Now()
	
	result := &NodeResult{
		NodeID:    planNode.ID,
		NodeType:  planNode.Type,
		Status:    ExecutionStatusRunning,
		StartedAt: startTime,
	}

	// Get executor
	executor, err := runtime.Get(planNode.Type)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("executor not found: %v", err)
		return result
	}

	// Build input from dependencies
	inputData := make(map[string]interface{})
	for _, depID := range planNode.Dependencies {
		if output, ok := nodeOutputs[depID]; ok {
			for k, v := range output {
				if k != "_output" && k != "_loopState" {
					inputData[k] = v
				}
			}
		}
	}
	result.Input = inputData

	// Evaluate expressions in config
	exprCtx := expression.NewContext()
	exprCtx.SetInput(inputData)
	exprCtx.Env = options.Environment
	exprCtx.Variables = options.Variables
	for nodeID, output := range nodeOutputs {
		exprCtx.SetNodeOutput(nodeID, output)
	}

	evaluatedConfig, err := e.parser.EvaluateTemplate(planNode.Config, exprCtx)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("config evaluation failed: %v", err)
		return result
	}

	// Get credentials
	var credentials map[string]interface{}
	for _, node := range workflow.Nodes {
		if node.ID == planNode.ID && node.Credential != "" && options.Credentials != nil {
			credentials = options.Credentials[node.Credential]
			break
		}
	}

	// Build execution context
	execCtx := &runtime.ExecutionContext{
		ExecutionID: uuid.New().String(),
		WorkflowID:  workflow.ID,
		UserID:      options.UserID,
		WorkspaceID: options.WorkspaceID,
		Variables:   options.Variables,
		Env:         options.Environment,
		Mode:        options.Mode,
	}

	// Execute node
	input := &runtime.ExecutionInput{
		NodeID:      planNode.ID,
		NodeConfig:  evaluatedConfig,
		InputData:   inputData,
		Credentials: credentials,
		Context:     execCtx,
	}

	output, err := executor.Execute(ctx, input)
	
	endTime := time.Now()
	result.CompletedAt = &endTime
	result.DurationMs = endTime.Sub(startTime).Milliseconds()

	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = err.Error()
		return result
	}

	if output.Error != nil {
		result.Status = ExecutionStatusFailed
		result.Error = output.Error.Error()
		return result
	}

	result.Status = ExecutionStatusCompleted
	result.Output = output.Data

	return result
}

// ExecuteNodeAsync executes a single node asynchronously
func (e *AdvancedExecutor) ExecuteNodeAsync(ctx context.Context, nodeID string, workflow *WorkflowDefinition, options *ExecutionOptions, input map[string]interface{}) (string, error) {
	taskID := uuid.New().String()

	task := &Task{
		ID:          taskID,
		Type:        TaskTypeNodeExecution,
		ExecutionID: uuid.New().String(),
		WorkflowID:  workflow.ID,
		NodeID:      nodeID,
		Workflow:    workflow,
		Options:     options,
		Priority:    5,
		Timeout:     5 * time.Minute,
		MaxRetries:  3,
		Metadata: map[string]interface{}{
			"input": input,
		},
	}

	if e.queue != nil {
		return taskID, e.queue.Enqueue(ctx, task)
	}

	return taskID, fmt.Errorf("no queue configured")
}

// GetExecutionStatus returns the status of an execution
func (e *AdvancedExecutor) GetExecutionStatus(executionID string) (*ExecutionResult, error) {
	// Would need to track executions in a map or database
	return nil, fmt.Errorf("execution %s not found", executionID)
}

// EventEmitter handles execution events
type EventEmitter struct {
	handlers map[EventType][]EventHandler
	mu       sync.RWMutex
}

// EventType represents the type of event
type EventType string

const (
	EventTypeExecutionStarted   EventType = "execution.started"
	EventTypeExecutionCompleted EventType = "execution.completed"
	EventTypeExecutionFailed    EventType = "execution.failed"
	EventTypeNodeStarted        EventType = "node.started"
	EventTypeNodeCompleted      EventType = "node.completed"
	EventTypeNodeFailed         EventType = "node.failed"
)

// ExecutionEvent represents an execution event
type ExecutionEvent struct {
	Type        EventType
	ExecutionID string
	WorkflowID  string
	NodeID      string
	Timestamp   time.Time
	Data        map[string]interface{}
}

// EventHandler handles an event
type EventHandler func(event ExecutionEvent)

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		handlers: make(map[EventType][]EventHandler),
	}
}

// On registers an event handler
func (e *EventEmitter) On(eventType EventType, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers[eventType] = append(e.handlers[eventType], handler)
}

// Off removes all handlers for an event type
func (e *EventEmitter) Off(eventType EventType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.handlers, eventType)
}

// Emit emits an event
func (e *EventEmitter) Emit(event ExecutionEvent) {
	e.mu.RLock()
	handlers := e.handlers[event.Type]
	e.mu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// SortedExecution sorts nodes for execution
func SortedExecution(nodes []NodeDefinition, connections []Connection) ([]NodeDefinition, error) {
	// Build in-degree map
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for _, node := range nodes {
		inDegree[node.ID] = 0
		adj[node.ID] = []string{}
	}

	for _, conn := range connections {
		adj[conn.SourceNodeID] = append(adj[conn.SourceNodeID], conn.TargetNodeID)
		inDegree[conn.TargetNodeID]++
	}

	// Topological sort
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []NodeDefinition
	nodeMap := make(map[string]NodeDefinition)
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}

	for len(queue) > 0 {
		nodeID := queue[0]
		queue = queue[1:]

		sorted = append(sorted, nodeMap[nodeID])

		for _, nextID := range adj[nodeID] {
			inDegree[nextID]--
			if inDegree[nextID] == 0 {
				queue = append(queue, nextID)
			}
		}
	}

	if len(sorted) != len(nodes) {
		return nil, fmt.Errorf("workflow contains cycles")
	}

	// Sort within each stage by name for deterministic order
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted, nil
}
