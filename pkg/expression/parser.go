// Package expression provides expression parsing and evaluation for workflow data
package expression

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Expression patterns
var (
	// {{$node.nodeName.data.field}} or {{$json.field}} or {{$env.VAR}}
	expressionPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)
	
	// $node.nodeName.data.field
	nodeDataPattern = regexp.MustCompile(`^\$node\.([^.]+)\.([^.]+)(?:\.(.+))?$`)
	
	// $json.field or $json["field"]
	jsonPattern = regexp.MustCompile(`^\$json(?:\.|\[)(.+)$`)
	
	// $env.VARIABLE
	envPattern = regexp.MustCompile(`^\$env\.(.+)$`)
	
	// $input.item or $input.all
	inputPattern = regexp.MustCompile(`^\$input\.(.+)$`)
	
	// $now, $today, $timestamp
	datePattern = regexp.MustCompile(`^\$(now|today|timestamp)$`)
	
	// $execution.id, $execution.mode
	executionPattern = regexp.MustCompile(`^\$execution\.(.+)$`)
	
	// $workflow.id, $workflow.name
	workflowPattern = regexp.MustCompile(`^\$workflow\.(.+)$`)
	
	// Function calls: $func.uppercase(value)
	funcPattern = regexp.MustCompile(`^\$func\.(\w+)\((.+)\)$`)
)

// Context holds the evaluation context
type Context struct {
	NodeOutputs   map[string]map[string]interface{} // nodeID -> output data
	Input         interface{}                        // Current input data
	Env           map[string]string                  // Environment variables
	Variables     map[string]interface{}             // User variables
	Execution     ExecutionContext                   // Execution metadata
	Workflow      WorkflowContext                    // Workflow metadata
}

// ExecutionContext holds execution metadata
type ExecutionContext struct {
	ID        string
	Mode      string // manual, webhook, schedule
	Timestamp time.Time
}

// WorkflowContext holds workflow metadata
type WorkflowContext struct {
	ID          string
	Name        string
	Active      bool
}

// NewContext creates a new evaluation context
func NewContext() *Context {
	return &Context{
		NodeOutputs: make(map[string]map[string]interface{}),
		Env:         make(map[string]string),
		Variables:   make(map[string]interface{}),
		Execution: ExecutionContext{
			Timestamp: time.Now(),
		},
	}
}

// SetNodeOutput sets the output for a node
func (c *Context) SetNodeOutput(nodeID string, output map[string]interface{}) {
	c.NodeOutputs[nodeID] = output
}

// SetInput sets the current input data
func (c *Context) SetInput(input interface{}) {
	c.Input = input
}

// Parser handles expression parsing and evaluation
type Parser struct {
	functions map[string]Function
}

// Function represents a built-in function
type Function func(args ...interface{}) (interface{}, error)

// NewParser creates a new expression parser
func NewParser() *Parser {
	p := &Parser{
		functions: make(map[string]Function),
	}
	p.registerBuiltinFunctions()
	return p
}

// registerBuiltinFunctions registers all built-in functions
func (p *Parser) registerBuiltinFunctions() {
	// String functions
	p.functions["uppercase"] = funcUppercase
	p.functions["lowercase"] = funcLowercase
	p.functions["trim"] = funcTrim
	p.functions["length"] = funcLength
	p.functions["substring"] = funcSubstring
	p.functions["replace"] = funcReplace
	p.functions["split"] = funcSplit
	p.functions["join"] = funcJoin
	p.functions["contains"] = funcContains
	p.functions["startsWith"] = funcStartsWith
	p.functions["endsWith"] = funcEndsWith
	
	// Number functions
	p.functions["round"] = funcRound
	p.functions["floor"] = funcFloor
	p.functions["ceil"] = funcCeil
	p.functions["abs"] = funcAbs
	p.functions["min"] = funcMin
	p.functions["max"] = funcMax
	p.functions["sum"] = funcSum
	p.functions["avg"] = funcAvg
	
	// Date functions
	p.functions["now"] = funcNow
	p.functions["formatDate"] = funcFormatDate
	p.functions["parseDate"] = funcParseDate
	p.functions["addDays"] = funcAddDays
	p.functions["addHours"] = funcAddHours
	
	// JSON functions
	p.functions["toJson"] = funcToJSON
	p.functions["fromJson"] = funcFromJSON
	p.functions["keys"] = funcKeys
	p.functions["values"] = funcValues
	
	// Array functions
	p.functions["first"] = funcFirst
	p.functions["last"] = funcLast
	p.functions["count"] = funcCount
	p.functions["reverse"] = funcReverse
	p.functions["sort"] = funcSort
	p.functions["unique"] = funcUnique
	p.functions["filter"] = funcFilter
	p.functions["map"] = funcMap
	
	// Type functions
	p.functions["toString"] = funcToString
	p.functions["toNumber"] = funcToNumber
	p.functions["toBoolean"] = funcToBoolean
	p.functions["isNull"] = funcIsNull
	p.functions["isEmpty"] = funcIsEmpty
	p.functions["typeof"] = funcTypeof
	
	// Utility functions
	p.functions["if"] = funcIf
	p.functions["default"] = funcDefault
	p.functions["uuid"] = funcUUID
	p.functions["base64Encode"] = funcBase64Encode
	p.functions["base64Decode"] = funcBase64Decode
	p.functions["hash"] = funcHash
}

// Evaluate evaluates an expression string with the given context
func (p *Parser) Evaluate(expr string, ctx *Context) (interface{}, error) {
	// Check if it's a simple expression (no template syntax)
	if !strings.Contains(expr, "{{") {
		return expr, nil
	}
	
	// Find and replace all expressions
	result := expressionPattern.ReplaceAllStringFunc(expr, func(match string) string {
		// Extract expression without {{ }}
		inner := strings.TrimPrefix(strings.TrimSuffix(match, "}}"), "{{")
		inner = strings.TrimSpace(inner)
		
		val, err := p.evaluateExpression(inner, ctx)
		if err != nil {
			return match // Return original on error
		}
		
		return toString(val)
	})
	
	// If the entire string was a single expression, return the typed value
	if strings.HasPrefix(strings.TrimSpace(expr), "{{") && strings.HasSuffix(strings.TrimSpace(expr), "}}") {
		inner := strings.TrimPrefix(strings.TrimSuffix(strings.TrimSpace(expr), "}}"), "{{")
		inner = strings.TrimSpace(inner)
		return p.evaluateExpression(inner, ctx)
	}
	
	return result, nil
}

// EvaluateTemplate evaluates all expressions in a map
func (p *Parser) EvaluateTemplate(data map[string]interface{}, ctx *Context) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for key, value := range data {
		evaluated, err := p.evaluateValue(value, ctx)
		if err != nil {
			return nil, fmt.Errorf("error evaluating %s: %w", key, err)
		}
		result[key] = evaluated
	}
	
	return result, nil
}

func (p *Parser) evaluateValue(value interface{}, ctx *Context) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return p.Evaluate(v, ctx)
	case map[string]interface{}:
		return p.EvaluateTemplate(v, ctx)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			evaluated, err := p.evaluateValue(item, ctx)
			if err != nil {
				return nil, err
			}
			result[i] = evaluated
		}
		return result, nil
	default:
		return value, nil
	}
}

// evaluateExpression evaluates a single expression
func (p *Parser) evaluateExpression(expr string, ctx *Context) (interface{}, error) {
	expr = strings.TrimSpace(expr)
	
	// Check for function call
	if matches := funcPattern.FindStringSubmatch(expr); len(matches) == 3 {
		return p.evaluateFunction(matches[1], matches[2], ctx)
	}
	
	// Check for node data reference
	if matches := nodeDataPattern.FindStringSubmatch(expr); len(matches) >= 3 {
		return p.evaluateNodeData(matches, ctx)
	}
	
	// Check for $json reference
	if matches := jsonPattern.FindStringSubmatch(expr); len(matches) == 2 {
		return p.evaluateJSONPath(matches[1], ctx.Input)
	}
	
	// Check for $env reference
	if matches := envPattern.FindStringSubmatch(expr); len(matches) == 2 {
		return ctx.Env[matches[1]], nil
	}
	
	// Check for $input reference
	if matches := inputPattern.FindStringSubmatch(expr); len(matches) == 2 {
		return p.evaluateInput(matches[1], ctx)
	}
	
	// Check for date shortcuts
	if matches := datePattern.FindStringSubmatch(expr); len(matches) == 2 {
		return p.evaluateDate(matches[1])
	}
	
	// Check for $execution reference
	if matches := executionPattern.FindStringSubmatch(expr); len(matches) == 2 {
		return p.evaluateExecution(matches[1], ctx)
	}
	
	// Check for $workflow reference
	if matches := workflowPattern.FindStringSubmatch(expr); len(matches) == 2 {
		return p.evaluateWorkflow(matches[1], ctx)
	}
	
	// Check for variable reference
	if strings.HasPrefix(expr, "$vars.") {
		varName := strings.TrimPrefix(expr, "$vars.")
		return ctx.Variables[varName], nil
	}
	
	// Return as literal
	return expr, nil
}

func (p *Parser) evaluateNodeData(matches []string, ctx *Context) (interface{}, error) {
	nodeID := matches[1]
	dataType := matches[2] // "data", "json", "binary"
	path := ""
	if len(matches) > 3 {
		path = matches[3]
	}
	
	nodeOutput, exists := ctx.NodeOutputs[nodeID]
	if !exists {
		return nil, fmt.Errorf("node '%s' not found in context", nodeID)
	}
	
	var data interface{}
	switch dataType {
	case "data", "json":
		data = nodeOutput
	default:
		data = nodeOutput[dataType]
	}
	
	if path == "" {
		return data, nil
	}
	
	return getValueByPath(data, path)
}

func (p *Parser) evaluateJSONPath(path string, data interface{}) (interface{}, error) {
	// Handle bracket notation: ["field"] -> field
	path = strings.ReplaceAll(path, `["`, ".")
	path = strings.ReplaceAll(path, `"]`, "")
	path = strings.TrimPrefix(path, ".")
	
	return getValueByPath(data, path)
}

func (p *Parser) evaluateInput(field string, ctx *Context) (interface{}, error) {
	switch field {
	case "item":
		return ctx.Input, nil
	case "all":
		return ctx.Input, nil
	default:
		return getValueByPath(ctx.Input, field)
	}
}

func (p *Parser) evaluateDate(dateType string) (interface{}, error) {
	now := time.Now()
	switch dateType {
	case "now":
		return now.Format(time.RFC3339), nil
	case "today":
		return now.Format("2006-01-02"), nil
	case "timestamp":
		return now.Unix(), nil
	}
	return nil, fmt.Errorf("unknown date type: %s", dateType)
}

func (p *Parser) evaluateExecution(field string, ctx *Context) (interface{}, error) {
	switch field {
	case "id":
		return ctx.Execution.ID, nil
	case "mode":
		return ctx.Execution.Mode, nil
	case "timestamp":
		return ctx.Execution.Timestamp.Format(time.RFC3339), nil
	}
	return nil, fmt.Errorf("unknown execution field: %s", field)
}

func (p *Parser) evaluateWorkflow(field string, ctx *Context) (interface{}, error) {
	switch field {
	case "id":
		return ctx.Workflow.ID, nil
	case "name":
		return ctx.Workflow.Name, nil
	case "active":
		return ctx.Workflow.Active, nil
	}
	return nil, fmt.Errorf("unknown workflow field: %s", field)
}

func (p *Parser) evaluateFunction(name, argsStr string, ctx *Context) (interface{}, error) {
	fn, exists := p.functions[name]
	if !exists {
		return nil, fmt.Errorf("unknown function: %s", name)
	}
	
	// Parse arguments
	args, err := p.parseArguments(argsStr, ctx)
	if err != nil {
		return nil, err
	}
	
	return fn(args...)
}

func (p *Parser) parseArguments(argsStr string, ctx *Context) ([]interface{}, error) {
	if argsStr == "" {
		return []interface{}{}, nil
	}
	
	// Simple argument splitting (doesn't handle nested functions well)
	parts := splitArguments(argsStr)
	args := make([]interface{}, len(parts))
	
	for i, part := range parts {
		part = strings.TrimSpace(part)
		
		// Check if it's an expression
		if strings.HasPrefix(part, "$") {
			val, err := p.evaluateExpression(part, ctx)
			if err != nil {
				return nil, err
			}
			args[i] = val
		} else if strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`) {
			// String literal
			args[i] = strings.Trim(part, `"`)
		} else if strings.HasPrefix(part, `'`) && strings.HasSuffix(part, `'`) {
			// String literal (single quotes)
			args[i] = strings.Trim(part, `'`)
		} else if num, err := strconv.ParseFloat(part, 64); err == nil {
			// Number
			args[i] = num
		} else if part == "true" {
			args[i] = true
		} else if part == "false" {
			args[i] = false
		} else if part == "null" {
			args[i] = nil
		} else {
			args[i] = part
		}
	}
	
	return args, nil
}

// Helper functions

func getValueByPath(data interface{}, path string) (interface{}, error) {
	if path == "" {
		return data, nil
	}
	
	parts := strings.Split(path, ".")
	current := data
	
	for _, part := range parts {
		// Handle array index: field[0]
		if idx := strings.Index(part, "["); idx != -1 {
			fieldName := part[:idx]
			indexStr := strings.TrimSuffix(part[idx+1:], "]")
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", indexStr)
			}
			
			// Get field first if fieldName is not empty
			if fieldName != "" {
				current, err = getField(current, fieldName)
				if err != nil {
					return nil, err
				}
			}
			
			// Then get array element
			arr, ok := current.([]interface{})
			if !ok {
				return nil, fmt.Errorf("expected array at %s", part)
			}
			if index < 0 || index >= len(arr) {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}
			current = arr[index]
		} else {
			var err error
			current, err = getField(current, part)
			if err != nil {
				return nil, err
			}
		}
	}
	
	return current, nil
}

func getField(data interface{}, field string) (interface{}, error) {
	switch d := data.(type) {
	case map[string]interface{}:
		val, exists := d[field]
		if !exists {
			return nil, fmt.Errorf("field '%s' not found", field)
		}
		return val, nil
	case map[string]string:
		val, exists := d[field]
		if !exists {
			return nil, fmt.Errorf("field '%s' not found", field)
		}
		return val, nil
	default:
		return nil, fmt.Errorf("cannot get field '%s' from %T", field, data)
	}
}

func splitArguments(s string) []string {
	var result []string
	var current strings.Builder
	depth := 0
	inString := false
	stringChar := byte(0)
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		
		if inString {
			current.WriteByte(c)
			if c == stringChar && (i == 0 || s[i-1] != '\\') {
				inString = false
			}
			continue
		}
		
		if c == '"' || c == '\'' {
			inString = true
			stringChar = c
			current.WriteByte(c)
			continue
		}
		
		if c == '(' || c == '[' || c == '{' {
			depth++
			current.WriteByte(c)
			continue
		}
		
		if c == ')' || c == ']' || c == '}' {
			depth--
			current.WriteByte(c)
			continue
		}
		
		if c == ',' && depth == 0 {
			result = append(result, current.String())
			current.Reset()
			continue
		}
		
		current.WriteByte(c)
	}
	
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	
	return result
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}
