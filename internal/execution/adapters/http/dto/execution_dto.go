package dto

import (
	"errors"
	"time"
)

// StartExecutionRequest represents a request to start an execution
type StartExecutionRequest struct {
	WorkflowID  string                 `json:"workflowId"`
	TriggerType string                 `json:"triggerType"`
	InputData   map[string]interface{} `json:"inputData,omitempty"`
}

// Validate validates the start execution request
func (r *StartExecutionRequest) Validate() error {
	if r.WorkflowID == "" {
		return errors.New("workflow ID is required")
	}
	if r.TriggerType == "" {
		r.TriggerType = "manual"
	}
	return nil
}

// ExecuteWorkflowRequest represents a request to execute a workflow
type ExecuteWorkflowRequest struct {
	InputData map[string]interface{} `json:"inputData,omitempty"`
	Async     bool                   `json:"async,omitempty"`
}

// ExecutionResponse represents an execution response
type ExecutionResponse struct {
	ID              string                            `json:"id"`
	WorkflowID      string                            `json:"workflowId"`
	WorkflowVersion int                               `json:"workflowVersion"`
	UserID          string                            `json:"userId"`
	TriggerType     string                            `json:"triggerType"`
	Status          string                            `json:"status"`
	InputData       map[string]interface{}            `json:"inputData,omitempty"`
	OutputData      map[string]interface{}            `json:"outputData,omitempty"`
	Error           *ExecutionError                   `json:"error,omitempty"`
	NodeExecutions  map[string]NodeExecutionResponse  `json:"nodeExecutions,omitempty"`
	StartedAt       *time.Time                        `json:"startedAt,omitempty"`
	CompletedAt     *time.Time                        `json:"completedAt,omitempty"`
	DurationMs      int64                             `json:"durationMs,omitempty"`
	CreatedAt       time.Time                         `json:"createdAt"`
}

// NodeExecutionResponse represents a node execution response
type NodeExecutionResponse struct {
	NodeID      string          `json:"nodeId"`
	NodeType    string          `json:"nodeType"`
	Status      string          `json:"status"`
	Error       *ExecutionError `json:"error,omitempty"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	DurationMs  int64           `json:"durationMs,omitempty"`
	RetryCount  int             `json:"retryCount"`
}

// ExecutionError represents an execution error
type ExecutionError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ListExecutionsResponse represents a list of executions response
type ListExecutionsResponse struct {
	Items      []ExecutionResponse `json:"items"`
	Total      int64               `json:"total"`
	Pagination Pagination          `json:"pagination"`
}

// Pagination represents pagination information
type Pagination struct {
	Offset int   `json:"offset"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

// ExecutionLog represents a log entry for an execution
type ExecutionLog struct {
	NodeID      string                 `json:"nodeId"`
	NodeType    string                 `json:"nodeType"`
	Status      string                 `json:"status"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	DurationMs  int64                  `json:"durationMs,omitempty"`
	InputData   map[string]interface{} `json:"inputData,omitempty"`
	OutputData  map[string]interface{} `json:"outputData,omitempty"`
	Error       *ExecutionError        `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// ExecutionLogsResponse represents execution logs response
type ExecutionLogsResponse struct {
	ExecutionID string         `json:"executionId"`
	Logs        []ExecutionLog `json:"logs"`
}
