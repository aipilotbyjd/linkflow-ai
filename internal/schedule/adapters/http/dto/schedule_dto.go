package dto

import (
	"errors"
	"time"
)

// CreateScheduleRequest represents a request to create a schedule
type CreateScheduleRequest struct {
	OrganizationID string                 `json:"organizationId,omitempty"`
	WorkflowID     string                 `json:"workflowId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	CronExpression string                 `json:"cronExpression"`
	Timezone       string                 `json:"timezone"`
	StartDate      string                 `json:"startDate,omitempty"`
	EndDate        string                 `json:"endDate,omitempty"`
	InputData      map[string]interface{} `json:"inputData,omitempty"`
}

// Validate validates the create schedule request
func (r *CreateScheduleRequest) Validate() error {
	if r.WorkflowID == "" {
		return errors.New("workflow ID is required")
	}
	if r.Name == "" {
		return errors.New("schedule name is required")
	}
	if r.CronExpression == "" {
		return errors.New("cron expression is required")
	}
	if r.Timezone == "" {
		r.Timezone = "UTC"
	}
	return nil
}

// UpdateScheduleRequest represents a request to update a schedule
type UpdateScheduleRequest struct {
	Name           string                 `json:"name,omitempty"`
	Description    string                 `json:"description,omitempty"`
	CronExpression string                 `json:"cronExpression,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	StartDate      string                 `json:"startDate,omitempty"`
	EndDate        string                 `json:"endDate,omitempty"`
	InputData      map[string]interface{} `json:"inputData,omitempty"`
}

// ScheduleResponse represents a schedule response
type ScheduleResponse struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"userId"`
	OrganizationID string                 `json:"organizationId,omitempty"`
	WorkflowID     string                 `json:"workflowId"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	CronExpression string                 `json:"cronExpression"`
	Timezone       string                 `json:"timezone"`
	StartDate      *string                `json:"startDate,omitempty"`
	EndDate        *string                `json:"endDate,omitempty"`
	Status         string                 `json:"status"`
	LastRunAt      *time.Time             `json:"lastRunAt,omitempty"`
	NextRunAt      *time.Time             `json:"nextRunAt,omitempty"`
	RunCount       int64                  `json:"runCount"`
	SuccessCount   int64                  `json:"successCount"`
	FailureCount   int64                  `json:"failureCount"`
	LastError      string                 `json:"lastError,omitempty"`
	InputData      map[string]interface{} `json:"inputData,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// ListSchedulesResponse represents a list of schedules response
type ListSchedulesResponse struct {
	Items      []ScheduleResponse `json:"items"`
	Total      int64              `json:"total"`
	Pagination Pagination         `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

// ValidateCronRequest represents a request to validate a cron expression
type ValidateCronRequest struct {
	CronExpression string `json:"cronExpression"`
	Timezone       string `json:"timezone,omitempty"`
}

// ValidateCronResponse represents a validation response
type ValidateCronResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}
