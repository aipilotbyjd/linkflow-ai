package unit_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NodeType string

const (
	NodeTypeTrigger   NodeType = "trigger"
	NodeTypeAction    NodeType = "action"
	NodeTypeCondition NodeType = "condition"
	NodeTypeLoop      NodeType = "loop"
	NodeTypeTransform NodeType = "transform"
)

type NodeDefinition struct {
	ID          string
	Name        string
	Description string
	Type        NodeType
	Category    string
	Icon        string
	Color       string
	Version     string
	Inputs      []ParameterDefinition
	Outputs     []ParameterDefinition
	Config      map[string]interface{}
}

type ParameterDefinition struct {
	Name        string
	Type        string
	Required    bool
	Default     interface{}
	Description string
	Options     []ParameterOption
}

type ParameterOption struct {
	Label string
	Value interface{}
}

func NewNodeDefinition(id, name string, nodeType NodeType) *NodeDefinition {
	return &NodeDefinition{
		ID:      id,
		Name:    name,
		Type:    nodeType,
		Version: "1.0.0",
		Inputs:  []ParameterDefinition{},
		Outputs: []ParameterDefinition{},
		Config:  make(map[string]interface{}),
	}
}

func (n *NodeDefinition) AddInput(param ParameterDefinition) {
	n.Inputs = append(n.Inputs, param)
}

func (n *NodeDefinition) AddOutput(param ParameterDefinition) {
	n.Outputs = append(n.Outputs, param)
}

func (n *NodeDefinition) Validate() []string {
	var errors []string
	if n.ID == "" {
		errors = append(errors, "node ID is required")
	}
	if n.Name == "" {
		errors = append(errors, "node name is required")
	}
	if n.Type == "" {
		errors = append(errors, "node type is required")
	}
	return errors
}

func (n *NodeDefinition) ValidateConfig(config map[string]interface{}) []string {
	var errors []string
	for _, input := range n.Inputs {
		if input.Required {
			if _, ok := config[input.Name]; !ok {
				errors = append(errors, "required parameter missing: "+input.Name)
			}
		}
	}
	return errors
}

func (n *NodeDefinition) GetRequiredInputs() []ParameterDefinition {
	var required []ParameterDefinition
	for _, input := range n.Inputs {
		if input.Required {
			required = append(required, input)
		}
	}
	return required
}

func (n *NodeDefinition) Clone() *NodeDefinition {
	clone := *n
	clone.Inputs = make([]ParameterDefinition, len(n.Inputs))
	copy(clone.Inputs, n.Inputs)
	clone.Outputs = make([]ParameterDefinition, len(n.Outputs))
	copy(clone.Outputs, n.Outputs)
	clone.Config = make(map[string]interface{})
	for k, v := range n.Config {
		clone.Config[k] = v
	}
	return &clone
}

// HTTP Request Node builder
func NewHTTPRequestNode() *NodeDefinition {
	node := NewNodeDefinition("http-request", "HTTP Request", NodeTypeAction)
	node.Description = "Make HTTP requests to external APIs"
	node.Category = "Core"
	node.Icon = "http"
	node.Color = "#4CAF50"

	node.AddInput(ParameterDefinition{
		Name:        "url",
		Type:        "string",
		Required:    true,
		Description: "The URL to send the request to",
	})
	node.AddInput(ParameterDefinition{
		Name:        "method",
		Type:        "string",
		Required:    true,
		Default:     "GET",
		Description: "HTTP method",
		Options: []ParameterOption{
			{Label: "GET", Value: "GET"},
			{Label: "POST", Value: "POST"},
			{Label: "PUT", Value: "PUT"},
			{Label: "DELETE", Value: "DELETE"},
			{Label: "PATCH", Value: "PATCH"},
		},
	})
	node.AddInput(ParameterDefinition{
		Name:        "headers",
		Type:        "object",
		Required:    false,
		Description: "Request headers",
	})
	node.AddInput(ParameterDefinition{
		Name:        "body",
		Type:        "object",
		Required:    false,
		Description: "Request body",
	})

	node.AddOutput(ParameterDefinition{
		Name:        "statusCode",
		Type:        "number",
		Description: "HTTP status code",
	})
	node.AddOutput(ParameterDefinition{
		Name:        "response",
		Type:        "object",
		Description: "Response body",
	})
	node.AddOutput(ParameterDefinition{
		Name:        "headers",
		Type:        "object",
		Description: "Response headers",
	})

	return node
}

// Tests
func TestNewNodeDefinition(t *testing.T) {
	t.Run("creates node definition", func(t *testing.T) {
		node := NewNodeDefinition("test-node", "Test Node", NodeTypeAction)
		require.NotNil(t, node)
		assert.Equal(t, "test-node", node.ID)
		assert.Equal(t, "Test Node", node.Name)
		assert.Equal(t, NodeTypeAction, node.Type)
		assert.Equal(t, "1.0.0", node.Version)
	})
}

func TestNodeDefinition_Validate(t *testing.T) {
	t.Run("valid node has no errors", func(t *testing.T) {
		node := NewNodeDefinition("test", "Test", NodeTypeAction)
		errors := node.Validate()
		assert.Empty(t, errors)
	})

	t.Run("missing ID returns error", func(t *testing.T) {
		node := &NodeDefinition{Name: "Test", Type: NodeTypeAction}
		errors := node.Validate()
		assert.Contains(t, errors, "node ID is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		node := &NodeDefinition{ID: "test", Type: NodeTypeAction}
		errors := node.Validate()
		assert.Contains(t, errors, "node name is required")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		node := &NodeDefinition{ID: "test", Name: "Test"}
		errors := node.Validate()
		assert.Contains(t, errors, "node type is required")
	})
}

func TestNodeDefinition_ValidateConfig(t *testing.T) {
	t.Run("valid config passes", func(t *testing.T) {
		node := NewHTTPRequestNode()
		config := map[string]interface{}{
			"url":    "https://api.example.com",
			"method": "GET",
		}
		errors := node.ValidateConfig(config)
		assert.Empty(t, errors)
	})

	t.Run("missing required param fails", func(t *testing.T) {
		node := NewHTTPRequestNode()
		config := map[string]interface{}{
			"method": "GET",
		}
		errors := node.ValidateConfig(config)
		assert.Contains(t, errors, "required parameter missing: url")
	})
}

func TestNodeDefinition_GetRequiredInputs(t *testing.T) {
	node := NewHTTPRequestNode()
	required := node.GetRequiredInputs()
	assert.Len(t, required, 2)
	assert.Equal(t, "url", required[0].Name)
	assert.Equal(t, "method", required[1].Name)
}

func TestNodeDefinition_Clone(t *testing.T) {
	t.Run("creates independent copy", func(t *testing.T) {
		original := NewHTTPRequestNode()
		original.Config["test"] = "value"

		clone := original.Clone()
		clone.Name = "Modified"
		clone.Config["test"] = "modified"

		assert.Equal(t, "HTTP Request", original.Name)
		assert.Equal(t, "value", original.Config["test"])
		assert.Equal(t, "Modified", clone.Name)
		assert.Equal(t, "modified", clone.Config["test"])
	})
}

func TestHTTPRequestNode(t *testing.T) {
	node := NewHTTPRequestNode()

	t.Run("has correct metadata", func(t *testing.T) {
		assert.Equal(t, "http-request", node.ID)
		assert.Equal(t, "HTTP Request", node.Name)
		assert.Equal(t, NodeTypeAction, node.Type)
		assert.Equal(t, "Core", node.Category)
	})

	t.Run("has required inputs", func(t *testing.T) {
		assert.GreaterOrEqual(t, len(node.Inputs), 2)
		urlInput := node.Inputs[0]
		assert.Equal(t, "url", urlInput.Name)
		assert.True(t, urlInput.Required)
	})

	t.Run("has outputs", func(t *testing.T) {
		assert.Len(t, node.Outputs, 3)
	})
}
