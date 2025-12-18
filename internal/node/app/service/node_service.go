package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrNodeAlreadyExists = errors.New("node already exists")
	ErrInvalidNode       = errors.New("invalid node")
	ErrSystemNode        = errors.New("cannot modify system node")
)

// NodeService handles node definition application logic
type NodeService struct {
	repository     repository.NodeDefinitionRepository
	eventPublisher *kafka.EventPublisher
	cache          *cache.RedisCache
	logger         logger.Logger
}

// NewNodeService creates a new node service
func NewNodeService(
	repository repository.NodeDefinitionRepository,
	eventPublisher *kafka.EventPublisher,
	cache *cache.RedisCache,
	logger logger.Logger,
) *NodeService {
	return &NodeService{
		repository:     repository,
		eventPublisher: eventPublisher,
		cache:          cache,
		logger:         logger,
	}
}

// CreateNodeCommand represents a command to create a node
type CreateNodeCommand struct {
	Name             string
	Type             string
	Category         string
	Description      string
	Icon             string
	Color            string
	Inputs           []model.NodePort
	Outputs          []model.NodePort
	Properties       []model.NodeProperty
	ExecutionHandler string
	Documentation    string
	Tags             []string
	IsPremium        bool
}

// CreateNode creates a new node definition
func (s *NodeService) CreateNode(ctx context.Context, cmd CreateNodeCommand) (*model.NodeDefinition, error) {
	// Check if node with same name exists
	existing, _ := s.repository.FindByName(ctx, cmd.Name)
	if existing != nil {
		return nil, ErrNodeAlreadyExists
	}

	// Create node definition
	node, err := model.NewNodeDefinition(
		cmd.Name,
		model.NodeType(cmd.Type),
		model.NodeCategory(cmd.Category),
		cmd.Description,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	// Set properties
	if cmd.Icon != "" {
		node.SetIcon(cmd.Icon)
	}
	if cmd.Color != "" {
		node.SetColor(cmd.Color)
	}
	if cmd.ExecutionHandler != "" {
		node.SetExecutionHandler(cmd.ExecutionHandler)
	}
	if cmd.Documentation != "" {
		node.SetDocumentation(cmd.Documentation)
	}
	if cmd.IsPremium {
		node.MarkAsPremium()
	}

	// Add inputs
	for _, input := range cmd.Inputs {
		if err := node.AddInput(input); err != nil {
			return nil, fmt.Errorf("failed to add input: %w", err)
		}
	}

	// Add outputs
	for _, output := range cmd.Outputs {
		if err := node.AddOutput(output); err != nil {
			return nil, fmt.Errorf("failed to add output: %w", err)
		}
	}

	// Add properties
	for _, prop := range cmd.Properties {
		if err := node.AddProperty(prop); err != nil {
			return nil, fmt.Errorf("failed to add property: %w", err)
		}
	}

	// Add tags
	for _, tag := range cmd.Tags {
		node.AddTag(tag)
	}

	// Validate node
	if err := node.Validate(); err != nil {
		return nil, fmt.Errorf("node validation failed: %w", err)
	}

	// Save to repository
	if err := s.repository.Save(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	// Publish event
	if s.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"nodeId":   node.ID().String(),
			"name":     node.Name(),
			"type":     string(node.Type()),
			"category": string(node.Category()),
		})
		event := &events.Event{
			AggregateID:   node.ID().String(),
			AggregateType: "NodeDefinition",
			EventType:     "node.created",
			Timestamp:     time.Now(),
			Payload:       json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Node created", 
		"node_id", node.ID(),
		"name", node.Name(),
		"type", node.Type(),
	)

	return node, nil
}

// UpdateNodeCommand represents a command to update a node
type UpdateNodeCommand struct {
	ID               string
	Name             string
	Description      string
	Icon             string
	Color            string
	Inputs           []model.NodePort
	Outputs          []model.NodePort
	Properties       []model.NodeProperty
	ExecutionHandler string
	Documentation    string
	Tags             []string
	Status           string
}

// UpdateNode updates a node definition
func (s *NodeService) UpdateNode(ctx context.Context, cmd UpdateNodeCommand) (*model.NodeDefinition, error) {
	// Get existing node
	node, err := s.repository.FindByID(ctx, model.NodeID(cmd.ID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to find node: %w", err)
	}

	// Check if system node
	if node.IsSystem() {
		return nil, ErrSystemNode
	}

	// Update fields
	if cmd.Name != "" && cmd.Name != node.Name() {
		if err := node.SetName(cmd.Name); err != nil {
			return nil, err
		}
	}
	if cmd.Description != "" {
		node.SetDescription(cmd.Description)
	}
	if cmd.Icon != "" {
		node.SetIcon(cmd.Icon)
	}
	if cmd.Color != "" {
		node.SetColor(cmd.Color)
	}
	if cmd.ExecutionHandler != "" {
		node.SetExecutionHandler(cmd.ExecutionHandler)
	}
	if cmd.Documentation != "" {
		node.SetDocumentation(cmd.Documentation)
	}
	if cmd.Status != "" {
		node.SetStatus(model.NodeStatus(cmd.Status))
	}

	// Update repository
	if err := s.repository.Update(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to update node: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("node:%s", cmd.ID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   node.ID().String(),
			AggregateType: "NodeDefinition",
			EventType:     "node.updated",
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Node updated", "node_id", node.ID())
	return node, nil
}

// GetNode gets a node by ID
func (s *NodeService) GetNode(ctx context.Context, nodeID model.NodeID) (*model.NodeDefinition, error) {
	// Try cache first
	if s.cache != nil {
		var node model.NodeDefinition
		cacheKey := fmt.Sprintf("node:%s", nodeID)
		if err := s.cache.Get(ctx, cacheKey, &node); err == nil {
			return &node, nil
		}
	}

	// Get from repository
	node, err := s.repository.FindByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// Cache the result
	if s.cache != nil {
		cacheKey := fmt.Sprintf("node:%s", nodeID)
		_ = s.cache.Set(ctx, cacheKey, node, 5*time.Minute)
	}

	return node, nil
}

// ListNodesQuery represents a query to list nodes
type ListNodesQuery struct {
	Type        string
	Category    string
	Status      string
	IsSystem    *bool
	IsPremium   *bool
	SearchQuery string
	Offset      int
	Limit       int
}

// ListNodes lists node definitions
func (s *NodeService) ListNodes(ctx context.Context, query ListNodesQuery) ([]*model.NodeDefinition, int64, error) {
	var nodes []*model.NodeDefinition
	var err error

	// Search if query provided
	if query.SearchQuery != "" {
		nodes, err = s.repository.Search(ctx, query.SearchQuery, query.Offset, query.Limit)
	} else if query.Type != "" {
		nodes, err = s.repository.FindByType(ctx, model.NodeType(query.Type), query.Offset, query.Limit)
	} else if query.Category != "" {
		nodes, err = s.repository.FindByCategory(ctx, model.NodeCategory(query.Category), query.Offset, query.Limit)
	} else if query.Status != "" {
		nodes, err = s.repository.FindByStatus(ctx, model.NodeStatus(query.Status), query.Offset, query.Limit)
	} else if query.IsSystem != nil && *query.IsSystem {
		nodes, err = s.repository.FindSystemNodes(ctx, query.Offset, query.Limit)
	} else {
		nodes, err = s.repository.FindAll(ctx, query.Offset, query.Limit)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Filter by premium status if specified
	if query.IsPremium != nil {
		filtered := make([]*model.NodeDefinition, 0)
		for _, node := range nodes {
			if node.IsPremium() == *query.IsPremium {
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}

	// Get total count
	total, err := s.repository.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count nodes: %w", err)
	}

	return nodes, total, nil
}

// CloneNode clones a node definition
func (s *NodeService) CloneNode(ctx context.Context, nodeID model.NodeID) (*model.NodeDefinition, error) {
	// Get original node
	original, err := s.repository.FindByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, fmt.Errorf("failed to find node: %w", err)
	}

	// Clone the node
	clone := original.Clone()

	// Save the clone
	if err := s.repository.Save(ctx, clone); err != nil {
		return nil, fmt.Errorf("failed to save cloned node: %w", err)
	}

	// Publish event
	if s.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"originalId": original.ID().String(),
			"cloneId":    clone.ID().String(),
		})
		event := &events.Event{
			AggregateID:   clone.ID().String(),
			AggregateType: "NodeDefinition",
			EventType:     "node.cloned",
			Timestamp:     time.Now(),
			Payload:       json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Node cloned", 
		"original_id", original.ID(),
		"clone_id", clone.ID(),
	)

	return clone, nil
}

// DeleteNode deletes a node definition
func (s *NodeService) DeleteNode(ctx context.Context, nodeID model.NodeID) error {
	// Get node to check if it's a system node
	node, err := s.repository.FindByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNodeNotFound
		}
		return fmt.Errorf("failed to find node: %w", err)
	}

	// Check if system node
	if node.IsSystem() {
		return ErrSystemNode
	}

	// Delete from repository
	if err := s.repository.Delete(ctx, nodeID); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("node:%s", nodeID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   nodeID.String(),
			AggregateType: "NodeDefinition",
			EventType:     "node.deleted",
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Node deleted", "node_id", nodeID)
	return nil
}

// InitializeSystemNodes initializes default system nodes
func (s *NodeService) InitializeSystemNodes(ctx context.Context) error {
	systemNodes := s.getDefaultSystemNodes()
	
	for _, cmd := range systemNodes {
		// Check if node already exists
		existing, _ := s.repository.FindByName(ctx, cmd.Name)
		if existing != nil {
			continue // Skip if already exists
		}

		// Create node
		node, err := s.CreateNode(ctx, cmd)
		if err != nil {
			s.logger.Error("Failed to create system node", "name", cmd.Name, "error", err)
			continue
		}

		// Mark as system node
		node.MarkAsSystem()
		
		// Update in repository
		if err := s.repository.Update(ctx, node); err != nil {
			s.logger.Error("Failed to mark node as system", "node_id", node.ID(), "error", err)
		}
	}

	return nil
}

// getDefaultSystemNodes returns the default system node definitions
func (s *NodeService) getDefaultSystemNodes() []CreateNodeCommand {
	return []CreateNodeCommand{
		{
			Name:        "HTTP Request",
			Type:        string(model.NodeTypeHTTP),
			Category:    string(model.NodeCategoryIntegration),
			Description: "Make HTTP requests to external APIs",
			Icon:        "http",
			Color:       "#4CAF50",
			Inputs: []model.NodePort{
				{ID: "url", Name: "URL", Type: "string", Required: true},
				{ID: "method", Name: "Method", Type: "string", Required: true},
				{ID: "headers", Name: "Headers", Type: "object", Required: false},
				{ID: "body", Name: "Body", Type: "any", Required: false},
			},
			Outputs: []model.NodePort{
				{ID: "response", Name: "Response", Type: "object"},
				{ID: "status", Name: "Status Code", Type: "number"},
				{ID: "error", Name: "Error", Type: "string"},
			},
			Properties: []model.NodeProperty{
				{ID: "timeout", Name: "Timeout", Type: "number", DefaultValue: 30000},
				{ID: "retries", Name: "Retries", Type: "number", DefaultValue: 3},
			},
		},
		{
			Name:        "Condition",
			Type:        string(model.NodeTypeCondition),
			Category:    string(model.NodeCategoryControl),
			Description: "Branch workflow based on conditions",
			Icon:        "condition",
			Color:       "#2196F3",
			Inputs: []model.NodePort{
				{ID: "input", Name: "Input", Type: "any", Required: true},
			},
			Outputs: []model.NodePort{
				{ID: "true", Name: "True", Type: "any"},
				{ID: "false", Name: "False", Type: "any"},
			},
			Properties: []model.NodeProperty{
				{ID: "condition", Name: "Condition", Type: "string", Required: true},
				{ID: "operator", Name: "Operator", Type: "select", Required: true,
					Options: []model.PropertyOption{
						{Label: "Equals", Value: "=="},
						{Label: "Not Equals", Value: "!="},
						{Label: "Greater Than", Value: ">"},
						{Label: "Less Than", Value: "<"},
					},
				},
			},
		},
		{
			Name:        "Transform",
			Type:        string(model.NodeTypeTransform),
			Category:    string(model.NodeCategoryTransform),
			Description: "Transform data using JavaScript expressions",
			Icon:        "transform",
			Color:       "#FF9800",
			Inputs: []model.NodePort{
				{ID: "input", Name: "Input", Type: "any", Required: true},
			},
			Outputs: []model.NodePort{
				{ID: "output", Name: "Output", Type: "any"},
			},
			Properties: []model.NodeProperty{
				{ID: "expression", Name: "Expression", Type: "string", Required: true},
			},
		},
	}
}
