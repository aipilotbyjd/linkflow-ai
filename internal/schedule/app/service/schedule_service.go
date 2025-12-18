package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/messaging/kafka"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

var (
	ErrScheduleNotFound  = errors.New("schedule not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidSchedule   = errors.New("invalid schedule")
)

// ScheduleService handles schedule application logic
type ScheduleService struct {
	repository      repository.ScheduleRepository
	eventPublisher  *kafka.EventPublisher
	cache           *cache.RedisCache
	logger          logger.Logger
	scheduler       *Scheduler
}

// NewScheduleService creates a new schedule service
func NewScheduleService(
	repository repository.ScheduleRepository,
	eventPublisher *kafka.EventPublisher,
	cache *cache.RedisCache,
	logger logger.Logger,
) *ScheduleService {
	service := &ScheduleService{
		repository:     repository,
		eventPublisher: eventPublisher,
		cache:          cache,
		logger:         logger,
	}
	
	// Initialize scheduler
	service.scheduler = NewScheduler(service, logger)
	
	return service
}

// Start starts the scheduler
func (s *ScheduleService) Start(ctx context.Context) error {
	return s.scheduler.Start(ctx)
}

// Stop stops the scheduler
func (s *ScheduleService) Stop() error {
	return s.scheduler.Stop()
}

// CreateScheduleCommand represents a command to create a schedule
type CreateScheduleCommand struct {
	UserID           string
	OrganizationID   string
	WorkflowID       string
	Name             string
	Description      string
	CronExpression   string
	Timezone         string
	StartDate        *time.Time
	EndDate          *time.Time
	InputData        map[string]interface{}
}

// CreateSchedule creates a new schedule
func (s *ScheduleService) CreateSchedule(ctx context.Context, cmd CreateScheduleCommand) (*model.Schedule, error) {
	// Create schedule
	schedule, err := model.NewSchedule(
		cmd.UserID,
		cmd.WorkflowID,
		cmd.Name,
		cmd.CronExpression,
		cmd.Timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	// Set optional fields
	if cmd.OrganizationID != "" {
		schedule.SetOrganizationID(cmd.OrganizationID)
	}
	if cmd.Description != "" {
		schedule.SetDescription(cmd.Description)
	}
	if cmd.StartDate != nil || cmd.EndDate != nil {
		if err := schedule.SetDateRange(cmd.StartDate, cmd.EndDate); err != nil {
			return nil, err
		}
	}
	if len(cmd.InputData) > 0 {
		schedule.SetInputData(cmd.InputData)
	}

	// Validate schedule
	if err := schedule.Validate(); err != nil {
		return nil, ErrInvalidSchedule
	}

	// Save to repository
	if err := s.repository.Save(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to save schedule: %w", err)
	}

	// Publish event
	if s.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"scheduleId":  schedule.ID().String(),
			"workflowId":  schedule.WorkflowID(),
			"userId":      schedule.UserID(),
			"cron":        schedule.CronExpression(),
			"nextRunAt":   schedule.NextRunAt(),
		})
		event := &events.Event{
			AggregateID:   schedule.ID().String(),
			AggregateType: "Schedule",
			EventType:     "schedule.created",
			UserID:        cmd.UserID,
			Timestamp:     time.Now(),
			Payload:       json.RawMessage(payload),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	// Register with scheduler
	s.scheduler.RegisterSchedule(schedule)

	s.logger.Info("Schedule created", 
		"schedule_id", schedule.ID(),
		"workflow_id", schedule.WorkflowID(),
		"cron", schedule.CronExpression(),
	)

	return schedule, nil
}

// UpdateScheduleCommand represents a command to update a schedule
type UpdateScheduleCommand struct {
	ID               string
	UserID           string
	Name             string
	Description      string
	CronExpression   string
	Timezone         string
	StartDate        *time.Time
	EndDate          *time.Time
	InputData        map[string]interface{}
}

// UpdateSchedule updates a schedule
func (s *ScheduleService) UpdateSchedule(ctx context.Context, cmd UpdateScheduleCommand) (*model.Schedule, error) {
	// Get existing schedule
	schedule, err := s.repository.FindByID(ctx, model.ScheduleID(cmd.ID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("failed to find schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != cmd.UserID {
		return nil, ErrUnauthorized
	}

	// Update fields
	if cmd.CronExpression != "" && cmd.CronExpression != schedule.CronExpression() {
		if err := schedule.UpdateCronExpression(cmd.CronExpression); err != nil {
			return nil, err
		}
	}
	if cmd.Timezone != "" && cmd.Timezone != schedule.Timezone() {
		if err := schedule.UpdateTimezone(cmd.Timezone); err != nil {
			return nil, err
		}
	}
	if cmd.Description != "" {
		schedule.SetDescription(cmd.Description)
	}
	if cmd.StartDate != nil || cmd.EndDate != nil {
		if err := schedule.SetDateRange(cmd.StartDate, cmd.EndDate); err != nil {
			return nil, err
		}
	}
	if len(cmd.InputData) > 0 {
		schedule.SetInputData(cmd.InputData)
	}

	// Update repository
	if err := s.repository.Update(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("schedule:%s", cmd.ID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	// Update scheduler registration
	s.scheduler.UpdateSchedule(schedule)

	// Publish event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   schedule.ID().String(),
			AggregateType: "Schedule",
			EventType:     "schedule.updated",
			UserID:        cmd.UserID,
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Schedule updated", "schedule_id", schedule.ID())
	return schedule, nil
}

// GetSchedule gets a schedule by ID
func (s *ScheduleService) GetSchedule(ctx context.Context, scheduleID model.ScheduleID, userID string) (*model.Schedule, error) {
	// Try cache first
	if s.cache != nil {
		var schedule model.Schedule
		cacheKey := fmt.Sprintf("schedule:%s", scheduleID)
		if err := s.cache.Get(ctx, cacheKey, &schedule); err == nil {
			// Check authorization
			if schedule.UserID() != userID {
				return nil, ErrUnauthorized
			}
			return &schedule, nil
		}
	}

	// Get from repository
	schedule, err := s.repository.FindByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != userID {
		return nil, ErrUnauthorized
	}

	// Cache the result
	if s.cache != nil {
		cacheKey := fmt.Sprintf("schedule:%s", scheduleID)
		_ = s.cache.Set(ctx, cacheKey, schedule, 5*time.Minute)
	}

	return schedule, nil
}

// ListSchedulesQuery represents a query to list schedules
type ListSchedulesQuery struct {
	UserID         string
	OrganizationID string
	WorkflowID     string
	Status         string
	Offset         int
	Limit          int
}

// ListSchedules lists schedules
func (s *ScheduleService) ListSchedules(ctx context.Context, query ListSchedulesQuery) ([]*model.Schedule, int64, error) {
	var schedules []*model.Schedule
	var err error

	if query.WorkflowID != "" {
		schedules, err = s.repository.FindByWorkflowID(ctx, query.WorkflowID, query.Offset, query.Limit)
	} else if query.OrganizationID != "" {
		schedules, err = s.repository.FindByOrganizationID(ctx, query.OrganizationID, query.Offset, query.Limit)
	} else if query.Status != "" {
		schedules, err = s.repository.FindByStatus(ctx, model.ScheduleStatus(query.Status), query.Offset, query.Limit)
	} else {
		schedules, err = s.repository.FindByUserID(ctx, query.UserID, query.Offset, query.Limit)
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list schedules: %w", err)
	}

	// Filter by user authorization
	var authorized []*model.Schedule
	for _, schedule := range schedules {
		if schedule.UserID() == query.UserID {
			authorized = append(authorized, schedule)
		}
	}

	total, err := s.repository.CountByUserID(ctx, query.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	return authorized, total, nil
}

// PauseSchedule pauses a schedule
func (s *ScheduleService) PauseSchedule(ctx context.Context, scheduleID model.ScheduleID, userID string) error {
	schedule, err := s.repository.FindByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("failed to find schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != userID {
		return ErrUnauthorized
	}

	if err := schedule.Pause(); err != nil {
		return err
	}

	if err := s.repository.Update(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Update scheduler
	s.scheduler.PauseSchedule(scheduleID)

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("schedule:%s", scheduleID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	s.logger.Info("Schedule paused", "schedule_id", scheduleID)
	return nil
}

// ResumeSchedule resumes a paused schedule
func (s *ScheduleService) ResumeSchedule(ctx context.Context, scheduleID model.ScheduleID, userID string) error {
	schedule, err := s.repository.FindByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("failed to find schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != userID {
		return ErrUnauthorized
	}

	if err := schedule.Resume(); err != nil {
		return err
	}

	if err := s.repository.Update(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Update scheduler
	s.scheduler.ResumeSchedule(schedule)

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("schedule:%s", scheduleID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	s.logger.Info("Schedule resumed", "schedule_id", scheduleID)
	return nil
}

// DeleteSchedule deletes a schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, scheduleID model.ScheduleID, userID string) error {
	schedule, err := s.repository.FindByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("failed to find schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != userID {
		return ErrUnauthorized
	}

	if err := s.repository.Delete(ctx, scheduleID); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	// Remove from scheduler
	s.scheduler.RemoveSchedule(scheduleID)

	// Invalidate cache
	if s.cache != nil {
		cacheKey := fmt.Sprintf("schedule:%s", scheduleID)
		_ = s.cache.Delete(ctx, cacheKey)
	}

	// Publish event
	if s.eventPublisher != nil {
		event := &events.Event{
			AggregateID:   scheduleID.String(),
			AggregateType: "Schedule",
			EventType:     "schedule.deleted",
			UserID:        userID,
			Timestamp:     time.Now(),
		}
		_ = s.eventPublisher.Publish(ctx, event)
	}

	s.logger.Info("Schedule deleted", "schedule_id", scheduleID)
	return nil
}

// ExecuteSchedule executes a schedule immediately (for testing)
func (s *ScheduleService) ExecuteSchedule(ctx context.Context, scheduleID model.ScheduleID, userID string) error {
	schedule, err := s.repository.FindByID(ctx, scheduleID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("failed to find schedule: %w", err)
	}

	// Check authorization
	if schedule.UserID() != userID {
		return ErrUnauthorized
	}

	// Execute the schedule
	s.scheduler.ExecuteSchedule(ctx, schedule)

	return nil
}
