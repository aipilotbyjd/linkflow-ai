package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/logger"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

// Scheduler manages cron job scheduling
type Scheduler struct {
	cron      *cron.Cron
	service   *ScheduleService
	logger    logger.Logger
	jobs      map[model.ScheduleID]cron.EntryID
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
}

// NewScheduler creates a new scheduler
func NewScheduler(service *ScheduleService, logger logger.Logger) *Scheduler {
	location, _ := time.LoadLocation("UTC")
	c := cron.New(
		cron.WithLocation(location),
		cron.WithSeconds(),
	)
	
	return &Scheduler{
		cron:     c,
		service:  service,
		logger:   logger,
		jobs:     make(map[model.ScheduleID]cron.EntryID),
		stopChan: make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.running {
		return nil
	}
	
	// Load active schedules from database
	if err := s.loadActiveSchedules(ctx); err != nil {
		return err
	}
	
	// Start cron scheduler
	s.cron.Start()
	s.running = true
	
	// Start periodic check for due schedules
	go s.periodicCheck(ctx)
	
	s.logger.Info("Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return nil
	}
	
	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()
	
	// Stop periodic check
	close(s.stopChan)
	
	s.running = false
	s.logger.Info("Scheduler stopped")
	return nil
}

// RegisterSchedule registers a schedule with the cron scheduler
func (s *Scheduler) RegisterSchedule(schedule *model.Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.running {
		return
	}
	
	// Remove existing job if present
	if entryID, exists := s.jobs[schedule.ID()]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, schedule.ID())
	}
	
	// Only register active schedules
	if schedule.Status() != model.ScheduleStatusActive {
		return
	}
	
	// Add new cron job
	entryID, err := s.cron.AddFunc(schedule.CronExpression(), func() {
		ctx := context.Background()
		s.ExecuteSchedule(ctx, schedule)
	})
	
	if err != nil {
		s.logger.Error("Failed to register schedule", 
			"schedule_id", schedule.ID(),
			"error", err,
		)
		return
	}
	
	s.jobs[schedule.ID()] = entryID
	s.logger.Debug("Schedule registered", 
		"schedule_id", schedule.ID(),
		"cron", schedule.CronExpression(),
	)
}

// UpdateSchedule updates a registered schedule
func (s *Scheduler) UpdateSchedule(schedule *model.Schedule) {
	// Re-register the schedule (will remove old and add new)
	s.RegisterSchedule(schedule)
}

// PauseSchedule pauses a schedule
func (s *Scheduler) PauseSchedule(scheduleID model.ScheduleID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
		s.logger.Debug("Schedule paused", "schedule_id", scheduleID)
	}
}

// ResumeSchedule resumes a paused schedule
func (s *Scheduler) ResumeSchedule(schedule *model.Schedule) {
	s.RegisterSchedule(schedule)
}

// RemoveSchedule removes a schedule from the scheduler
func (s *Scheduler) RemoveSchedule(scheduleID model.ScheduleID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
		s.logger.Debug("Schedule removed", "schedule_id", scheduleID)
	}
}

// ExecuteSchedule executes a scheduled workflow
func (s *Scheduler) ExecuteSchedule(ctx context.Context, schedule *model.Schedule) {
	s.logger.Info("Executing schedule", 
		"schedule_id", schedule.ID(),
		"workflow_id", schedule.WorkflowID(),
	)
	
	// Publish execution request event
	if s.service.eventPublisher != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"scheduleId":  schedule.ID().String(),
			"workflowId":  schedule.WorkflowID(),
			"userId":      schedule.UserID(),
			"inputData":   schedule.InputData(),
			"triggerType": "schedule",
		})
		
		event := &events.Event{
			AggregateID:   schedule.WorkflowID(),
			AggregateType: "Workflow",
			Type:          events.ExecutionStarted,
			UserID:        schedule.UserID(),
			Timestamp:     time.Now(),
			Data:          json.RawMessage(payload),
		}
		
		if err := s.service.eventPublisher.Publish(ctx, event); err != nil {
			s.logger.Error("Failed to publish execution event", 
				"schedule_id", schedule.ID(),
				"error", err,
			)
			// Record failure
			schedule.RecordRun(false, err.Error())
			_ = s.service.repository.Update(ctx, schedule)
			return
		}
	}
	
	// Record successful trigger
	schedule.RecordRun(true, "")
	if err := s.service.repository.Update(ctx, schedule); err != nil {
		s.logger.Error("Failed to update schedule after execution", 
			"schedule_id", schedule.ID(),
			"error", err,
		)
	}
}

// loadActiveSchedules loads all active schedules from the database
func (s *Scheduler) loadActiveSchedules(ctx context.Context) error {
	// Get all active schedules
	schedules, err := s.service.repository.FindByStatus(ctx, model.ScheduleStatusActive, 0, 10000)
	if err != nil {
		return err
	}
	
	// Register each schedule
	for _, schedule := range schedules {
		s.RegisterSchedule(schedule)
	}
	
	s.logger.Info("Loaded active schedules", "count", len(schedules))
	return nil
}

// periodicCheck periodically checks for due schedules (backup mechanism)
func (s *Scheduler) periodicCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkDueSchedules(ctx)
		}
	}
}

// checkDueSchedules checks for schedules that are due to run
func (s *Scheduler) checkDueSchedules(ctx context.Context) {
	now := time.Now()
	
	// Find schedules that should have run
	schedules, err := s.service.repository.FindDueSchedules(ctx, now, 100)
	if err != nil {
		s.logger.Error("Failed to find due schedules", "error", err)
		return
	}
	
	for _, schedule := range schedules {
		// Check if schedule should run
		if schedule.ShouldRun(now) {
			// Check if it's registered
			s.mu.RLock()
			_, registered := s.jobs[schedule.ID()]
			s.mu.RUnlock()
			
			if !registered {
				// Register and execute
				s.RegisterSchedule(schedule)
				s.ExecuteSchedule(ctx, schedule)
			}
		}
	}
}
