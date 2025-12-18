package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExecutionID represents a unique execution identifier
type ExecutionID string

// NewExecutionID creates a new execution ID
func NewExecutionID() ExecutionID {
	return ExecutionID(uuid.New().String())
}

func (id ExecutionID) String() string {
	return string(id)
}

// ExecutionStatus represents the status of an execution
type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"
	ExecutionStatusRunning    ExecutionStatus = "running"
	ExecutionStatusCompleted  ExecutionStatus = "completed"
	ExecutionStatusFailed     ExecutionStatus = "failed"
	ExecutionStatusCancelled  ExecutionStatus = "cancelled"
	ExecutionStatusPaused     ExecutionStatus = "paused"
)

// TriggerType represents how the execution was triggered
type TriggerType string

const (
	TriggerTypeManual   TriggerType = "manual"
	TriggerTypeSchedule TriggerType = "schedule"
	TriggerTypeWebhook  TriggerType = "webhook"
	TriggerTypeEvent    TriggerType = "event"
	TriggerTypeAPI      TriggerType = "api"
)

// NodeExecution represents the execution state of a single node
type NodeExecution struct {
	NodeID      string                 `json:"nodeId"`
	NodeType    string                 `json:"nodeType"`
	Status      ExecutionStatus        `json:"status"`
	InputData   map[string]interface{} `json:"inputData"`
	OutputData  map[string]interface{} `json:"outputData"`
	Error       *ExecutionError        `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"startedAt,omitempty"`
	CompletedAt *time.Time             `json:"completedAt,omitempty"`
	DurationMs  int64                  `json:"durationMs"`
	RetryCount  int                    `json:"retryCount"`
}

// ExecutionError represents an error during execution
type ExecutionError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	NodeID  string                 `json:"nodeId,omitempty"`
}

// ExecutionContext holds the context data for the execution
type ExecutionContext struct {
	Variables   map[string]interface{} `json:"variables"`
	Credentials map[string]interface{} `json:"credentials"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Execution aggregate root
type Execution struct {
	id              ExecutionID
	workflowID      string
	workflowVersion int
	userID          string
	triggerType     TriggerType
	triggerID       string
	status          ExecutionStatus
	inputData       map[string]interface{}
	outputData      map[string]interface{}
	context         ExecutionContext
	nodeExecutions  map[string]*NodeExecution
	executionPath   []string
	error           *ExecutionError
	startedAt       *time.Time
	completedAt     *time.Time
	pausedAt        *time.Time
	durationMs      int64
	metadata        map[string]interface{}
	createdAt       time.Time
	updatedAt       time.Time
	version         int
}

// NewExecution creates a new execution
func NewExecution(
	workflowID string,
	workflowVersion int,
	userID string,
	triggerType TriggerType,
	inputData map[string]interface{},
) (*Execution, error) {
	if workflowID == "" {
		return nil, errors.New("workflow ID is required")
	}
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	now := time.Now()
	execution := &Execution{
		id:              NewExecutionID(),
		workflowID:      workflowID,
		workflowVersion: workflowVersion,
		userID:          userID,
		triggerType:     triggerType,
		status:          ExecutionStatusPending,
		inputData:       inputData,
		outputData:      make(map[string]interface{}),
		context: ExecutionContext{
			Variables:   make(map[string]interface{}),
			Credentials: make(map[string]interface{}),
			Metadata:    make(map[string]interface{}),
		},
		nodeExecutions: make(map[string]*NodeExecution),
		executionPath:  []string{},
		metadata:       make(map[string]interface{}),
		createdAt:      now,
		updatedAt:      now,
		version:        0,
	}

	if inputData != nil {
		execution.context.Variables = inputData
	}

	return execution, nil
}

// Getters
func (e *Execution) ID() ExecutionID                     { return e.id }
func (e *Execution) WorkflowID() string                  { return e.workflowID }
func (e *Execution) WorkflowVersion() int                { return e.workflowVersion }
func (e *Execution) UserID() string                      { return e.userID }
func (e *Execution) TriggerType() TriggerType            { return e.triggerType }
func (e *Execution) Status() ExecutionStatus             { return e.status }
func (e *Execution) InputData() map[string]interface{}   { return e.inputData }
func (e *Execution) OutputData() map[string]interface{}  { return e.outputData }
func (e *Execution) Context() ExecutionContext           { return e.context }
func (e *Execution) NodeExecutions() map[string]*NodeExecution { return e.nodeExecutions }
func (e *Execution) ExecutionPath() []string             { return e.executionPath }
func (e *Execution) Error() *ExecutionError              { return e.error }
func (e *Execution) StartedAt() *time.Time               { return e.startedAt }
func (e *Execution) CompletedAt() *time.Time             { return e.completedAt }
func (e *Execution) DurationMs() int64                   { return e.durationMs }
func (e *Execution) CreatedAt() time.Time                { return e.createdAt }
func (e *Execution) UpdatedAt() time.Time                { return e.updatedAt }
func (e *Execution) Version() int                        { return e.version }

// Start starts the execution
func (e *Execution) Start() error {
	if e.status != ExecutionStatusPending {
		return fmt.Errorf("cannot start execution in status %s", e.status)
	}

	now := time.Now()
	e.status = ExecutionStatusRunning
	e.startedAt = &now
	e.updatedAt = now
	e.version++

	return nil
}

// Complete marks the execution as completed
func (e *Execution) Complete(outputData map[string]interface{}) error {
	if e.status != ExecutionStatusRunning {
		return fmt.Errorf("cannot complete execution in status %s", e.status)
	}

	now := time.Now()
	e.status = ExecutionStatusCompleted
	e.completedAt = &now
	e.outputData = outputData
	
	if e.startedAt != nil {
		e.durationMs = now.Sub(*e.startedAt).Milliseconds()
	}
	
	e.updatedAt = now
	e.version++

	return nil
}

// Fail marks the execution as failed
func (e *Execution) Fail(err ExecutionError) error {
	if e.status != ExecutionStatusRunning && e.status != ExecutionStatusPending {
		return fmt.Errorf("cannot fail execution in status %s", e.status)
	}

	now := time.Now()
	e.status = ExecutionStatusFailed
	e.error = &err
	e.completedAt = &now
	
	if e.startedAt != nil {
		e.durationMs = now.Sub(*e.startedAt).Milliseconds()
	}
	
	e.updatedAt = now
	e.version++

	return nil
}

// Cancel cancels the execution
func (e *Execution) Cancel() error {
	if e.status != ExecutionStatusRunning && e.status != ExecutionStatusPaused {
		return fmt.Errorf("cannot cancel execution in status %s", e.status)
	}

	now := time.Now()
	e.status = ExecutionStatusCancelled
	e.completedAt = &now
	
	if e.startedAt != nil {
		e.durationMs = now.Sub(*e.startedAt).Milliseconds()
	}
	
	e.updatedAt = now
	e.version++

	return nil
}

// Pause pauses the execution
func (e *Execution) Pause() error {
	if e.status != ExecutionStatusRunning {
		return fmt.Errorf("cannot pause execution in status %s", e.status)
	}

	now := time.Now()
	e.status = ExecutionStatusPaused
	e.pausedAt = &now
	e.updatedAt = now
	e.version++

	return nil
}

// Resume resumes a paused execution
func (e *Execution) Resume() error {
	if e.status != ExecutionStatusPaused {
		return fmt.Errorf("cannot resume execution in status %s", e.status)
	}

	e.status = ExecutionStatusRunning
	e.pausedAt = nil
	e.updatedAt = time.Now()
	e.version++

	return nil
}

// StartNodeExecution starts the execution of a node
func (e *Execution) StartNodeExecution(nodeID, nodeType string, inputData map[string]interface{}) error {
	if e.status != ExecutionStatusRunning {
		return fmt.Errorf("cannot start node execution when execution is not running")
	}

	now := time.Now()
	nodeExec := &NodeExecution{
		NodeID:     nodeID,
		NodeType:   nodeType,
		Status:     ExecutionStatusRunning,
		InputData:  inputData,
		OutputData: make(map[string]interface{}),
		StartedAt:  &now,
		RetryCount: 0,
	}

	e.nodeExecutions[nodeID] = nodeExec
	e.executionPath = append(e.executionPath, nodeID)
	e.updatedAt = now
	e.version++

	return nil
}

// CompleteNodeExecution completes the execution of a node
func (e *Execution) CompleteNodeExecution(nodeID string, outputData map[string]interface{}) error {
	nodeExec, exists := e.nodeExecutions[nodeID]
	if !exists {
		return fmt.Errorf("node execution %s not found", nodeID)
	}

	if nodeExec.Status != ExecutionStatusRunning {
		return fmt.Errorf("node execution %s is not running", nodeID)
	}

	now := time.Now()
	nodeExec.Status = ExecutionStatusCompleted
	nodeExec.OutputData = outputData
	nodeExec.CompletedAt = &now
	
	if nodeExec.StartedAt != nil {
		nodeExec.DurationMs = now.Sub(*nodeExec.StartedAt).Milliseconds()
	}

	// Update context variables with output
	for key, value := range outputData {
		e.context.Variables[fmt.Sprintf("%s.%s", nodeID, key)] = value
	}

	e.updatedAt = now
	e.version++

	return nil
}

// FailNodeExecution marks a node execution as failed
func (e *Execution) FailNodeExecution(nodeID string, err ExecutionError) error {
	nodeExec, exists := e.nodeExecutions[nodeID]
	if !exists {
		return fmt.Errorf("node execution %s not found", nodeID)
	}

	now := time.Now()
	nodeExec.Status = ExecutionStatusFailed
	nodeExec.Error = &err
	nodeExec.CompletedAt = &now
	
	if nodeExec.StartedAt != nil {
		nodeExec.DurationMs = now.Sub(*nodeExec.StartedAt).Milliseconds()
	}

	e.updatedAt = now
	e.version++

	return nil
}

// RetryNodeExecution increments the retry count for a node
func (e *Execution) RetryNodeExecution(nodeID string) error {
	nodeExec, exists := e.nodeExecutions[nodeID]
	if !exists {
		return fmt.Errorf("node execution %s not found", nodeID)
	}

	nodeExec.RetryCount++
	nodeExec.Status = ExecutionStatusRunning
	nodeExec.Error = nil
	now := time.Now()
	nodeExec.StartedAt = &now
	
	e.updatedAt = now
	e.version++

	return nil
}

// SetVariable sets a variable in the execution context
func (e *Execution) SetVariable(key string, value interface{}) {
	e.context.Variables[key] = value
	e.updatedAt = time.Now()
	e.version++
}

// GetVariable gets a variable from the execution context
func (e *Execution) GetVariable(key string) (interface{}, bool) {
	val, exists := e.context.Variables[key]
	return val, exists
}

// ReconstructExecution reconstructs an execution from persisted state
func ReconstructExecution(
	id ExecutionID,
	workflowID string,
	workflowVersion int,
	userID string,
	triggerType TriggerType,
	triggerID string,
	status ExecutionStatus,
	inputData map[string]interface{},
	outputData map[string]interface{},
	context ExecutionContext,
	nodeExecutions map[string]*NodeExecution,
	executionPath []string,
	error *ExecutionError,
	startedAt *time.Time,
	completedAt *time.Time,
	pausedAt *time.Time,
	durationMs int64,
	metadata map[string]interface{},
	createdAt time.Time,
	updatedAt time.Time,
	version int,
) *Execution {
	return &Execution{
		id:              id,
		workflowID:      workflowID,
		workflowVersion: workflowVersion,
		userID:          userID,
		triggerType:     triggerType,
		triggerID:       triggerID,
		status:          status,
		inputData:       inputData,
		outputData:      outputData,
		context:         context,
		nodeExecutions:  nodeExecutions,
		executionPath:   executionPath,
		error:           error,
		startedAt:       startedAt,
		completedAt:     completedAt,
		pausedAt:        pausedAt,
		durationMs:      durationMs,
		metadata:        metadata,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
		version:         version,
	}
}
