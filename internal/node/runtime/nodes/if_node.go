// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// IFNode implements conditional branching
type IFNode struct{}

// NewIFNode creates a new IF node
func NewIFNode() *IFNode {
	return &IFNode{}
}

// GetType returns the node type
func (n *IFNode) GetType() string {
	return "if"
}

// GetMetadata returns node metadata
func (n *IFNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "if",
		Name:        "IF",
		Description: "Route items based on conditions",
		Category:    "core",
		Icon:        "git-branch",
		Color:       "#FF9800",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Required: true, Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "true", Type: "any", Description: "Items matching condition"},
			{Name: "false", Type: "any", Description: "Items not matching condition"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "conditions", Type: "json", Required: true, Description: "Conditions to evaluate", Default: []interface{}{
				map[string]interface{}{
					"field":    "",
					"operator": "equals",
					"value":    "",
				},
			}},
			{Name: "combineConditions", Type: "select", Default: "and", Description: "How to combine multiple conditions", Options: []runtime.PropertyOption{
				{Label: "AND (all must match)", Value: "and"},
				{Label: "OR (any must match)", Value: "or"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *IFNode) Validate(config map[string]interface{}) error {
	conditions, ok := config["conditions"]
	if !ok {
		return fmt.Errorf("conditions are required")
	}
	
	condList, ok := conditions.([]interface{})
	if !ok || len(condList) == 0 {
		return fmt.Errorf("at least one condition is required")
	}
	
	return nil
}

// Execute executes the IF node
func (n *IFNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	// Get conditions
	conditions, _ := input.NodeConfig["conditions"].([]interface{})
	combineMode := getStringConfig(input.NodeConfig, "combineConditions", "and")
	
	// Evaluate conditions
	result := n.evaluateConditions(conditions, input.InputData, combineMode)
	
	// Set output based on result
	if result {
		output.Data["_output"] = "true"
		output.Data["true"] = input.InputData
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   "Condition evaluated to TRUE",
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
	} else {
		output.Data["_output"] = "false"
		output.Data["false"] = input.InputData
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "info",
			Message:   "Condition evaluated to FALSE",
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

func (n *IFNode) evaluateConditions(conditions []interface{}, data map[string]interface{}, combineMode string) bool {
	if len(conditions) == 0 {
		return true
	}
	
	results := make([]bool, len(conditions))
	
	for i, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			results[i] = false
			continue
		}
		
		field := fmt.Sprintf("%v", condMap["field"])
		operator := fmt.Sprintf("%v", condMap["operator"])
		value := condMap["value"]
		
		// Get field value from data
		fieldValue := getFieldValue(data, field)
		
		// Evaluate condition
		results[i] = evaluateCondition(fieldValue, operator, value)
	}
	
	// Combine results
	if combineMode == "or" {
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	}
	
	// AND mode
	for _, r := range results {
		if !r {
			return false
		}
	}
	return true
}

func getFieldValue(data map[string]interface{}, field string) interface{} {
	if field == "" {
		return data
	}
	
	parts := strings.Split(field, ".")
	var current interface{} = data
	
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return nil
		}
	}
	
	return current
}

func evaluateCondition(fieldValue interface{}, operator string, compareValue interface{}) bool {
	switch operator {
	case "equals", "equal", "==":
		return compareEqual(fieldValue, compareValue)
	case "notEquals", "notEqual", "!=":
		return !compareEqual(fieldValue, compareValue)
	case "contains":
		return compareContains(fieldValue, compareValue)
	case "notContains":
		return !compareContains(fieldValue, compareValue)
	case "startsWith":
		return compareStartsWith(fieldValue, compareValue)
	case "endsWith":
		return compareEndsWith(fieldValue, compareValue)
	case "greaterThan", ">":
		return compareGreater(fieldValue, compareValue)
	case "greaterThanOrEqual", ">=":
		return compareGreater(fieldValue, compareValue) || compareEqual(fieldValue, compareValue)
	case "lessThan", "<":
		return compareLess(fieldValue, compareValue)
	case "lessThanOrEqual", "<=":
		return compareLess(fieldValue, compareValue) || compareEqual(fieldValue, compareValue)
	case "isEmpty":
		return isEmpty(fieldValue)
	case "isNotEmpty":
		return !isEmpty(fieldValue)
	case "isNull":
		return fieldValue == nil
	case "isNotNull":
		return fieldValue != nil
	case "isTrue":
		return toBool(fieldValue)
	case "isFalse":
		return !toBool(fieldValue)
	case "regex", "matches":
		return matchesRegex(fieldValue, compareValue)
	case "in":
		return isIn(fieldValue, compareValue)
	case "notIn":
		return !isIn(fieldValue, compareValue)
	default:
		return false
	}
}

func compareEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	
	// Convert to string for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	
	return aStr == bStr
}

func compareContains(a, b interface{}) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.Contains(aStr, bStr)
}

func compareStartsWith(a, b interface{}) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.HasPrefix(aStr, bStr)
}

func compareEndsWith(a, b interface{}) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return strings.HasSuffix(aStr, bStr)
}

func compareGreater(a, b interface{}) bool {
	aNum := toNumber(a)
	bNum := toNumber(b)
	return aNum > bNum
}

func compareLess(a, b interface{}) bool {
	aNum := toNumber(a)
	bNum := toNumber(b)
	return aNum < bNum
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	
	switch val := v.(type) {
	case string:
		return val == ""
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	
	return false
}

func toBool(v interface{}) bool {
	if v == nil {
		return false
	}
	
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "false" && val != "0"
	case float64:
		return val != 0
	case int:
		return val != 0
	}
	
	return true
}

func toNumber(v interface{}) float64 {
	if v == nil {
		return 0
	}
	
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		n, _ := strconv.ParseFloat(val, 64)
		return n
	}
	
	return 0
}

func matchesRegex(v, pattern interface{}) bool {
	vStr := fmt.Sprintf("%v", v)
	patternStr := fmt.Sprintf("%v", pattern)
	
	re, err := regexp.Compile(patternStr)
	if err != nil {
		return false
	}
	
	return re.MatchString(vStr)
}

func isIn(v, list interface{}) bool {
	vStr := fmt.Sprintf("%v", v)
	
	listVal := reflect.ValueOf(list)
	if listVal.Kind() != reflect.Slice {
		// Try comma-separated string
		if s, ok := list.(string); ok {
			parts := strings.Split(s, ",")
			for _, p := range parts {
				if strings.TrimSpace(p) == vStr {
					return true
				}
			}
		}
		return false
	}
	
	for i := 0; i < listVal.Len(); i++ {
		if fmt.Sprintf("%v", listVal.Index(i).Interface()) == vStr {
			return true
		}
	}
	
	return false
}

func init() {
	runtime.Register(NewIFNode())
}
