// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// ErrorTriggerNode triggers on workflow errors
type ErrorTriggerNode struct{}

// NewErrorTriggerNode creates a new Error Trigger node
func NewErrorTriggerNode() *ErrorTriggerNode {
	return &ErrorTriggerNode{}
}

// GetType returns the node type
func (n *ErrorTriggerNode) GetType() string {
	return "error_trigger"
}

// GetMetadata returns node metadata
func (n *ErrorTriggerNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "error_trigger",
		Name:        "Error Trigger",
		Description: "Triggers when an error occurs in the workflow",
		Category:    "trigger",
		Icon:        "alert-triangle",
		Color:       "#F44336",
		Version:     "1.0.0",
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Error data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "errorMode", Type: "select", Default: "all", Description: "Which errors to catch", Options: []runtime.PropertyOption{
				{Label: "All Errors", Value: "all"},
				{Label: "Specific Nodes", Value: "specific"},
			}},
			{Name: "nodeNames", Type: "string", Description: "Comma-separated node names to catch errors from (for specific mode)"},
		},
		IsTrigger: true,
	}
}

// Validate validates the node configuration
func (n *ErrorTriggerNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute is called when an error triggers this node
func (n *ErrorTriggerNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	// Extract error information from input
	errorData := map[string]interface{}{
		"message":     input.InputData["message"],
		"nodeId":      input.InputData["nodeId"],
		"nodeName":    input.InputData["nodeName"],
		"nodeType":    input.InputData["nodeType"],
		"executionId": input.InputData["executionId"],
		"timestamp":   time.Now().Format(time.RFC3339),
		"stack":       input.InputData["stack"],
	}
	
	output.Data = errorData
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "error",
		Message:   fmt.Sprintf("Error caught from node: %v", input.InputData["nodeName"]),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func init() {
	runtime.Register(NewErrorTriggerNode())
}

// StopAndErrorNode stops workflow with an error
type StopAndErrorNode struct{}

// NewStopAndErrorNode creates a new Stop and Error node
func NewStopAndErrorNode() *StopAndErrorNode {
	return &StopAndErrorNode{}
}

// GetType returns the node type
func (n *StopAndErrorNode) GetType() string {
	return "stop_and_error"
}

// GetMetadata returns node metadata
func (n *StopAndErrorNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "stop_and_error",
		Name:        "Stop and Error",
		Description: "Stop workflow execution and throw an error",
		Category:    "core",
		Icon:        "x-circle",
		Color:       "#F44336",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{},
		Properties: []runtime.PropertyDefinition{
			{Name: "errorMessage", Type: "string", Required: true, Description: "Error message"},
			{Name: "errorType", Type: "select", Default: "error", Description: "Error type", Options: []runtime.PropertyOption{
				{Label: "Error", Value: "error"},
				{Label: "Warning", Value: "warning"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *StopAndErrorNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the Stop and Error node
func (n *StopAndErrorNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	errorMessage := getStringConfig(input.NodeConfig, "errorMessage", "Workflow stopped by user")
	
	output.Error = fmt.Errorf("%s", errorMessage)
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "error",
		Message:   errorMessage,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func init() {
	runtime.Register(NewStopAndErrorNode())
}

// NoOpNode does nothing (pass through)
type NoOpNode struct{}

// NewNoOpNode creates a new NoOp node
func NewNoOpNode() *NoOpNode {
	return &NoOpNode{}
}

// GetType returns the node type
func (n *NoOpNode) GetType() string {
	return "no_op"
}

// GetMetadata returns node metadata
func (n *NoOpNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "no_op",
		Name:        "No Operation",
		Description: "Does nothing, just passes data through",
		Category:    "core",
		Icon:        "minus",
		Color:       "#9E9E9E",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Same data"},
		},
		Properties: []runtime.PropertyDefinition{},
		IsTrigger:  false,
	}
}

// Validate validates the node configuration
func (n *NoOpNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute just passes through the data
func (n *NoOpNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	return &runtime.ExecutionOutput{
		Data: input.InputData,
		Metrics: runtime.ExecutionMetrics{
			StartTime:  time.Now().UnixMilli(),
			EndTime:    time.Now().UnixMilli(),
			DurationMs: 0,
		},
	}, nil
}

func init() {
	runtime.Register(NewNoOpNode())
}
