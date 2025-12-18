package repository

import (
	"context"
	"errors"

	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/model"
)

var (
	// ErrNotFound is returned when an execution is not found
	ErrNotFound = errors.New("execution not found")
	
	// ErrOptimisticLocking is returned when optimistic locking fails
	ErrOptimisticLocking = errors.New("optimistic locking failed")
)

// ExecutionRepository defines the interface for execution persistence
type ExecutionRepository interface {
	// Save saves a new execution
	Save(ctx context.Context, execution *model.Execution) error
	
	// Update updates an existing execution
	Update(ctx context.Context, execution *model.Execution) error
	
	// FindByID finds an execution by ID
	FindByID(ctx context.Context, id model.ExecutionID) (*model.Execution, error)
	
	// FindByWorkflowID finds executions by workflow ID
	FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Execution, error)
	
	// FindByUserID finds executions by user ID
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Execution, error)
	
	// FindByStatus finds executions by status
	FindByStatus(ctx context.Context, status model.ExecutionStatus, offset, limit int) ([]*model.Execution, error)
	
	// CountByUserID counts executions for a user
	CountByUserID(ctx context.Context, userID string) (int64, error)
	
	// CountByWorkflowID counts executions for a workflow
	CountByWorkflowID(ctx context.Context, workflowID string) (int64, error)
	
	// Delete deletes an execution
	Delete(ctx context.Context, id model.ExecutionID) error
}
