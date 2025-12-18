package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// NodeID represents a unique node identifier
type NodeID string

// NewNodeID creates a new node ID
func NewNodeID() NodeID {
	return NodeID(uuid.New().String())
}

func (id NodeID) String() string {
	return string(id)
}

// NodeType represents the type of node
type NodeType string

const (
	NodeTypeTrigger     NodeType = "trigger"
	NodeTypeAction      NodeType = "action"
	NodeTypeCondition   NodeType = "condition"
	NodeTypeLoop        NodeType = "loop"
	NodeTypeSwitch      NodeType = "switch"
	NodeTypeTransform   NodeType = "transform"
	NodeTypeAggregator  NodeType = "aggregator"
	NodeTypeSchedule    NodeType = "schedule"
	NodeTypeWebhook     NodeType = "webhook"
	NodeTypeHTTP        NodeType = "http"
	NodeTypeDatabase    NodeType = "database"
	NodeTypeFile        NodeType = "file"
	NodeTypeEmail       NodeType = "email"
	NodeTypeNotification NodeType = "notification"
	NodeTypeCustom      NodeType = "custom"
)

// NodeCategory represents the category of node
type NodeCategory string

const (
	NodeCategoryCore         NodeCategory = "core"
	NodeCategoryIntegration  NodeCategory = "integration"
	NodeCategoryTransform    NodeCategory = "transform"
	NodeCategoryControl      NodeCategory = "control"
	NodeCategoryCommunication NodeCategory = "communication"
	NodeCategoryStorage      NodeCategory = "storage"
	NodeCategoryCustom       NodeCategory = "custom"
)

// NodeStatus represents the status of a node definition
type NodeStatus string

const (
	NodeStatusActive     NodeStatus = "active"
	NodeStatusDeprecated NodeStatus = "deprecated"
	NodeStatusBeta       NodeStatus = "beta"
	NodeStatusDisabled   NodeStatus = "disabled"
)

// NodePort represents an input or output port
type NodePort struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // string, number, boolean, object, array, any
	Required    bool                   `json:"required"`
	Multiple    bool                   `json:"multiple"`
	Description string                 `json:"description"`
	DefaultValue interface{}           `json:"defaultValue,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
}

// NodeProperty represents a configuration property
type NodeProperty struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // string, number, boolean, select, multiselect, json
	Required     bool                   `json:"required"`
	Description  string                 `json:"description"`
	DefaultValue interface{}           `json:"defaultValue,omitempty"`
	Options      []PropertyOption       `json:"options,omitempty"`
	Validation   map[string]interface{} `json:"validation,omitempty"`
	Placeholder  string                 `json:"placeholder,omitempty"`
	Hidden       bool                   `json:"hidden"`
}

// PropertyOption represents an option for select/multiselect properties
type PropertyOption struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// NodeDefinition represents a node definition aggregate
type NodeDefinition struct {
	id                NodeID
	name              string
	nodeType          NodeType
	category          NodeCategory
	description       string
	icon              string
	color             string
	version           string
	status            NodeStatus
	inputs            []NodePort
	outputs           []NodePort
	properties        []NodeProperty
	configuration     map[string]interface{}
	validationRules   map[string]interface{}
	documentation     string
	examples          []map[string]interface{}
	tags              []string
	author            string
	isSystem          bool
	isPremium         bool
	executionHandler  string
	metadata          map[string]interface{}
	createdAt         time.Time
	updatedAt         time.Time
}

// NewNodeDefinition creates a new node definition
func NewNodeDefinition(
	name string,
	nodeType NodeType,
	category NodeCategory,
	description string,
) (*NodeDefinition, error) {
	if name == "" {
		return nil, errors.New("node name is required")
	}
	if nodeType == "" {
		return nil, errors.New("node type is required")
	}
	if category == "" {
		category = NodeCategoryCore
	}

	now := time.Now()
	return &NodeDefinition{
		id:              NewNodeID(),
		name:            name,
		nodeType:        nodeType,
		category:        category,
		description:     description,
		version:         "1.0.0",
		status:          NodeStatusActive,
		inputs:          []NodePort{},
		outputs:         []NodePort{},
		properties:      []NodeProperty{},
		configuration:   make(map[string]interface{}),
		validationRules: make(map[string]interface{}),
		examples:        []map[string]interface{}{},
		tags:            []string{},
		metadata:        make(map[string]interface{}),
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

// Getters
func (n *NodeDefinition) ID() NodeID                          { return n.id }
func (n *NodeDefinition) Name() string                        { return n.name }
func (n *NodeDefinition) Type() NodeType                      { return n.nodeType }
func (n *NodeDefinition) Category() NodeCategory              { return n.category }
func (n *NodeDefinition) Description() string                 { return n.description }
func (n *NodeDefinition) Icon() string                        { return n.icon }
func (n *NodeDefinition) Color() string                       { return n.color }
func (n *NodeDefinition) Version() string                     { return n.version }
func (n *NodeDefinition) Status() NodeStatus                  { return n.status }
func (n *NodeDefinition) Inputs() []NodePort                  { return n.inputs }
func (n *NodeDefinition) Outputs() []NodePort                 { return n.outputs }
func (n *NodeDefinition) Properties() []NodeProperty          { return n.properties }
func (n *NodeDefinition) Configuration() map[string]interface{} { return n.configuration }
func (n *NodeDefinition) Tags() []string                      { return n.tags }
func (n *NodeDefinition) IsSystem() bool                      { return n.isSystem }
func (n *NodeDefinition) IsPremium() bool                     { return n.isPremium }
func (n *NodeDefinition) ExecutionHandler() string            { return n.executionHandler }
func (n *NodeDefinition) CreatedAt() time.Time                { return n.createdAt }
func (n *NodeDefinition) UpdatedAt() time.Time                { return n.updatedAt }

// SetName updates the node name
func (n *NodeDefinition) SetName(name string) error {
	if name == "" {
		return errors.New("node name cannot be empty")
	}
	n.name = name
	n.updatedAt = time.Now()
	return nil
}

// SetDescription updates the node description
func (n *NodeDefinition) SetDescription(description string) {
	n.description = description
	n.updatedAt = time.Now()
}

// SetIcon sets the node icon
func (n *NodeDefinition) SetIcon(icon string) {
	n.icon = icon
	n.updatedAt = time.Now()
}

// SetColor sets the node color
func (n *NodeDefinition) SetColor(color string) {
	n.color = color
	n.updatedAt = time.Now()
}

// AddInput adds an input port
func (n *NodeDefinition) AddInput(port NodePort) error {
	// Check for duplicate port ID
	for _, existing := range n.inputs {
		if existing.ID == port.ID {
			return fmt.Errorf("input port with ID %s already exists", port.ID)
		}
	}
	
	n.inputs = append(n.inputs, port)
	n.updatedAt = time.Now()
	return nil
}

// AddOutput adds an output port
func (n *NodeDefinition) AddOutput(port NodePort) error {
	// Check for duplicate port ID
	for _, existing := range n.outputs {
		if existing.ID == port.ID {
			return fmt.Errorf("output port with ID %s already exists", port.ID)
		}
	}
	
	n.outputs = append(n.outputs, port)
	n.updatedAt = time.Now()
	return nil
}

// AddProperty adds a configuration property
func (n *NodeDefinition) AddProperty(property NodeProperty) error {
	// Check for duplicate property ID
	for _, existing := range n.properties {
		if existing.ID == property.ID {
			return fmt.Errorf("property with ID %s already exists", property.ID)
		}
	}
	
	n.properties = append(n.properties, property)
	n.updatedAt = time.Now()
	return nil
}

// RemoveInput removes an input port by ID
func (n *NodeDefinition) RemoveInput(portID string) error {
	for i, port := range n.inputs {
		if port.ID == portID {
			n.inputs = append(n.inputs[:i], n.inputs[i+1:]...)
			n.updatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("input port with ID %s not found", portID)
}

// RemoveOutput removes an output port by ID
func (n *NodeDefinition) RemoveOutput(portID string) error {
	for i, port := range n.outputs {
		if port.ID == portID {
			n.outputs = append(n.outputs[:i], n.outputs[i+1:]...)
			n.updatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("output port with ID %s not found", portID)
}

// RemoveProperty removes a property by ID
func (n *NodeDefinition) RemoveProperty(propertyID string) error {
	for i, prop := range n.properties {
		if prop.ID == propertyID {
			n.properties = append(n.properties[:i], n.properties[i+1:]...)
			n.updatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("property with ID %s not found", propertyID)
}

// SetStatus updates the node status
func (n *NodeDefinition) SetStatus(status NodeStatus) {
	n.status = status
	n.updatedAt = time.Now()
}

// Deprecate marks the node as deprecated
func (n *NodeDefinition) Deprecate() {
	n.status = NodeStatusDeprecated
	n.updatedAt = time.Now()
}

// Disable disables the node
func (n *NodeDefinition) Disable() {
	n.status = NodeStatusDisabled
	n.updatedAt = time.Now()
}

// Enable enables the node
func (n *NodeDefinition) Enable() {
	n.status = NodeStatusActive
	n.updatedAt = time.Now()
}

// AddTag adds a tag to the node
func (n *NodeDefinition) AddTag(tag string) {
	// Check if tag already exists
	for _, existing := range n.tags {
		if existing == tag {
			return
		}
	}
	n.tags = append(n.tags, tag)
	n.updatedAt = time.Now()
}

// RemoveTag removes a tag from the node
func (n *NodeDefinition) RemoveTag(tag string) {
	for i, existing := range n.tags {
		if existing == tag {
			n.tags = append(n.tags[:i], n.tags[i+1:]...)
			n.updatedAt = time.Now()
			return
		}
	}
}

// SetExecutionHandler sets the execution handler
func (n *NodeDefinition) SetExecutionHandler(handler string) {
	n.executionHandler = handler
	n.updatedAt = time.Now()
}

// MarkAsSystem marks the node as a system node
func (n *NodeDefinition) MarkAsSystem() {
	n.isSystem = true
	n.updatedAt = time.Now()
}

// MarkAsPremium marks the node as premium
func (n *NodeDefinition) MarkAsPremium() {
	n.isPremium = true
	n.updatedAt = time.Now()
}

// SetDocumentation sets the documentation
func (n *NodeDefinition) SetDocumentation(documentation string) {
	n.documentation = documentation
	n.updatedAt = time.Now()
}

// AddExample adds an example
func (n *NodeDefinition) AddExample(example map[string]interface{}) {
	n.examples = append(n.examples, example)
	n.updatedAt = time.Now()
}

// Validate validates the node definition
func (n *NodeDefinition) Validate() error {
	if n.name == "" {
		return errors.New("node name is required")
	}
	if n.nodeType == "" {
		return errors.New("node type is required")
	}
	if n.category == "" {
		return errors.New("node category is required")
	}
	
	// Validate inputs
	for _, input := range n.inputs {
		if input.ID == "" {
			return errors.New("input port ID is required")
		}
		if input.Name == "" {
			return errors.New("input port name is required")
		}
		if input.Type == "" {
			return errors.New("input port type is required")
		}
	}
	
	// Validate outputs
	for _, output := range n.outputs {
		if output.ID == "" {
			return errors.New("output port ID is required")
		}
		if output.Name == "" {
			return errors.New("output port name is required")
		}
		if output.Type == "" {
			return errors.New("output port type is required")
		}
	}
	
	// Validate properties
	for _, prop := range n.properties {
		if prop.ID == "" {
			return errors.New("property ID is required")
		}
		if prop.Name == "" {
			return errors.New("property name is required")
		}
		if prop.Type == "" {
			return errors.New("property type is required")
		}
	}
	
	return nil
}

// Clone creates a copy of the node definition
func (n *NodeDefinition) Clone() *NodeDefinition {
	clone := &NodeDefinition{
		id:               NewNodeID(),
		name:             n.name + " (Copy)",
		nodeType:         n.nodeType,
		category:         n.category,
		description:      n.description,
		icon:             n.icon,
		color:            n.color,
		version:          n.version,
		status:           n.status,
		inputs:           make([]NodePort, len(n.inputs)),
		outputs:          make([]NodePort, len(n.outputs)),
		properties:       make([]NodeProperty, len(n.properties)),
		configuration:    make(map[string]interface{}),
		validationRules:  make(map[string]interface{}),
		documentation:    n.documentation,
		examples:         make([]map[string]interface{}, len(n.examples)),
		tags:             make([]string, len(n.tags)),
		author:           n.author,
		isSystem:         false, // Cloned nodes are not system nodes
		isPremium:        n.isPremium,
		executionHandler: n.executionHandler,
		metadata:         make(map[string]interface{}),
		createdAt:        time.Now(),
		updatedAt:        time.Now(),
	}
	
	// Deep copy arrays and maps
	copy(clone.inputs, n.inputs)
	copy(clone.outputs, n.outputs)
	copy(clone.properties, n.properties)
	copy(clone.tags, n.tags)
	copy(clone.examples, n.examples)
	
	for k, v := range n.configuration {
		clone.configuration[k] = v
	}
	
	for k, v := range n.validationRules {
		clone.validationRules[k] = v
	}
	
	for k, v := range n.metadata {
		clone.metadata[k] = v
	}
	
	return clone
}
