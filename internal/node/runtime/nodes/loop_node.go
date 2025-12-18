// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// LoopNode implements iteration over items
type LoopNode struct{}

// NewLoopNode creates a new Loop node
func NewLoopNode() *LoopNode {
	return &LoopNode{}
}

// GetType returns the node type
func (n *LoopNode) GetType() string {
	return "loop"
}

// GetMetadata returns node metadata
func (n *LoopNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "loop",
		Name:        "Loop",
		Description: "Iterate over items in an array",
		Category:    "core",
		Icon:        "repeat",
		Color:       "#673AB7",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Array to iterate"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "loop", Type: "any", Description: "Current item (for loop body)"},
			{Name: "done", Type: "any", Description: "All items after loop completes"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "items", Type: "string", Description: "Path to array field (leave empty for root array)"},
			{Name: "batchSize", Type: "number", Default: 1, Description: "Number of items per batch"},
			{Name: "pauseBetweenBatches", Type: "number", Default: 0, Description: "Pause in ms between batches"},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *LoopNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the Loop node
func (n *LoopNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	itemsPath := getStringConfig(input.NodeConfig, "items", "")
	batchSize := getIntConfig(input.NodeConfig, "batchSize", 1)
	if batchSize < 1 {
		batchSize = 1
	}
	
	// Get array to iterate
	var items []interface{}
	
	if itemsPath == "" {
		// Check if input is already an array
		if arr, ok := input.InputData["items"].([]interface{}); ok {
			items = arr
		} else {
			// Wrap single item
			items = []interface{}{input.InputData}
		}
	} else {
		// Get from path
		value := getFieldValue(input.InputData, itemsPath)
		if arr, ok := value.([]interface{}); ok {
			items = arr
		} else {
			output.Error = fmt.Errorf("field '%s' is not an array", itemsPath)
			return output, nil
		}
	}
	
	if len(items) == 0 {
		output.Data["done"] = []interface{}{}
		output.Data["_output"] = "done"
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   "Empty array, nothing to iterate",
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
		return output, nil
	}
	
	// Create batched items
	var batches [][]interface{}
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	
	// Store iteration state
	output.Data["_loopState"] = map[string]interface{}{
		"items":       items,
		"batches":     batches,
		"totalItems":  len(items),
		"totalBatches": len(batches),
		"currentBatch": 0,
	}
	
	// Output first batch
	if batchSize == 1 {
		output.Data["loop"] = map[string]interface{}{
			"item":  items[0],
			"index": 0,
			"first": true,
			"last":  len(items) == 1,
		}
	} else {
		output.Data["loop"] = map[string]interface{}{
			"items":      batches[0],
			"batchIndex": 0,
			"first":      true,
			"last":       len(batches) == 1,
		}
	}
	
	output.Data["_output"] = "loop"
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Starting loop with %d items in %d batches", len(items), len(batches)),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
		ItemsIn:    len(items),
	}
	
	return output, nil
}

func init() {
	runtime.Register(NewLoopNode())
}

// SplitInBatchesNode splits items into batches
type SplitInBatchesNode struct{}

// NewSplitInBatchesNode creates a new SplitInBatches node
func NewSplitInBatchesNode() *SplitInBatchesNode {
	return &SplitInBatchesNode{}
}

// GetType returns the node type
func (n *SplitInBatchesNode) GetType() string {
	return "split_in_batches"
}

// GetMetadata returns node metadata
func (n *SplitInBatchesNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "split_in_batches",
		Name:        "Split In Batches",
		Description: "Split items into smaller batches for processing",
		Category:    "core",
		Icon:        "layers",
		Color:       "#673AB7",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Items to split"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Batched items"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "batchSize", Type: "number", Default: 10, Required: true, Description: "Number of items per batch"},
			{Name: "options", Type: "json", Description: "Additional options"},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *SplitInBatchesNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the SplitInBatches node
func (n *SplitInBatchesNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	batchSize := getIntConfig(input.NodeConfig, "batchSize", 10)
	if batchSize < 1 {
		batchSize = 1
	}
	
	// Get items
	var items []interface{}
	if arr, ok := input.InputData["items"].([]interface{}); ok {
		items = arr
	} else {
		items = []interface{}{input.InputData}
	}
	
	// Split into batches
	var batches []interface{}
	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	
	output.Data["batches"] = batches
	output.Data["totalItems"] = len(items)
	output.Data["totalBatches"] = len(batches)
	output.Data["batchSize"] = batchSize
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Split %d items into %d batches of %d", len(items), len(batches), batchSize),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
		ItemsIn:    len(items),
		ItemsOut:   len(batches),
	}
	
	return output, nil
}

func init() {
	runtime.Register(NewSplitInBatchesNode())
}
