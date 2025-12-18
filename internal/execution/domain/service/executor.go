package service

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/model"
	workflowmodel "github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// NodeExecutor interface for executing different node types
type NodeExecutor interface {
	Execute(ctx context.Context, node workflowmodel.Node, input map[string]interface{}) (map[string]interface{}, error)
	GetType() workflowmodel.NodeType
}

// WorkflowExecutor orchestrates workflow execution
type WorkflowExecutor struct {
	executors map[workflowmodel.NodeType]NodeExecutor
}

// NewWorkflowExecutor creates a new workflow executor
func NewWorkflowExecutor() *WorkflowExecutor {
	executor := &WorkflowExecutor{
		executors: make(map[workflowmodel.NodeType]NodeExecutor),
	}
	
	// Register default executors
	executor.RegisterExecutor(&HTTPNodeExecutor{})
	executor.RegisterExecutor(&ConditionNodeExecutor{})
	executor.RegisterExecutor(&TransformNodeExecutor{})
	
	return executor
}

// RegisterExecutor registers a node executor
func (e *WorkflowExecutor) RegisterExecutor(executor NodeExecutor) {
	e.executors[executor.GetType()] = executor
}

// ExecuteWorkflow executes a complete workflow
func (e *WorkflowExecutor) ExecuteWorkflow(
	ctx context.Context,
	workflow *workflowmodel.Workflow,
	execution *model.Execution,
) error {
	// Start execution
	if err := execution.Start(); err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	// Find trigger nodes
	triggerNodes := e.findNodesByType(workflow, workflowmodel.NodeTypeTrigger)
	if len(triggerNodes) == 0 {
		err := model.ExecutionError{
			Code:    "NO_TRIGGER",
			Message: "No trigger node found in workflow",
		}
		execution.Fail(err)
		return fmt.Errorf("no trigger node found")
	}

	// Execute from trigger nodes
	for _, triggerNode := range triggerNodes {
		if err := e.executeNode(ctx, workflow, execution, triggerNode); err != nil {
			// Continue with other triggers if one fails
			continue
		}
	}

	// Check if execution completed successfully
	if execution.Status() == model.ExecutionStatusRunning {
		// Complete execution with final output
		execution.Complete(execution.Context().Variables)
	}

	return nil
}

// executeNode executes a single node and its downstream nodes
func (e *WorkflowExecutor) executeNode(
	ctx context.Context,
	workflow *workflowmodel.Workflow,
	execution *model.Execution,
	node workflowmodel.Node,
) error {
	// Check if node already executed
	if _, exists := execution.NodeExecutions()[node.ID]; exists {
		return nil // Already executed
	}

	// Get node executor
	executor, exists := e.executors[node.Type]
	if !exists {
		return fmt.Errorf("no executor for node type %s", node.Type)
	}

	// Prepare input data
	inputData := e.prepareNodeInput(execution, node)

	// Start node execution
	if err := execution.StartNodeExecution(node.ID, string(node.Type), inputData); err != nil {
		return fmt.Errorf("failed to start node execution: %w", err)
	}

	// Execute node with timeout
	nodeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	outputData, err := executor.Execute(nodeCtx, node, inputData)
	if err != nil {
		execErr := model.ExecutionError{
			Code:    "NODE_EXECUTION_FAILED",
			Message: err.Error(),
			Details: map[string]interface{}{
				"nodeId":   node.ID,
				"nodeType": node.Type,
			},
		}
		execution.FailNodeExecution(node.ID, execErr)
		return fmt.Errorf("node %s execution failed: %w", node.ID, err)
	}

	// Complete node execution
	if err := execution.CompleteNodeExecution(node.ID, outputData); err != nil {
		return fmt.Errorf("failed to complete node execution: %w", err)
	}

	// Execute downstream nodes
	downstreamNodes := e.findDownstreamNodes(workflow, node.ID)
	for _, downstreamNode := range downstreamNodes {
		if err := e.executeNode(ctx, workflow, execution, downstreamNode); err != nil {
			// Log error but continue with other nodes
			fmt.Printf("Error executing downstream node %s: %v\n", downstreamNode.ID, err)
		}
	}

	return nil
}

// prepareNodeInput prepares input data for a node
func (e *WorkflowExecutor) prepareNodeInput(execution *model.Execution, node workflowmodel.Node) map[string]interface{} {
	input := make(map[string]interface{})
	
	// Add execution context variables
	for key, value := range execution.Context().Variables {
		input[key] = value
	}
	
	// Add node-specific configuration
	for key, value := range node.Config {
		input[key] = value
	}
	
	return input
}

// findNodesByType finds all nodes of a specific type
func (e *WorkflowExecutor) findNodesByType(workflow *workflowmodel.Workflow, nodeType workflowmodel.NodeType) []workflowmodel.Node {
	var nodes []workflowmodel.Node
	for _, node := range workflow.Nodes() {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// findDownstreamNodes finds all nodes connected downstream from a node
func (e *WorkflowExecutor) findDownstreamNodes(workflow *workflowmodel.Workflow, nodeID string) []workflowmodel.Node {
	var downstreamNodes []workflowmodel.Node
	nodeMap := make(map[string]workflowmodel.Node)
	
	// Build node map
	for _, node := range workflow.Nodes() {
		nodeMap[node.ID] = node
	}
	
	// Find connections from this node
	for _, conn := range workflow.Connections() {
		if conn.SourceNodeID == nodeID {
			if node, exists := nodeMap[conn.TargetNodeID]; exists {
				downstreamNodes = append(downstreamNodes, node)
			}
		}
	}
	
	return downstreamNodes
}

// HTTPNodeExecutor executes HTTP request nodes
type HTTPNodeExecutor struct{}

func (e *HTTPNodeExecutor) GetType() workflowmodel.NodeType {
	return workflowmodel.NodeTypeAction
}

func (e *HTTPNodeExecutor) Execute(ctx context.Context, node workflowmodel.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// Simplified HTTP execution
	output := make(map[string]interface{})
	
	// Extract configuration
	url, _ := node.Config["url"].(string)
	method, _ := node.Config["method"].(string)
	
	if url == "" {
		return nil, fmt.Errorf("URL not configured")
	}
	
	// TODO: Implement actual HTTP request
	// For now, return mock response
	output["status"] = 200
	output["response"] = map[string]interface{}{
		"message": "HTTP request executed",
		"url":     url,
		"method":  method,
	}
	
	return output, nil
}

// ConditionNodeExecutor executes conditional nodes
type ConditionNodeExecutor struct{}

func (e *ConditionNodeExecutor) GetType() workflowmodel.NodeType {
	return workflowmodel.NodeTypeCondition
}

func (e *ConditionNodeExecutor) Execute(ctx context.Context, node workflowmodel.Node, input map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	
	// Extract condition
	condition, _ := node.Config["condition"].(string)
	if condition == "" {
		return nil, fmt.Errorf("condition not configured")
	}
	
	// TODO: Implement actual condition evaluation
	// For now, return true
	output["result"] = true
	output["condition"] = condition
	
	return output, nil
}

// TransformNodeExecutor executes data transformation nodes
type TransformNodeExecutor struct{}

func (e *TransformNodeExecutor) GetType() workflowmodel.NodeType {
	return workflowmodel.NodeTypeAction
}

func (e *TransformNodeExecutor) Execute(ctx context.Context, node workflowmodel.Node, input map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	
	// Extract transformation config
	transform, _ := node.Config["transform"].(map[string]interface{})
	
	// TODO: Implement actual data transformation
	// For now, pass through input with a transform flag
	output["transformed"] = true
	output["data"] = input
	output["transform"] = transform
	
	return output, nil
}
