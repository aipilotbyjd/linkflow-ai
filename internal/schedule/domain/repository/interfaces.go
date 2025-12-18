package repository

import (
	"context"
	"errors"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/model"
)

var (
	// ErrNotFound is returned when a schedule is not found
	ErrNotFound = errors.New("schedule not found")
	
	// ErrOptimisticLocking is returned when optimistic locking fails
	ErrOptimisticLocking = errors.New("optimistic locking failed")
)

// ScheduleRepository defines the interface for schedule persistence
type ScheduleRepository interface {
	// Save saves a new schedule
	Save(ctx context.Context, schedule *model.Schedule) error
	
	// Update updates an existing schedule
	Update(ctx context.Context, schedule *model.Schedule) error
	
	// FindByID finds a schedule by ID
	FindByID(ctx context.Context, id model.ScheduleID) (*model.Schedule, error)
	
	// FindByUserID finds schedules by user ID
	FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Schedule, error)
	
	// FindByWorkflowID finds schedules by workflow ID
	FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Schedule, error)
	
	// FindByOrganizationID finds schedules by organization ID
	FindByOrganizationID(ctx context.Context, orgID string, offset, limit int) ([]*model.Schedule, error)
	
	// FindByStatus finds schedules by status
	FindByStatus(ctx context.Context, status model.ScheduleStatus, offset, limit int) ([]*model.Schedule, error)
	
	// FindDueSchedules finds schedules that are due to run
	FindDueSchedules(ctx context.Context, before time.Time, limit int) ([]*model.Schedule, error)
	
	// CountByUserID counts schedules for a user
	CountByUserID(ctx context.Context, userID string) (int64, error)
	
	// CountByWorkflowID counts schedules for a workflow
	CountByWorkflowID(ctx context.Context, workflowID string) (int64, error)
	
	// Delete deletes a schedule
	Delete(ctx context.Context, id model.ScheduleID) error
}
