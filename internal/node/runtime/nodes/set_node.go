// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
	"github.com/linkflow-ai/linkflow-ai/pkg/expression"
)

// SetNode implements data transformation/setting
type SetNode struct {
	parser *expression.Parser
}

// NewSetNode creates a new Set node
func NewSetNode() *SetNode {
	return &SetNode{
		parser: expression.NewParser(),
	}
}

// GetType returns the node type
func (n *SetNode) GetType() string {
	return "set"
}

// GetMetadata returns node metadata
func (n *SetNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "set",
		Name:        "Set",
		Description: "Set, modify, or create data fields",
		Category:    "core",
		Icon:        "edit",
		Color:       "#2196F3",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Modified data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "mode", Type: "select", Default: "manual", Description: "How to set values", Options: []runtime.PropertyOption{
				{Label: "Manual Mapping", Value: "manual"},
				{Label: "JSON", Value: "json"},
				{Label: "Expression", Value: "expression"},
			}},
			{Name: "values", Type: "json", Description: "Values to set (for manual mode)", Default: []interface{}{
				map[string]interface{}{
					"name":  "",
					"value": "",
					"type":  "string",
				},
			}},
			{Name: "jsonData", Type: "code", Description: "JSON data (for JSON mode)"},
			{Name: "keepOnlySet", Type: "boolean", Default: false, Description: "Keep only the fields being set"},
			{Name: "dotNotation", Type: "boolean", Default: true, Description: "Support dot notation for nested fields"},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *SetNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the Set node
func (n *SetNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	mode := getStringConfig(input.NodeConfig, "mode", "manual")
	keepOnlySet := getBoolConfig(input.NodeConfig, "keepOnlySet", false)
	useDotNotation := getBoolConfig(input.NodeConfig, "dotNotation", true)
	
	// Create expression context
	exprCtx := expression.NewContext()
	exprCtx.SetInput(input.InputData)
	if input.Context != nil {
		exprCtx.Execution.ID = input.Context.ExecutionID
		exprCtx.Execution.Mode = input.Context.Mode
		exprCtx.Env = input.Context.Env
		exprCtx.Variables = input.Context.Variables
	}
	
	// Start with existing data or empty
	var result map[string]interface{}
	if keepOnlySet {
		result = make(map[string]interface{})
	} else {
		result = copyMap(input.InputData)
	}
	
	switch mode {
	case "manual":
		values, _ := input.NodeConfig["values"].([]interface{})
		for _, v := range values {
			valueMap, ok := v.(map[string]interface{})
			if !ok {
				continue
			}
			
			name := fmt.Sprintf("%v", valueMap["name"])
			value := valueMap["value"]
			valueType := getStringConfig(valueMap, "type", "string")
			
			// Evaluate expressions in value
			if strVal, ok := value.(string); ok && strings.Contains(strVal, "{{") {
				evaluated, err := n.parser.Evaluate(strVal, exprCtx)
				if err == nil {
					value = evaluated
				}
			}
			
			// Convert type
			value = convertType(value, valueType)
			
			// Set value
			if useDotNotation && strings.Contains(name, ".") {
				setNestedValue(result, name, value)
			} else {
				result[name] = value
			}
		}
		
	case "json":
		jsonData := getStringConfig(input.NodeConfig, "jsonData", "{}")
		
		// Evaluate expressions in JSON
		if strings.Contains(jsonData, "{{") {
			evaluated, err := n.parser.Evaluate(jsonData, exprCtx)
			if err == nil {
				if s, ok := evaluated.(string); ok {
					jsonData = s
				}
			}
		}
		
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &parsed); err != nil {
			output.Error = fmt.Errorf("invalid JSON: %w", err)
			return output, nil
		}
		
		// Merge with result
		for k, v := range parsed {
			result[k] = v
		}
		
	case "expression":
		// Expression mode - evaluate a single expression to get the entire output
		expr := getStringConfig(input.NodeConfig, "expression", "")
		if expr != "" {
			evaluated, err := n.parser.Evaluate(expr, exprCtx)
			if err == nil {
				if m, ok := evaluated.(map[string]interface{}); ok {
					result = m
				}
			}
		}
	}
	
	output.Data = result
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Set %d fields", len(result)),
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

func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := m
	
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}
		if nested, ok := current[part].(map[string]interface{}); ok {
			current = nested
		} else {
			// Cannot set nested value on non-object
			return
		}
	}
	
	current[parts[len(parts)-1]] = value
}

func convertType(value interface{}, targetType string) interface{} {
	switch targetType {
	case "string":
		return fmt.Sprintf("%v", value)
	case "number":
		return toNumber(value)
	case "boolean":
		return toBool(value)
	case "json":
		if s, ok := value.(string); ok {
			var parsed interface{}
			if err := json.Unmarshal([]byte(s), &parsed); err == nil {
				return parsed
			}
		}
		return value
	default:
		return value
	}
}

func getBoolConfig(config map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := config[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func init() {
	runtime.Register(NewSetNode())
}
