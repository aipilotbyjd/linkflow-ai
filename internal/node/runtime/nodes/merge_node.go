// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// MergeNode implements data merging from multiple inputs
type MergeNode struct{}

// NewMergeNode creates a new Merge node
func NewMergeNode() *MergeNode {
	return &MergeNode{}
}

// GetType returns the node type
func (n *MergeNode) GetType() string {
	return "merge"
}

// GetMetadata returns node metadata
func (n *MergeNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "merge",
		Name:        "Merge",
		Description: "Merge data from multiple inputs",
		Category:    "core",
		Icon:        "git-merge",
		Color:       "#00BCD4",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "input1", Type: "any", Required: true, Description: "First input"},
			{Name: "input2", Type: "any", Required: true, Description: "Second input"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Merged data"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "mode", Type: "select", Default: "append", Description: "Merge mode", Options: []runtime.PropertyOption{
				{Label: "Append", Value: "append"},
				{Label: "Merge by Index", Value: "mergeByIndex"},
				{Label: "Merge by Key", Value: "mergeByKey"},
				{Label: "Keep Key Matches", Value: "keepKeyMatches"},
				{Label: "Remove Key Matches", Value: "removeKeyMatches"},
				{Label: "Combine", Value: "combine"},
				{Label: "Choose Branch", Value: "chooseBranch"},
				{Label: "Wait", Value: "wait"},
			}},
			{Name: "mergeKey", Type: "string", Description: "Field to use as merge key (for merge by key modes)"},
			{Name: "clashHandling", Type: "select", Default: "preferInput2", Description: "How to handle field conflicts", Options: []runtime.PropertyOption{
				{Label: "Prefer Input 1", Value: "preferInput1"},
				{Label: "Prefer Input 2", Value: "preferInput2"},
				{Label: "Merge Objects", Value: "merge"},
			}},
			{Name: "chooseBranchValue", Type: "select", Default: "input1", Description: "Which branch to output (for choose branch mode)", Options: []runtime.PropertyOption{
				{Label: "Input 1", Value: "input1"},
				{Label: "Input 2", Value: "input2"},
			}},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *MergeNode) Validate(config map[string]interface{}) error {
	return nil
}

// Execute executes the Merge node
func (n *MergeNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	mode := getStringConfig(input.NodeConfig, "mode", "append")
	mergeKey := getStringConfig(input.NodeConfig, "mergeKey", "id")
	clashHandling := getStringConfig(input.NodeConfig, "clashHandling", "preferInput2")
	
	// Get inputs
	input1 := getInputData(input.InputData, "input1")
	input2 := getInputData(input.InputData, "input2")
	
	var result interface{}
	var err error
	
	switch mode {
	case "append":
		result = n.appendItems(input1, input2)
	case "mergeByIndex":
		result = n.mergeByIndex(input1, input2, clashHandling)
	case "mergeByKey":
		result = n.mergeByKey(input1, input2, mergeKey, clashHandling)
	case "keepKeyMatches":
		result = n.keepKeyMatches(input1, input2, mergeKey)
	case "removeKeyMatches":
		result = n.removeKeyMatches(input1, input2, mergeKey)
	case "combine":
		result = n.combineAll(input1, input2)
	case "chooseBranch":
		branch := getStringConfig(input.NodeConfig, "chooseBranchValue", "input1")
		if branch == "input1" {
			result = input1
		} else {
			result = input2
		}
	case "wait":
		// Wait mode just passes through both inputs once both are available
		result = map[string]interface{}{
			"input1": input1,
			"input2": input2,
		}
	default:
		err = fmt.Errorf("unknown merge mode: %s", mode)
	}
	
	if err != nil {
		output.Error = err
		return output, nil
	}
	
	// Set output
	if m, ok := result.(map[string]interface{}); ok {
		output.Data = m
	} else if arr, ok := result.([]interface{}); ok {
		output.Data["items"] = arr
	} else {
		output.Data["result"] = result
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Merged using mode: %s", mode),
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

func getInputData(data map[string]interface{}, key string) interface{} {
	if v, ok := data[key]; ok {
		return v
	}
	return data
}

func (n *MergeNode) appendItems(input1, input2 interface{}) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	result := make([]interface{}, 0, len(arr1)+len(arr2))
	result = append(result, arr1...)
	result = append(result, arr2...)
	
	return result
}

func (n *MergeNode) mergeByIndex(input1, input2 interface{}, clashHandling string) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	maxLen := len(arr1)
	if len(arr2) > maxLen {
		maxLen = len(arr2)
	}
	
	result := make([]interface{}, maxLen)
	
	for i := 0; i < maxLen; i++ {
		var item1, item2 map[string]interface{}
		
		if i < len(arr1) {
			item1, _ = arr1[i].(map[string]interface{})
		}
		if i < len(arr2) {
			item2, _ = arr2[i].(map[string]interface{})
		}
		
		result[i] = mergeObjects(item1, item2, clashHandling)
	}
	
	return result
}

func (n *MergeNode) mergeByKey(input1, input2 interface{}, key, clashHandling string) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	// Index arr1 by key
	index1 := make(map[string]map[string]interface{})
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			keyVal := fmt.Sprintf("%v", m[key])
			index1[keyVal] = m
		}
	}
	
	// Merge with arr2
	result := make([]interface{}, 0)
	seen := make(map[string]bool)
	
	for _, item := range arr2 {
		if m, ok := item.(map[string]interface{}); ok {
			keyVal := fmt.Sprintf("%v", m[key])
			seen[keyVal] = true
			
			if item1, exists := index1[keyVal]; exists {
				result = append(result, mergeObjects(item1, m, clashHandling))
			} else {
				result = append(result, m)
			}
		}
	}
	
	// Add remaining items from arr1
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			keyVal := fmt.Sprintf("%v", m[key])
			if !seen[keyVal] {
				result = append(result, m)
			}
		}
	}
	
	return result
}

func (n *MergeNode) keepKeyMatches(input1, input2 interface{}, key string) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	// Build set of keys from arr2
	keys2 := make(map[string]bool)
	for _, item := range arr2 {
		if m, ok := item.(map[string]interface{}); ok {
			keys2[fmt.Sprintf("%v", m[key])] = true
		}
	}
	
	// Keep only items from arr1 that match
	result := make([]interface{}, 0)
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			if keys2[fmt.Sprintf("%v", m[key])] {
				result = append(result, m)
			}
		}
	}
	
	return result
}

func (n *MergeNode) removeKeyMatches(input1, input2 interface{}, key string) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	// Build set of keys from arr2
	keys2 := make(map[string]bool)
	for _, item := range arr2 {
		if m, ok := item.(map[string]interface{}); ok {
			keys2[fmt.Sprintf("%v", m[key])] = true
		}
	}
	
	// Keep only items from arr1 that DON'T match
	result := make([]interface{}, 0)
	for _, item := range arr1 {
		if m, ok := item.(map[string]interface{}); ok {
			if !keys2[fmt.Sprintf("%v", m[key])] {
				result = append(result, m)
			}
		}
	}
	
	return result
}

func (n *MergeNode) combineAll(input1, input2 interface{}) []interface{} {
	arr1 := toArray(input1)
	arr2 := toArray(input2)
	
	// Create all combinations
	result := make([]interface{}, 0, len(arr1)*len(arr2))
	
	for _, item1 := range arr1 {
		m1, _ := item1.(map[string]interface{})
		for _, item2 := range arr2 {
			m2, _ := item2.(map[string]interface{})
			combined := make(map[string]interface{})
			
			// Copy from m1
			for k, v := range m1 {
				combined[k] = v
			}
			// Copy from m2 with prefix to avoid conflicts
			for k, v := range m2 {
				if _, exists := combined[k]; exists {
					combined["input2_"+k] = v
				} else {
					combined[k] = v
				}
			}
			
			result = append(result, combined)
		}
	}
	
	return result
}

func toArray(v interface{}) []interface{} {
	if arr, ok := v.([]interface{}); ok {
		return arr
	}
	if m, ok := v.(map[string]interface{}); ok {
		if items, ok := m["items"].([]interface{}); ok {
			return items
		}
		return []interface{}{m}
	}
	if v != nil {
		return []interface{}{v}
	}
	return []interface{}{}
}

func mergeObjects(m1, m2 map[string]interface{}, clashHandling string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy m1
	for k, v := range m1 {
		result[k] = v
	}
	
	// Merge m2
	for k, v := range m2 {
		if existing, exists := result[k]; exists {
			switch clashHandling {
			case "preferInput1":
				// Keep existing
			case "preferInput2":
				result[k] = v
			case "merge":
				// Try to merge if both are objects
				if existingMap, ok := existing.(map[string]interface{}); ok {
					if newMap, ok := v.(map[string]interface{}); ok {
						result[k] = mergeObjects(existingMap, newMap, clashHandling)
						continue
					}
				}
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}
	
	return result
}

func init() {
	runtime.Register(NewMergeNode())
}
