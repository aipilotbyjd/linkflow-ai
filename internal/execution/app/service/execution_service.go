package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/repository"
	executorService "github.com/linkflow-ai/linkflow-ai/internal/execution/domain/service"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

var (
	ErrExecutionNotFound = errors.New("execution not found")
	ErrWorkflowNotFound  = errors.New("workflow not found")
	ErrUnauthorized      = errors.New("unauthorized")
)

// ExecutionService handles execution application logic
type ExecutionService struct {
	executionRepo    repository.ExecutionRepository
	executor         *executorService.WorkflowExecutor
	eventPublisher   *kafka.EventPublisher
	cache            *cache.RedisCache
	logger           logger.Logger
}

// NewExecutionService creates a new execution service
func NewExecutionService(
	executionRepo repository.ExecutionRepository,
	executor *executorService.WorkflowExecutor,
	eventPublisher *kafka.EventPublisher,
	cache *cache.RedisCache,
	logger logger.Logger,
) *ExecutionService {
	return &ExecutionService{
		executionRepo:  executionRepo,
		executor:       executor,
		eventPublisher: eventPublisher,
		cache:          cache,
		logger:         logger,
	}
}

// StartExecutionCommand represents a command to start an execution
type StartExecutionCommand struct {
	WorkflowID  string
	UserID      string
	TriggerType string
	InputData   map[string]interface{}
}

// StartExecution starts a new execution
func (s *ExecutionService) StartExecution(ctx context.Context, cmd StartExecutionCommand) (*model.Execution, error) {
	// Get workflow from cache or repository
	// TODO: Need to inject workflow repository or fetch via API
	
	// Create execution
	triggerType := model.TriggerType(cmd.TriggerType)
	if triggerType == "" {
		triggerType = model.TriggerTypeManual
	}

	execution, err := model.NewExecution(
		cmd.WorkflowID,
		1, // TODO: Get actual workflow version
		cmd.UserID,
		triggerType,
		cmd.InputData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Save execution
	if err := s.executionRepo.Save(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Publish execution started event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   execution.ID().String(),
			AggregateType: "Execution",
			Type:          events.ExecutionStarted,
			Timestamp:     time.Now(),
			UserID:        cmd.UserID,
		}
		if err := s.eventPublisher.Publish(ctx, event); err != nil {
			s.logger.Error("Failed to publish execution started event", "error", err)
		}
	}

	// Start async execution
	go s.executeAsync(context.Background(), execution)

	s.logger.Info("Execution started", 
		"execution_id", execution.ID(),
		"workflow_id", cmd.WorkflowID,
		"user_id", cmd.UserID,
	)

	return execution, nil
}

// ExecuteWorkflow executes a workflow synchronously
func (s *ExecutionService) ExecuteWorkflow(ctx context.Context, workflowID, userID string, inputData map[string]interface{}) (*model.Execution, error) {
	// Create execution
	execution, err := model.NewExecution(
		workflowID,
		1, // TODO: Get actual workflow version
		userID,
		model.TriggerTypeAPI,
		inputData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Save execution
	if err := s.executionRepo.Save(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// Execute synchronously (simplified - in production would need workflow data)
	if err := execution.Start(); err != nil {
		return nil, fmt.Errorf("failed to start execution: %w", err)
	}

	// Simulate execution
	time.Sleep(100 * time.Millisecond)
	
	// Complete execution
	outputData := map[string]interface{}{
		"status": "completed",
		"message": "Workflow executed successfully",
		"timestamp": time.Now(),
	}
	
	if err := execution.Complete(outputData); err != nil {
		return nil, fmt.Errorf("failed to complete execution: %w", err)
	}

	// Update execution
	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to update execution: %w", err)
	}

	return execution, nil
}

// GetExecution gets an execution by ID
func (s *ExecutionService) GetExecution(ctx context.Context, executionID model.ExecutionID) (*model.Execution, error) {
	// Try cache first
	if s.cache != nil {
		var execution model.Execution
		cacheKey := fmt.Sprintf("execution:%s", executionID)
		if err := s.cache.Get(ctx, cacheKey, &execution); err == nil {
			return &execution, nil
		}
	}

	// Get from repository
	execution, err := s.executionRepo.FindByID(ctx, executionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrExecutionNotFound
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	// Cache the result
	if s.cache != nil {
		cacheKey := fmt.Sprintf("execution:%s", executionID)
		_ = s.cache.Set(ctx, cacheKey, execution, 1*time.Minute)
	}

	return execution, nil
}

// ListExecutionsQuery represents a query to list executions
type ListExecutionsQuery struct {
	UserID     string
	WorkflowID string
	Status     string
	Offset     int
	Limit      int
}

// ListExecutions lists executions
func (s *ExecutionService) ListExecutions(ctx context.Context, query ListExecutionsQuery) ([]*model.Execution, int64, error) {
	executions, err := s.executionRepo.FindByUserID(ctx, query.UserID, query.Offset, query.Limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}

	total, err := s.executionRepo.CountByUserID(ctx, query.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return executions, total, nil
}

// CancelExecution cancels a running execution
func (s *ExecutionService) CancelExecution(ctx context.Context, executionID model.ExecutionID) error {
	execution, err := s.executionRepo.FindByID(ctx, executionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrExecutionNotFound
		}
		return fmt.Errorf("failed to get execution: %w", err)
	}

	if err := execution.Cancel(); err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Publish cancellation event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   execution.ID().String(),
			AggregateType: "Execution",
			Type:          events.ExecutionCancelled,
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("execution:%s", executionID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	s.logger.Info("Execution cancelled", "execution_id", executionID)
	return nil
}

// PauseExecution pauses a running execution
func (s *ExecutionService) PauseExecution(ctx context.Context, executionID model.ExecutionID) error {
	execution, err := s.executionRepo.FindByID(ctx, executionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrExecutionNotFound
		}
		return fmt.Errorf("failed to get execution: %w", err)
	}

	if err := execution.Pause(); err != nil {
		return fmt.Errorf("failed to pause execution: %w", err)
	}

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("execution:%s", executionID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	s.logger.Info("Execution paused", "execution_id", executionID)
	return nil
}

// ResumeExecution resumes a paused execution
func (s *ExecutionService) ResumeExecution(ctx context.Context, executionID model.ExecutionID) error {
	execution, err := s.executionRepo.FindByID(ctx, executionID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrExecutionNotFound
		}
		return fmt.Errorf("failed to get execution: %w", err)
	}

	if err := execution.Resume(); err != nil {
		return fmt.Errorf("failed to resume execution: %w", err)
	}

	if err := s.executionRepo.Update(ctx, execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Resume async execution
	go s.executeAsync(context.Background(), execution)

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("execution:%s", executionID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	s.logger.Info("Execution resumed", "execution_id", executionID)
	return nil
}

// executeAsync executes a workflow asynchronously
func (s *ExecutionService) executeAsync(ctx context.Context, execution *model.Execution) {
	// TODO: Implement actual workflow execution
	// This would fetch the workflow and execute it using the executor
	
	// Simulate execution
	if err := execution.Start(); err != nil {
		s.logger.Error("Failed to start execution", "error", err, "execution_id", execution.ID())
		return
	}

	// Update execution
	if err := s.executionRepo.Update(ctx, execution); err != nil {
		s.logger.Error("Failed to update execution", "error", err, "execution_id", execution.ID())
	}

	// Simulate some work
	time.Sleep(2 * time.Second)

	// Complete execution
	outputData := map[string]interface{}{
		"status": "completed",
		"timestamp": time.Now(),
	}
	
	if err := execution.Complete(outputData); err != nil {
		s.logger.Error("Failed to complete execution", "error", err, "execution_id", execution.ID())
		return
	}

	// Update final state
	if err := s.executionRepo.Update(ctx, execution); err != nil {
		s.logger.Error("Failed to update completed execution", "error", err, "execution_id", execution.ID())
	}

	// Publish completion event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   execution.ID().String(),
			AggregateType: "Execution",
			Type:          events.ExecutionCompleted,
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Execution completed", 
		"execution_id", execution.ID(),
		"duration_ms", execution.DurationMs(),
	)
}
