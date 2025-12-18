package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Schedule struct for testing
type Schedule struct {
	ID             string
	WorkflowID     string
	UserID         string
	Name           string
	CronExpression string
	Timezone       string
	Status         ScheduleStatus
	StartDate      *time.Time
	EndDate        *time.Time
	NextRunAt      *time.Time
	LastRunAt      *time.Time
	ExecutionCount int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ScheduleStatus string

const (
	ScheduleStatusActive   ScheduleStatus = "active"
	ScheduleStatusPaused   ScheduleStatus = "paused"
	ScheduleStatusExpired  ScheduleStatus = "expired"
	ScheduleStatusDisabled ScheduleStatus = "disabled"
)

func NewSchedule(workflowID, userID, name, cronExpr, timezone string) (*Schedule, error) {
	if workflowID == "" || userID == "" || name == "" || cronExpr == "" {
		return nil, assert.AnError
	}

	if timezone == "" {
		timezone = "UTC"
	}

	now := time.Now()
	return &Schedule{
		ID:             "schedule-" + name,
		WorkflowID:     workflowID,
		UserID:         userID,
		Name:           name,
		CronExpression: cronExpr,
		Timezone:       timezone,
		Status:         ScheduleStatusActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (s *Schedule) Pause() error {
	if s.Status != ScheduleStatusActive {
		return assert.AnError
	}
	s.Status = ScheduleStatusPaused
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Schedule) Resume() error {
	if s.Status != ScheduleStatusPaused {
		return assert.AnError
	}
	s.Status = ScheduleStatusActive
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Schedule) Disable() {
	s.Status = ScheduleStatusDisabled
	s.UpdatedAt = time.Now()
}

func (s *Schedule) SetDateRange(start, end *time.Time) error {
	if start != nil && end != nil && start.After(*end) {
		return assert.AnError
	}
	s.StartDate = start
	s.EndDate = end
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Schedule) UpdateCronExpression(cronExpr string) error {
	if cronExpr == "" {
		return assert.AnError
	}
	s.CronExpression = cronExpr
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Schedule) RecordExecution() {
	now := time.Now()
	s.LastRunAt = &now
	s.ExecutionCount++
	s.UpdatedAt = now
}

func (s *Schedule) IsWithinDateRange() bool {
	now := time.Now()
	
	if s.StartDate != nil && now.Before(*s.StartDate) {
		return false
	}
	
	if s.EndDate != nil && now.After(*s.EndDate) {
		s.Status = ScheduleStatusExpired
		return false
	}
	
	return true
}

func (s *Schedule) ShouldRun() bool {
	if s.Status != ScheduleStatusActive {
		return false
	}
	return s.IsWithinDateRange()
}

// Tests
func TestNewSchedule(t *testing.T) {
	tests := []struct {
		name       string
		workflowID string
		userID     string
		schedName  string
		cronExpr   string
		timezone   string
		wantErr    bool
	}{
		{
			name:       "valid schedule",
			workflowID: "workflow-123",
			userID:     "user-123",
			schedName:  "Daily Report",
			cronExpr:   "0 9 * * *",
			timezone:   "America/New_York",
			wantErr:    false,
		},
		{
			name:       "default timezone",
			workflowID: "workflow-123",
			userID:     "user-123",
			schedName:  "Hourly Check",
			cronExpr:   "0 * * * *",
			timezone:   "",
			wantErr:    false,
		},
		{
			name:       "empty workflow ID",
			workflowID: "",
			userID:     "user-123",
			schedName:  "Test",
			cronExpr:   "0 * * * *",
			timezone:   "UTC",
			wantErr:    true,
		},
		{
			name:       "empty cron expression",
			workflowID: "workflow-123",
			userID:     "user-123",
			schedName:  "Test",
			cronExpr:   "",
			timezone:   "UTC",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := NewSchedule(tt.workflowID, tt.userID, tt.schedName, tt.cronExpr, tt.timezone)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, schedule)
			} else {
				require.NoError(t, err)
				require.NotNil(t, schedule)

				assert.Equal(t, tt.workflowID, schedule.WorkflowID)
				assert.Equal(t, tt.userID, schedule.UserID)
				assert.Equal(t, tt.schedName, schedule.Name)
				assert.Equal(t, tt.cronExpr, schedule.CronExpression)
				assert.Equal(t, ScheduleStatusActive, schedule.Status)
				
				if tt.timezone == "" {
					assert.Equal(t, "UTC", schedule.Timezone)
				} else {
					assert.Equal(t, tt.timezone, schedule.Timezone)
				}
			}
		})
	}
}

func TestSchedulePauseResume(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	// Initially active
	assert.Equal(t, ScheduleStatusActive, schedule.Status)

	// Pause
	err = schedule.Pause()
	assert.NoError(t, err)
	assert.Equal(t, ScheduleStatusPaused, schedule.Status)

	// Pause again should fail
	err = schedule.Pause()
	assert.Error(t, err)

	// Resume
	err = schedule.Resume()
	assert.NoError(t, err)
	assert.Equal(t, ScheduleStatusActive, schedule.Status)

	// Resume again should fail
	err = schedule.Resume()
	assert.Error(t, err)
}

func TestScheduleDisable(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	schedule.Disable()
	assert.Equal(t, ScheduleStatusDisabled, schedule.Status)
}

func TestScheduleDateRange(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(24 * time.Hour)

	// Valid date range
	err = schedule.SetDateRange(&start, &end)
	assert.NoError(t, err)
	assert.Equal(t, &start, schedule.StartDate)
	assert.Equal(t, &end, schedule.EndDate)

	// Invalid date range (end before start)
	invalidStart := time.Now().Add(48 * time.Hour)
	invalidEnd := time.Now().Add(24 * time.Hour)
	err = schedule.SetDateRange(&invalidStart, &invalidEnd)
	assert.Error(t, err)
}

func TestScheduleIsWithinDateRange(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	// No date range - should return true
	assert.True(t, schedule.IsWithinDateRange())

	// Future start date
	futureStart := time.Now().Add(24 * time.Hour)
	schedule.StartDate = &futureStart
	assert.False(t, schedule.IsWithinDateRange())

	// Past start date, no end date
	pastStart := time.Now().Add(-24 * time.Hour)
	schedule.StartDate = &pastStart
	assert.True(t, schedule.IsWithinDateRange())

	// Past end date
	pastEnd := time.Now().Add(-1 * time.Hour)
	schedule.EndDate = &pastEnd
	assert.False(t, schedule.IsWithinDateRange())
	assert.Equal(t, ScheduleStatusExpired, schedule.Status)
}

func TestScheduleShouldRun(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	// Active schedule should run
	assert.True(t, schedule.ShouldRun())

	// Paused schedule should not run
	schedule.Pause()
	assert.False(t, schedule.ShouldRun())

	// Disabled schedule should not run
	schedule.Resume()
	schedule.Disable()
	assert.False(t, schedule.ShouldRun())
}

func TestScheduleRecordExecution(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	assert.Nil(t, schedule.LastRunAt)
	assert.Equal(t, 0, schedule.ExecutionCount)

	schedule.RecordExecution()
	assert.NotNil(t, schedule.LastRunAt)
	assert.Equal(t, 1, schedule.ExecutionCount)

	schedule.RecordExecution()
	assert.Equal(t, 2, schedule.ExecutionCount)
}

func TestScheduleUpdateCronExpression(t *testing.T) {
	schedule, err := NewSchedule("workflow-123", "user-123", "Test", "0 * * * *", "UTC")
	require.NoError(t, err)

	// Valid update
	err = schedule.UpdateCronExpression("0 0 * * *")
	assert.NoError(t, err)
	assert.Equal(t, "0 0 * * *", schedule.CronExpression)

	// Empty cron expression
	err = schedule.UpdateCronExpression("")
	assert.Error(t, err)
}
