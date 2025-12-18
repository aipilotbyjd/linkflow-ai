// Package engine provides workflow scheduling
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled workflow executions
type Scheduler struct {
	cron       *cron.Cron
	engine     *Engine
	pool       *WorkerPool
	schedules  map[string]*ScheduleEntry
	mu         sync.RWMutex
	repository ScheduleRepository
}

// ScheduleEntry represents a scheduled workflow
type ScheduleEntry struct {
	ID          string
	WorkflowID  string
	CronExpr    string
	Interval    time.Duration
	NextRun     time.Time
	LastRun     *time.Time
	RunCount    int64
	Enabled     bool
	Timezone    *time.Location
	Options     *ExecutionOptions
	EntryID     cron.EntryID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ScheduleRepository defines schedule persistence
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *ScheduleEntry) error
	FindByID(ctx context.Context, id string) (*ScheduleEntry, error)
	FindByWorkflowID(ctx context.Context, workflowID string) ([]*ScheduleEntry, error)
	Update(ctx context.Context, schedule *ScheduleEntry) error
	Delete(ctx context.Context, id string) error
	ListEnabled(ctx context.Context) ([]*ScheduleEntry, error)
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	Timezone        string
	MaxConcurrent   int
	MissedRunPolicy string // skip, catchup
}

// NewScheduler creates a new scheduler
func NewScheduler(engine *Engine, pool *WorkerPool, repo ScheduleRepository, config *SchedulerConfig) *Scheduler {
	location := time.UTC
	if config != nil && config.Timezone != "" {
		if loc, err := time.LoadLocation(config.Timezone); err == nil {
			location = loc
		}
	}

	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(location),
		cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		),
	)

	return &Scheduler{
		cron:       c,
		engine:     engine,
		pool:       pool,
		schedules:  make(map[string]*ScheduleEntry),
		repository: repo,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	// Load existing schedules
	if s.repository != nil {
		schedules, err := s.repository.ListEnabled(ctx)
		if err != nil {
			return fmt.Errorf("failed to load schedules: %w", err)
		}

		for _, schedule := range schedules {
			if err := s.addSchedule(schedule); err != nil {
				// Log error but continue
				continue
			}
		}
	}

	s.cron.Start()
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() context.Context {
	return s.cron.Stop()
}

// CreateSchedule creates a new schedule
func (s *Scheduler) CreateSchedule(ctx context.Context, schedule *ScheduleEntry) error {
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	// Validate cron expression
	if schedule.CronExpr != "" {
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(schedule.CronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// Save to repository
	if s.repository != nil {
		if err := s.repository.Create(ctx, schedule); err != nil {
			return err
		}
	}

	// Add to scheduler if enabled
	if schedule.Enabled {
		if err := s.addSchedule(schedule); err != nil {
			return err
		}
	}

	return nil
}

// UpdateSchedule updates an existing schedule
func (s *Scheduler) UpdateSchedule(ctx context.Context, schedule *ScheduleEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old schedule
	if existing, ok := s.schedules[schedule.ID]; ok {
		s.cron.Remove(existing.EntryID)
		delete(s.schedules, schedule.ID)
	}

	schedule.UpdatedAt = time.Now()

	// Save to repository
	if s.repository != nil {
		if err := s.repository.Update(ctx, schedule); err != nil {
			return err
		}
	}

	// Re-add if enabled
	if schedule.Enabled {
		return s.addSchedule(schedule)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (s *Scheduler) DeleteSchedule(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.schedules[id]; ok {
		s.cron.Remove(existing.EntryID)
		delete(s.schedules, id)
	}

	if s.repository != nil {
		return s.repository.Delete(ctx, id)
	}

	return nil
}

// EnableSchedule enables a schedule
func (s *Scheduler) EnableSchedule(ctx context.Context, id string) error {
	s.mu.Lock()
	schedule, ok := s.schedules[id]
	s.mu.Unlock()

	if !ok {
		// Try to load from repository
		if s.repository != nil {
			var err error
			schedule, err = s.repository.FindByID(ctx, id)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("schedule %s not found", id)
		}
	}

	schedule.Enabled = true
	schedule.UpdatedAt = time.Now()

	if s.repository != nil {
		if err := s.repository.Update(ctx, schedule); err != nil {
			return err
		}
	}

	return s.addSchedule(schedule)
}

// DisableSchedule disables a schedule
func (s *Scheduler) DisableSchedule(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, ok := s.schedules[id]
	if !ok {
		return fmt.Errorf("schedule %s not found", id)
	}

	s.cron.Remove(schedule.EntryID)
	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()
	delete(s.schedules, id)

	if s.repository != nil {
		return s.repository.Update(ctx, schedule)
	}

	return nil
}

// GetSchedule returns a schedule by ID
func (s *Scheduler) GetSchedule(id string) (*ScheduleEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, ok := s.schedules[id]
	if !ok {
		return nil, fmt.Errorf("schedule %s not found", id)
	}

	return schedule, nil
}

// ListSchedules returns all schedules
func (s *Scheduler) ListSchedules() []*ScheduleEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedules := make([]*ScheduleEntry, 0, len(s.schedules))
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	return schedules
}

// ListByWorkflow returns schedules for a workflow
func (s *Scheduler) ListByWorkflow(workflowID string) []*ScheduleEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var schedules []*ScheduleEntry
	for _, schedule := range s.schedules {
		if schedule.WorkflowID == workflowID {
			schedules = append(schedules, schedule)
		}
	}

	return schedules
}

func (s *Scheduler) addSchedule(schedule *ScheduleEntry) error {
	var spec string
	if schedule.CronExpr != "" {
		spec = schedule.CronExpr
	} else if schedule.Interval > 0 {
		spec = fmt.Sprintf("@every %s", schedule.Interval)
	} else {
		return fmt.Errorf("schedule must have cron expression or interval")
	}

	entryID, err := s.cron.AddFunc(spec, func() {
		s.executeScheduledWorkflow(schedule)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	schedule.EntryID = entryID

	// Calculate next run
	entry := s.cron.Entry(entryID)
	schedule.NextRun = entry.Next

	s.mu.Lock()
	s.schedules[schedule.ID] = schedule
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) executeScheduledWorkflow(schedule *ScheduleEntry) {
	ctx := context.Background()

	// Update last run
	now := time.Now()
	schedule.LastRun = &now
	schedule.RunCount++

	// Update next run
	entry := s.cron.Entry(schedule.EntryID)
	schedule.NextRun = entry.Next

	// Set trigger mode
	options := schedule.Options
	if options == nil {
		options = &ExecutionOptions{}
	}
	options.Mode = "schedule"
	options.TriggerData = map[string]interface{}{
		"scheduleId":   schedule.ID,
		"scheduledAt":  now.Format(time.RFC3339),
		"runCount":     schedule.RunCount,
		"cronExpr":     schedule.CronExpr,
	}

	// Submit to worker pool if available
	if s.pool != nil {
		// Need to load workflow - in production, would use repository
		// For now, skip actual execution
		return
	}

	// Update repository
	if s.repository != nil {
		s.repository.Update(ctx, schedule)
	}
}

// TriggerNow manually triggers a scheduled workflow
func (s *Scheduler) TriggerNow(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	schedule, ok := s.schedules[id]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("schedule %s not found", id)
	}

	// Set trigger mode
	options := schedule.Options
	if options == nil {
		options = &ExecutionOptions{}
	}
	options.Mode = "manual"
	options.TriggerData = map[string]interface{}{
		"scheduleId":  schedule.ID,
		"triggeredAt": time.Now().Format(time.RFC3339),
		"manual":      true,
	}

	// Submit to worker pool
	if s.pool != nil {
		// Would need to load workflow from repository
		return "", fmt.Errorf("workflow not loaded")
	}

	return "", nil
}

// GetNextRuns returns the next N scheduled runs
func (s *Scheduler) GetNextRuns(count int) []ScheduleRun {
	entries := s.cron.Entries()
	
	runs := make([]ScheduleRun, 0, count)
	for _, entry := range entries {
		if len(runs) >= count {
			break
		}

		// Find corresponding schedule
		s.mu.RLock()
		for _, schedule := range s.schedules {
			if schedule.EntryID == entry.ID {
				runs = append(runs, ScheduleRun{
					ScheduleID: schedule.ID,
					WorkflowID: schedule.WorkflowID,
					NextRun:    entry.Next,
				})
				break
			}
		}
		s.mu.RUnlock()
	}

	return runs
}

// ScheduleRun represents an upcoming scheduled run
type ScheduleRun struct {
	ScheduleID string
	WorkflowID string
	NextRun    time.Time
}

// InMemoryScheduleRepository is an in-memory schedule repository
type InMemoryScheduleRepository struct {
	schedules map[string]*ScheduleEntry
	mu        sync.RWMutex
}

// NewInMemoryScheduleRepository creates a new in-memory repository
func NewInMemoryScheduleRepository() *InMemoryScheduleRepository {
	return &InMemoryScheduleRepository{
		schedules: make(map[string]*ScheduleEntry),
	}
}

// Create creates a new schedule
func (r *InMemoryScheduleRepository) Create(ctx context.Context, schedule *ScheduleEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schedules[schedule.ID] = schedule
	return nil
}

// FindByID finds a schedule by ID
func (r *InMemoryScheduleRepository) FindByID(ctx context.Context, id string) (*ScheduleEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schedule, ok := r.schedules[id]
	if !ok {
		return nil, fmt.Errorf("schedule %s not found", id)
	}

	return schedule, nil
}

// FindByWorkflowID finds schedules by workflow ID
func (r *InMemoryScheduleRepository) FindByWorkflowID(ctx context.Context, workflowID string) ([]*ScheduleEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var schedules []*ScheduleEntry
	for _, schedule := range r.schedules {
		if schedule.WorkflowID == workflowID {
			schedules = append(schedules, schedule)
		}
	}

	return schedules, nil
}

// Update updates a schedule
func (r *InMemoryScheduleRepository) Update(ctx context.Context, schedule *ScheduleEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.schedules[schedule.ID] = schedule
	return nil
}

// Delete deletes a schedule
func (r *InMemoryScheduleRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.schedules, id)
	return nil
}

// ListEnabled returns all enabled schedules
func (r *InMemoryScheduleRepository) ListEnabled(ctx context.Context) ([]*ScheduleEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var schedules []*ScheduleEntry
	for _, schedule := range r.schedules {
		if schedule.Enabled {
			schedules = append(schedules, schedule)
		}
	}

	return schedules, nil
}
