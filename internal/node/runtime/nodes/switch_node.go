// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// SwitchNode implements multi-branch routing
type SwitchNode struct{}

// NewSwitchNode creates a new Switch node
func NewSwitchNode() *SwitchNode {
	return &SwitchNode{}
}

// GetType returns the node type
func (n *SwitchNode) GetType() string {
	return "switch"
}

// GetMetadata returns node metadata
func (n *SwitchNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "switch",
		Name:        "Switch",
		Description: "Route items to different outputs based on rules",
		Category:    "core",
		Icon:        "shuffle",
		Color:       "#FF9800",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "output0", Type: "any", Description: "First route"},
			{Name: "output1", Type: "any", Description: "Second route"},
			{Name: "output2", Type: "any", Description: "Third route"},
			{Name: "output3", Type: "any", Description: "Fourth route"},
			{Name: "fallback", Type: "any", Description: "Default route (no match)"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "mode", Type: "select", Default: "rules", Description: "Switch mode", Options: []runtime.PropertyOption{
				{Label: "Rules", Value: "rules"},
				{Label: "Expression", Value: "expression"},
			}},
			{Name: "dataType", Type: "select", Default: "string", Description: "Data type to compare", Options: []runtime.PropertyOption{
				{Label: "String", Value: "string"},
				{Label: "Number", Value: "number"},
				{Label: "Boolean", Value: "boolean"},
			}},
			{Name: "rules", Type: "json", Description: "Routing rules", Default: []interface{}{
				map[string]interface{}{
					"output":   0,
					"field":    "",
					"operator": "equals",
					"value":    "",
				},
			}},
			{Name: "fallbackOutput", Type: "select", Default: "fallback", Description: "Output for non-matching items", Options: []runtime.PropertyOption{
				{Label: "Fallback output", Value: "fallback"},
				{Label: "None (drop)", Value: "none"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *SwitchNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the Switch node
func (n *SwitchNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	mode := getStringConfig(input.NodeConfig, "mode", "rules")
	fallbackOutput := getStringConfig(input.NodeConfig, "fallbackOutput", "fallback")
	
	var matchedOutput string
	
	if mode == "rules" {
		rules, _ := input.NodeConfig["rules"].([]interface{})
		matchedOutput = n.evaluateRules(rules, input.InputData)
	} else {
		// Expression mode - use field value directly as output selector
		expression := getStringConfig(input.NodeConfig, "expression", "")
		fieldValue := getFieldValue(input.InputData, expression)
		matchedOutput = fmt.Sprintf("output%v", fieldValue)
	}
	
	// Set output
	if matchedOutput != "" {
		output.Data["_output"] = matchedOutput
		output.Data[matchedOutput] = input.InputData
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   fmt.Sprintf("Routed to: %s", matchedOutput),
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
	} else if fallbackOutput != "none" {
		output.Data["_output"] = fallbackOutput
		output.Data[fallbackOutput] = input.InputData
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   "No rule matched, routed to fallback",
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
	} else {
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   "No rule matched, item dropped",
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
	}
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func (n *SwitchNode) evaluateRules(rules []interface{}, data map[string]interface{}) string {
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		
		outputNum := ruleMap["output"]
		field := fmt.Sprintf("%v", ruleMap["field"])
		operator := fmt.Sprintf("%v", ruleMap["operator"])
		value := ruleMap["value"]
		
		fieldValue := getFieldValue(data, field)
		
		if evaluateCondition(fieldValue, operator, value) {
			return fmt.Sprintf("output%v", outputNum)
		}
	}
	
	return ""
}

func init() {
	runtime.Register(NewSwitchNode())
}
