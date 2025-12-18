package domainservice

import (
	"context"
	"fmt"

	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/repository"
)

// WorkflowDomainService handles complex business logic involving multiple aggregates
type WorkflowDomainService struct {
	repo repository.WorkflowRepository
}

// NewWorkflowDomainService creates a new workflow domain service
func NewWorkflowDomainService(repo repository.WorkflowRepository) *WorkflowDomainService {
	return &WorkflowDomainService{
		repo: repo,
	}
}

// DuplicateWorkflow creates a duplicate of an existing workflow
func (s *WorkflowDomainService) DuplicateWorkflow(ctx context.Context, sourceID model.WorkflowID, newName string) (*model.Workflow, error) {
	// Load source workflow
	source, err := s.repo.FindByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find source workflow: %w", err)
	}

	// Create new workflow with copied properties
	duplicate, err := model.NewWorkflow(source.UserID(), newName, source.Description())
	if err != nil {
		return nil, fmt.Errorf("failed to create duplicate workflow: %w", err)
	}

	// Copy nodes
	for _, node := range source.Nodes() {
		nodeCopy := node // Create a copy
		if err := duplicate.AddNode(nodeCopy); err != nil {
			return nil, fmt.Errorf("failed to add node to duplicate: %w", err)
		}
	}

	// Copy connections
	for _, conn := range source.Connections() {
		connCopy := conn // Create a copy
		if err := duplicate.AddConnection(connCopy); err != nil {
			return nil, fmt.Errorf("failed to add connection to duplicate: %w", err)
		}
	}

	// Copy settings
	if err := duplicate.UpdateSettings(source.Settings()); err != nil {
		return nil, fmt.Errorf("failed to update settings in duplicate: %w", err)
	}

	// Save duplicate
	if err := s.repo.Save(ctx, duplicate); err != nil {
		return nil, fmt.Errorf("failed to save duplicate workflow: %w", err)
	}

	return duplicate, nil
}

// ValidateWorkflowExecutability checks if a workflow can be executed
func (s *WorkflowDomainService) ValidateWorkflowExecutability(ctx context.Context, id model.WorkflowID) error {
	workflow, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find workflow: %w", err)
	}

	// Check if workflow is active
	if workflow.Status() != model.WorkflowStatusActive {
		return fmt.Errorf("workflow must be active to execute")
	}

	// Validate has trigger node
	hasTrigger := false
	for _, node := range workflow.Nodes() {
		if node.Type == model.NodeTypeTrigger {
			hasTrigger = true
			break
		}
	}

	if !hasTrigger {
		return fmt.Errorf("workflow must have at least one trigger node")
	}

	// Validate all nodes have required configuration
	for _, node := range workflow.Nodes() {
		if err := s.validateNodeConfig(node); err != nil {
			return fmt.Errorf("node %s has invalid configuration: %w", node.ID, err)
		}
	}

	// Validate connections are complete
	if err := s.validateConnections(workflow); err != nil {
		return fmt.Errorf("invalid workflow connections: %w", err)
	}

	return nil
}

// MergeWorkflows merges multiple workflows into one
func (s *WorkflowDomainService) MergeWorkflows(ctx context.Context, sourceIDs []model.WorkflowID, newName string) (*model.Workflow, error) {
	if len(sourceIDs) < 2 {
		return nil, fmt.Errorf("at least two workflows required for merge")
	}

	// Load all source workflows
	var sources []*model.Workflow
	var userID string
	
	for i, id := range sourceIDs {
		source, err := s.repo.FindByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to find workflow %s: %w", id, err)
		}
		
		if i == 0 {
			userID = source.UserID()
		} else if source.UserID() != userID {
			return nil, fmt.Errorf("all workflows must belong to the same user")
		}
		
		sources = append(sources, source)
	}

	// Create merged workflow
	merged, err := model.NewWorkflow(userID, newName, "Merged workflow")
	if err != nil {
		return nil, fmt.Errorf("failed to create merged workflow: %w", err)
	}

	// Merge nodes from all workflows
	nodeIDMapping := make(map[string]string) // old ID -> new ID mapping
	
	for _, source := range sources {
		for _, node := range source.Nodes() {
			// Create new ID to avoid conflicts
			oldID := node.ID
			node.ID = fmt.Sprintf("%s_%s", source.ID().String()[:8], oldID)
			nodeIDMapping[oldID] = node.ID
			
			if err := merged.AddNode(node); err != nil {
				return nil, fmt.Errorf("failed to add node to merged workflow: %w", err)
			}
		}
	}

	// Merge connections with updated node IDs
	for _, source := range sources {
		for _, conn := range source.Connections() {
			// Update connection IDs based on mapping
			if newSourceID, ok := nodeIDMapping[conn.SourceNodeID]; ok {
				conn.SourceNodeID = newSourceID
			}
			if newTargetID, ok := nodeIDMapping[conn.TargetNodeID]; ok {
				conn.TargetNodeID = newTargetID
			}
			
			if err := merged.AddConnection(conn); err != nil {
				// Ignore connection errors in merge (might be invalid after merge)
				continue
			}
		}
	}

	// Save merged workflow
	if err := s.repo.Save(ctx, merged); err != nil {
		return nil, fmt.Errorf("failed to save merged workflow: %w", err)
	}

	return merged, nil
}

// validateNodeConfig validates a node's configuration
func (s *WorkflowDomainService) validateNodeConfig(node model.Node) error {
	// Basic validation - extend based on node type
	if node.ID == "" {
		return fmt.Errorf("node ID is required")
	}
	if node.Name == "" {
		return fmt.Errorf("node name is required")
	}
	
	// Type-specific validation
	switch node.Type {
	case model.NodeTypeTrigger:
		// Validate trigger-specific config
		if node.Config == nil {
			return fmt.Errorf("trigger node requires configuration")
		}
	case model.NodeTypeAction:
		// Validate action-specific config
		if node.Config == nil {
			return fmt.Errorf("action node requires configuration")
		}
	case model.NodeTypeCondition:
		// Validate condition-specific config
		if node.Config == nil || node.Config["condition"] == nil {
			return fmt.Errorf("condition node requires condition configuration")
		}
	}
	
	return nil
}

// validateConnections validates workflow connections are complete
func (s *WorkflowDomainService) validateConnections(workflow *model.Workflow) error {
	nodeMap := make(map[string]bool)
	for _, node := range workflow.Nodes() {
		nodeMap[node.ID] = true
	}

	// Check all connections reference valid nodes
	for _, conn := range workflow.Connections() {
		if !nodeMap[conn.SourceNodeID] {
			return fmt.Errorf("connection references non-existent source node: %s", conn.SourceNodeID)
		}
		if !nodeMap[conn.TargetNodeID] {
			return fmt.Errorf("connection references non-existent target node: %s", conn.TargetNodeID)
		}
	}

	// Check all non-trigger nodes have incoming connections
	incomingConnections := make(map[string]bool)
	for _, conn := range workflow.Connections() {
		incomingConnections[conn.TargetNodeID] = true
	}

	for _, node := range workflow.Nodes() {
		if node.Type != model.NodeTypeTrigger && !incomingConnections[node.ID] {
			return fmt.Errorf("node %s has no incoming connections", node.ID)
		}
	}

	return nil
}
