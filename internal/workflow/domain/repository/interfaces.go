package repository

import (
	"context"
	"github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
)

// WorkflowRepository defines the interface for workflow persistence
type WorkflowRepository interface {
	// Save saves a new workflow
	Save(ctx context.Context, workflow *model.Workflow) error
	
	// FindByID finds a workflow by ID
	FindByID(ctx context.Context, id model.WorkflowID) (*model.Workflow, error)
	
	// FindByUserID finds workflows by user ID
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Workflow, error)
	
	// Update updates an existing workflow
	Update(ctx context.Context, workflow *model.Workflow) error
	
	// Delete deletes a workflow
	Delete(ctx context.Context, id model.WorkflowID) error
	
	// Count counts workflows for a user
	Count(ctx context.Context, userID string) (int64, error)
	
	// FindActive finds all active workflows
	FindActive(ctx context.Context, offset, limit int) ([]*model.Workflow, error)
	
	// ExistsByName checks if a workflow with the given name exists for a user
	ExistsByName(ctx context.Context, userID, name string) (bool, error)
}
