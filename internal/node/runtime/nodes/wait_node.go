// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// WaitNode implements delay functionality
type WaitNode struct{}

// NewWaitNode creates a new Wait node
func NewWaitNode() *WaitNode {
	return &WaitNode{}
}

// GetType returns the node type
func (n *WaitNode) GetType() string {
	return "wait"
}

// GetMetadata returns node metadata
func (n *WaitNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "wait",
		Name:        "Wait",
		Description: "Wait for a specified amount of time or until a condition",
		Category:    "core",
		Icon:        "clock",
		Color:       "#9E9E9E",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Data after wait"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "amount", Type: "number", Default: 1, Required: true, Description: "Amount to wait"},
			{Name: "unit", Type: "select", Default: "seconds", Description: "Time unit", Options: []runtime.PropertyOption{
				{Label: "Milliseconds", Value: "milliseconds"},
				{Label: "Seconds", Value: "seconds"},
				{Label: "Minutes", Value: "minutes"},
				{Label: "Hours", Value: "hours"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *WaitNode) Validate(config map[string]interface{}) error {
	amount := getIntConfig(config, "amount", 1)
	if amount < 0 {
		return fmt.Errorf("wait amount cannot be negative")
	}
	return nil
}

// Execute executes the Wait node
func (n *WaitNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: input.InputData,
		Logs: []runtime.LogEntry{},
	}
	
	amount := getIntConfig(input.NodeConfig, "amount", 1)
	unit := getStringConfig(input.NodeConfig, "unit", "seconds")
	
	// Calculate duration
	var duration time.Duration
	switch unit {
	case "milliseconds":
		duration = time.Duration(amount) * time.Millisecond
	case "seconds":
		duration = time.Duration(amount) * time.Second
	case "minutes":
		duration = time.Duration(amount) * time.Minute
	case "hours":
		duration = time.Duration(amount) * time.Hour
	default:
		duration = time.Duration(amount) * time.Second
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Waiting for %v", duration),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	// Wait with context
	select {
	case <-ctx.Done():
		output.Error = ctx.Err()
		return output, nil
	case <-time.After(duration):
		// Wait completed
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   "Wait completed",
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
	runtime.Register(NewWaitNode())
}
