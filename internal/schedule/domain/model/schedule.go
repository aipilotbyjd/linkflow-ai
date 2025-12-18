package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// ScheduleID represents a unique schedule identifier
type ScheduleID string

// NewScheduleID creates a new schedule ID
func NewScheduleID() ScheduleID {
	return ScheduleID(uuid.New().String())
}

func (id ScheduleID) String() string {
	return string(id)
}

// ScheduleStatus represents the status of a schedule
type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
	ScheduleStatusExpired  ScheduleStatus = "expired"
	ScheduleStatusError    ScheduleStatus = "error"
)

// Schedule aggregate root
type Schedule struct {
	id               ScheduleID
	userID           string
	organizationID   string
	workflowID       string
	name             string
	description      string
	cronExpression   string
	timezone         string
	startDate        *time.Time
	endDate          *time.Time
	status           ScheduleStatus
	lastRunAt        *time.Time
	nextRunAt        *time.Time
	runCount         int64
	successCount     int64
	failureCount     int64
	lastError        string
	inputData        map[string]interface{}
	metadata         map[string]interface{}
	cronSchedule     cron.Schedule
	createdAt        time.Time
	updatedAt        time.Time
	version          int
}

// NewSchedule creates a new schedule
func NewSchedule(
	userID string,
	workflowID string,
	name string,
	cronExpression string,
	timezone string,
) (*Schedule, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}
	if workflowID == "" {
		return nil, errors.New("workflow ID is required")
	}
	if name == "" {
		return nil, errors.New("schedule name is required")
	}
	if cronExpression == "" {
		return nil, errors.New("cron expression is required")
	}

	// Parse and validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(cronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Validate timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	now := time.Now()
	schedule := &Schedule{
		id:             NewScheduleID(),
		userID:         userID,
		workflowID:     workflowID,
		name:           name,
		cronExpression: cronExpression,
		timezone:       timezone,
		status:         ScheduleStatusActive,
		runCount:       0,
		successCount:   0,
		failureCount:   0,
		inputData:      make(map[string]interface{}),
		metadata:       make(map[string]interface{}),
		cronSchedule:   cronSchedule,
		createdAt:      now,
		updatedAt:      now,
		version:        0,
	}

	// Calculate next run time
	schedule.nextRunAt = schedule.calculateNextRun(loc)

	return schedule, nil
}

// Getters
func (s *Schedule) ID() ScheduleID                     { return s.id }
func (s *Schedule) UserID() string                     { return s.userID }
func (s *Schedule) OrganizationID() string             { return s.organizationID }
func (s *Schedule) WorkflowID() string                 { return s.workflowID }
func (s *Schedule) Name() string                       { return s.name }
func (s *Schedule) Description() string                { return s.description }
func (s *Schedule) CronExpression() string             { return s.cronExpression }
func (s *Schedule) Timezone() string                   { return s.timezone }
func (s *Schedule) StartDate() *time.Time              { return s.startDate }
func (s *Schedule) EndDate() *time.Time                { return s.endDate }
func (s *Schedule) Status() ScheduleStatus             { return s.status }
func (s *Schedule) LastRunAt() *time.Time              { return s.lastRunAt }
func (s *Schedule) NextRunAt() *time.Time              { return s.nextRunAt }
func (s *Schedule) RunCount() int64                    { return s.runCount }
func (s *Schedule) SuccessCount() int64                { return s.successCount }
func (s *Schedule) FailureCount() int64                { return s.failureCount }
func (s *Schedule) LastError() string                  { return s.lastError }
func (s *Schedule) InputData() map[string]interface{}  { return s.inputData }
func (s *Schedule) Metadata() map[string]interface{}   { return s.metadata }
func (s *Schedule) CreatedAt() time.Time               { return s.createdAt }
func (s *Schedule) UpdatedAt() time.Time               { return s.updatedAt }
func (s *Schedule) Version() int                       { return s.version }

// SetOrganizationID sets the organization ID
func (s *Schedule) SetOrganizationID(orgID string) {
	s.organizationID = orgID
	s.updatedAt = time.Now()
	s.version++
}

// SetDescription sets the description
func (s *Schedule) SetDescription(description string) {
	s.description = description
	s.updatedAt = time.Now()
	s.version++
}

// SetInputData sets the input data for the workflow execution
func (s *Schedule) SetInputData(data map[string]interface{}) {
	s.inputData = data
	s.updatedAt = time.Now()
	s.version++
}

// SetDateRange sets the start and end dates for the schedule
func (s *Schedule) SetDateRange(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil && startDate.After(*endDate) {
		return errors.New("start date must be before end date")
	}

	s.startDate = startDate
	s.endDate = endDate
	
	// Check if schedule has expired
	if endDate != nil && time.Now().After(*endDate) {
		s.status = ScheduleStatusExpired
	}

	s.updatedAt = time.Now()
	s.version++
	return nil
}

// UpdateCronExpression updates the cron expression
func (s *Schedule) UpdateCronExpression(cronExpression string) error {
	// Parse and validate new cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, err := parser.Parse(cronExpression)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.cronExpression = cronExpression
	s.cronSchedule = cronSchedule

	// Recalculate next run time
	loc, _ := time.LoadLocation(s.timezone)
	s.nextRunAt = s.calculateNextRun(loc)

	s.updatedAt = time.Now()
	s.version++
	return nil
}

// UpdateTimezone updates the timezone
func (s *Schedule) UpdateTimezone(timezone string) error {
	// Validate timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	s.timezone = timezone

	// Recalculate next run time
	s.nextRunAt = s.calculateNextRun(loc)

	s.updatedAt = time.Now()
	s.version++
	return nil
}

// Activate activates the schedule
func (s *Schedule) Activate() error {
	if s.status == ScheduleStatusExpired {
		return errors.New("cannot activate expired schedule")
	}
	if s.endDate != nil && time.Now().After(*s.endDate) {
		return errors.New("schedule has already ended")
	}

	s.status = ScheduleStatusActive
	
	// Recalculate next run time
	loc, _ := time.LoadLocation(s.timezone)
	s.nextRunAt = s.calculateNextRun(loc)

	s.updatedAt = time.Now()
	s.version++
	return nil
}

// Pause pauses the schedule
func (s *Schedule) Pause() error {
	if s.status != ScheduleStatusActive {
		return fmt.Errorf("cannot pause schedule in status %s", s.status)
	}

	s.status = ScheduleStatusPaused
	s.updatedAt = time.Now()
	s.version++
	return nil
}

// Resume resumes a paused schedule
func (s *Schedule) Resume() error {
	if s.status != ScheduleStatusPaused {
		return fmt.Errorf("cannot resume schedule in status %s", s.status)
	}

	s.status = ScheduleStatusActive
	
	// Recalculate next run time
	loc, _ := time.LoadLocation(s.timezone)
	s.nextRunAt = s.calculateNextRun(loc)

	s.updatedAt = time.Now()
	s.version++
	return nil
}

// Disable disables the schedule
func (s *Schedule) Disable() {
	s.status = ScheduleStatusDisabled
	s.nextRunAt = nil
	s.updatedAt = time.Now()
	s.version++
}

// RecordRun records a run attempt
func (s *Schedule) RecordRun(success bool, errorMessage string) {
	now := time.Now()
	s.lastRunAt = &now
	s.runCount++
	
	if success {
		s.successCount++
		s.lastError = ""
	} else {
		s.failureCount++
		s.lastError = errorMessage
		
		// If too many failures, mark as error
		failureRate := float64(s.failureCount) / float64(s.runCount)
		if s.runCount > 10 && failureRate > 0.5 {
			s.status = ScheduleStatusError
		}
	}

	// Calculate next run time if still active
	if s.status == ScheduleStatusActive {
		loc, _ := time.LoadLocation(s.timezone)
		s.nextRunAt = s.calculateNextRun(loc)
	}

	// Check if schedule has expired
	if s.endDate != nil && time.Now().After(*s.endDate) {
		s.status = ScheduleStatusExpired
		s.nextRunAt = nil
	}

	s.updatedAt = time.Now()
	s.version++
}

// ShouldRun checks if the schedule should run at the given time
func (s *Schedule) ShouldRun(checkTime time.Time) bool {
	// Check status
	if s.status != ScheduleStatusActive {
		return false
	}

	// Check date range
	if s.startDate != nil && checkTime.Before(*s.startDate) {
		return false
	}
	if s.endDate != nil && checkTime.After(*s.endDate) {
		return false
	}

	// Check if it's time to run
	if s.nextRunAt != nil && checkTime.After(*s.nextRunAt) {
		return true
	}

	return false
}

// calculateNextRun calculates the next run time
func (s *Schedule) calculateNextRun(loc *time.Location) *time.Time {
	if s.status != ScheduleStatusActive {
		return nil
	}

	now := time.Now().In(loc)
	
	// Consider start date if set and in future
	fromTime := now
	if s.startDate != nil && s.startDate.After(now) {
		fromTime = *s.startDate
	}

	next := s.cronSchedule.Next(fromTime)
	
	// Check if next run is within end date
	if s.endDate != nil && next.After(*s.endDate) {
		return nil
	}

	return &next
}

// Validate validates the schedule
func (s *Schedule) Validate() error {
	if s.userID == "" {
		return errors.New("user ID is required")
	}
	if s.workflowID == "" {
		return errors.New("workflow ID is required")
	}
	if s.name == "" {
		return errors.New("schedule name is required")
	}
	if s.cronExpression == "" {
		return errors.New("cron expression is required")
	}
	if s.timezone == "" {
		return errors.New("timezone is required")
	}

	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(s.cronExpression); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Validate timezone
	if _, err := time.LoadLocation(s.timezone); err != nil {
		return fmt.Errorf("invalid timezone: %w", err)
	}

	// Validate date range
	if s.startDate != nil && s.endDate != nil && s.startDate.After(*s.endDate) {
		return errors.New("start date must be before end date")
	}

	return nil
}
