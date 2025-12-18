package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/service/domainservice"
)

var (
	ErrWorkflowNotFound = errors.New("workflow not found")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrInvalidInput     = errors.New("invalid input")
)

// WorkflowService handles workflow application logic
type WorkflowService struct {
	domainService *domainservice.WorkflowDomainService
	repository    repository.WorkflowRepository
	logger        logger.Logger
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	domainService *domainservice.WorkflowDomainService,
	repository repository.WorkflowRepository,
	logger logger.Logger,
) *WorkflowService {
	return &WorkflowService{
		domainService: domainService,
		repository:    repository,
		logger:        logger,
	}
}

// CreateWorkflowCommand represents a command to create a workflow
type CreateWorkflowCommand struct {
	UserID      string
	Name        string
	Description string
	Nodes       []model.Node
	Connections []model.Connection
}

// CreateWorkflow creates a new workflow
func (s *WorkflowService) CreateWorkflow(ctx context.Context, cmd CreateWorkflowCommand) (*model.Workflow, error) {
	s.logger.Debug("Creating workflow", "user_id", cmd.UserID, "name", cmd.Name)

	// Check if workflow with same name exists
	exists, err := s.repository.ExistsByName(ctx, cmd.UserID, cmd.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check workflow existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("workflow with name '%s' already exists", cmd.Name)
	}

	// Create workflow domain model
	workflow, err := model.NewWorkflow(cmd.UserID, cmd.Name, cmd.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// Add nodes
	for _, node := range cmd.Nodes {
		if err := workflow.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	// Add connections
	for _, conn := range cmd.Connections {
		if err := workflow.AddConnection(conn); err != nil {
			return nil, fmt.Errorf("failed to add connection: %w", err)
		}
	}

	// Save workflow
	if err := s.repository.Save(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to save workflow: %w", err)
	}

	s.logger.Info("Workflow created successfully", "workflow_id", workflow.ID(), "user_id", cmd.UserID)
	return workflow, nil
}

// GetWorkflow gets a workflow by ID
func (s *WorkflowService) GetWorkflow(ctx context.Context, workflowID model.WorkflowID) (*model.Workflow, error) {
	workflow, err := s.repository.FindByID(ctx, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	return workflow, nil
}

// ListWorkflowsQuery represents a query to list workflows
type ListWorkflowsQuery struct {
	UserID string
	Offset int
	Limit  int
	Status string
}

// ListWorkflows lists workflows for a user
func (s *WorkflowService) ListWorkflows(ctx context.Context, query ListWorkflowsQuery) ([]*model.Workflow, int64, error) {
	workflows, err := s.repository.FindByUserID(ctx, query.UserID, query.Offset, query.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workflows: %w", err)
	}

	// Get total count
	total, err := s.repository.Count(ctx, query.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count workflows: %w", err)
	}

	return workflows, total, nil
}

// UpdateWorkflowCommand represents a command to update a workflow
type UpdateWorkflowCommand struct {
	WorkflowID  model.WorkflowID
	Name        string
	Description string
	Nodes       []model.Node
	Connections []model.Connection
}

// UpdateWorkflow updates an existing workflow
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, cmd UpdateWorkflowCommand) (*model.Workflow, error) {
	// Get existing workflow
	workflow, err := s.repository.FindByID(ctx, cmd.WorkflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Update basic fields (would normally have proper domain methods)
	// This is simplified - in real implementation, we'd have proper domain methods
	// to update the workflow while maintaining invariants

	// Clear existing nodes and connections
	for _, node := range workflow.Nodes() {
		if err := workflow.RemoveNode(node.ID); err != nil {
			s.logger.Warn("Failed to remove node", "node_id", node.ID, "error", err)
		}
	}

	// Add new nodes
	for _, node := range cmd.Nodes {
		if err := workflow.AddNode(node); err != nil {
			return nil, fmt.Errorf("failed to add node: %w", err)
		}
	}

	// Add new connections
	for _, conn := range cmd.Connections {
		if err := workflow.AddConnection(conn); err != nil {
			return nil, fmt.Errorf("failed to add connection: %w", err)
		}
	}

	// Update workflow
	if err := s.repository.Update(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	s.logger.Info("Workflow updated successfully", "workflow_id", workflow.ID())
	return workflow, nil
}

// DeleteWorkflow deletes a workflow
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, workflowID model.WorkflowID) error {
	// Check if workflow exists
	workflow, err := s.repository.FindByID(ctx, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrWorkflowNotFound
		}
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Archive the workflow instead of hard delete
	if err := workflow.Archive(); err != nil {
		return fmt.Errorf("failed to archive workflow: %w", err)
	}

	// Update the archived workflow
	if err := s.repository.Update(ctx, workflow); err != nil {
		return fmt.Errorf("failed to update archived workflow: %w", err)
	}

	s.logger.Info("Workflow archived successfully", "workflow_id", workflowID)
	return nil
}

// ActivateWorkflow activates a workflow
func (s *WorkflowService) ActivateWorkflow(ctx context.Context, workflowID model.WorkflowID) (*model.Workflow, error) {
	workflow, err := s.repository.FindByID(ctx, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Activate workflow
	if err := workflow.Activate(); err != nil {
		return nil, fmt.Errorf("failed to activate workflow: %w", err)
	}

	// Update workflow
	if err := s.repository.Update(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	s.logger.Info("Workflow activated successfully", "workflow_id", workflowID)
	return workflow, nil
}

// DeactivateWorkflow deactivates a workflow
func (s *WorkflowService) DeactivateWorkflow(ctx context.Context, workflowID model.WorkflowID) (*model.Workflow, error) {
	workflow, err := s.repository.FindByID(ctx, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrWorkflowNotFound
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	// Deactivate workflow
	if err := workflow.Deactivate(); err != nil {
		return nil, fmt.Errorf("failed to deactivate workflow: %w", err)
	}

	// Update workflow
	if err := s.repository.Update(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	s.logger.Info("Workflow deactivated successfully", "workflow_id", workflowID)
	return workflow, nil
}

// DuplicateWorkflow duplicates an existing workflow
func (s *WorkflowService) DuplicateWorkflow(ctx context.Context, workflowID model.WorkflowID, newName string) (*model.Workflow, error) {
	duplicate, err := s.domainService.DuplicateWorkflow(ctx, workflowID, newName)
	if err != nil {
		return nil, fmt.Errorf("failed to duplicate workflow: %w", err)
	}

	s.logger.Info("Workflow duplicated successfully", 
		"source_id", workflowID, 
		"duplicate_id", duplicate.ID(),
	)
	return duplicate, nil
}
