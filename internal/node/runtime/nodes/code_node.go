// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/expression"
)

// CodeNode implements custom code execution
type CodeNode struct {
	parser *expression.Parser
}

// NewCodeNode creates a new Code node
func NewCodeNode() *CodeNode {
	return &CodeNode{
		parser: expression.NewParser(),
	}
}

// GetType returns the node type
func (n *CodeNode) GetType() string {
	return "code"
}

// GetMetadata returns node metadata
func (n *CodeNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "code",
		Name:        "Code",
		Description: "Execute custom code to transform data",
		Category:    "core",
		Icon:        "code",
		Color:       "#607D8B",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Transformed data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "language", Type: "select", Default: "expression", Description: "Language", Options: []runtime.PropertyOption{
				{Label: "Expression", Value: "expression"},
				{Label: "JSON Transform", Value: "json"},
			}},
			{Name: "code", Type: "code", Required: true, Description: "Code to execute"},
			{Name: "mode", Type: "select", Default: "runOnceForAllItems", Description: "Execution mode", Options: []runtime.PropertyOption{
				{Label: "Run Once for All Items", Value: "runOnceForAllItems"},
				{Label: "Run Once for Each Item", Value: "runOnceForEachItem"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *CodeNode) Validate(config map[string]interface{}) error {
	if _, ok := config["code"]; !ok {
		return fmt.Errorf("code is required")
	}
	return nil
}

// Execute executes the Code node
func (n *CodeNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	language := getStringConfig(input.NodeConfig, "language", "expression")
	code := getStringConfig(input.NodeConfig, "code", "")
	mode := getStringConfig(input.NodeConfig, "mode", "runOnceForAllItems")
	
	// Create expression context
	exprCtx := expression.NewContext()
	exprCtx.SetInput(input.InputData)
	if input.Context != nil {
		exprCtx.Execution.ID = input.Context.ExecutionID
		exprCtx.Execution.Mode = input.Context.Mode
		exprCtx.Env = input.Context.Env
		exprCtx.Variables = input.Context.Variables
	}
	
	switch language {
	case "expression":
		result, err := n.executeExpression(code, exprCtx, input.InputData, mode)
		if err != nil {
			output.Error = err
			return output, nil
		}
		output.Data = result
		
	case "json":
		result, err := n.executeJSONTransform(code, exprCtx, input.InputData)
		if err != nil {
			output.Error = err
			return output, nil
		}
		output.Data = result
		
	default:
		output.Error = fmt.Errorf("unsupported language: %s", language)
		return output, nil
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   "Code executed successfully",
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

func (n *CodeNode) executeExpression(code string, ctx *expression.Context, inputData map[string]interface{}, mode string) (map[string]interface{}, error) {
	if mode == "runOnceForEachItem" {
		// Process each item separately
		items, ok := inputData["items"].([]interface{})
		if !ok {
			items = []interface{}{inputData}
		}
		
		results := make([]interface{}, 0, len(items))
		for _, item := range items {
			ctx.SetInput(item)
			result, err := n.parser.Evaluate(code, ctx)
			if err != nil {
				return nil, fmt.Errorf("error processing item: %w", err)
			}
			results = append(results, result)
		}
		
		return map[string]interface{}{"items": results}, nil
	}
	
	// Run once for all items
	result, err := n.parser.Evaluate(code, ctx)
	if err != nil {
		return nil, err
	}
	
	// Convert result to map
	switch v := result.(type) {
	case map[string]interface{}:
		return v, nil
	default:
		return map[string]interface{}{"result": result}, nil
	}
}

func (n *CodeNode) executeJSONTransform(code string, ctx *expression.Context, inputData map[string]interface{}) (map[string]interface{}, error) {
	// First evaluate any expressions in the JSON
	evaluated, err := n.parser.Evaluate(code, ctx)
	if err != nil {
		return nil, err
	}
	
	// Parse as JSON
	var jsonStr string
	switch v := evaluated.(type) {
	case string:
		jsonStr = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		jsonStr = string(b)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON output: %w", err)
	}
	
	return result, nil
}

func init() {
	runtime.Register(NewCodeNode())
}
