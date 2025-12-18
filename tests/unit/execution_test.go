package unit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "pending"
	ExecutionRunning   ExecutionStatus = "running"
	ExecutionCompleted ExecutionStatus = "completed"
	ExecutionFailed    ExecutionStatus = "failed"
	ExecutionCancelled ExecutionStatus = "cancelled"
)

type Execution struct {
	ID             string
	WorkflowID     string
	Status         ExecutionStatus
	TriggerType    string
	InputData      map[string]interface{}
	OutputData     map[string]interface{}
	StartedAt      *time.Time
	CompletedAt    *time.Time
	Duration       int64
	Error          string
	NodeExecutions []NodeExecution
	RetryCount     int
	MaxRetries     int
}

type NodeExecution struct {
	ID          string
	NodeID      string
	NodeName    string
	Status      ExecutionStatus
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    int64
	InputData   map[string]interface{}
	OutputData  map[string]interface{}
	Error       string
}

func NewExecution(workflowID, triggerType string, inputData map[string]interface{}) *Execution {
	return &Execution{
		ID:             "exec-" + time.Now().Format("20060102150405"),
		WorkflowID:     workflowID,
		Status:         ExecutionPending,
		TriggerType:    triggerType,
		InputData:      inputData,
		NodeExecutions: []NodeExecution{},
		MaxRetries:     3,
	}
}

func (e *Execution) Start() error {
	if e.Status != ExecutionPending {
		return assert.AnError
	}
	now := time.Now()
	e.Status = ExecutionRunning
	e.StartedAt = &now
	return nil
}

func (e *Execution) Complete(outputData map[string]interface{}) error {
	if e.Status != ExecutionRunning {
		return assert.AnError
	}
	now := time.Now()
	e.Status = ExecutionCompleted
	e.CompletedAt = &now
	e.OutputData = outputData
	if e.StartedAt != nil {
		e.Duration = now.Sub(*e.StartedAt).Milliseconds()
	}
	return nil
}

func (e *Execution) Fail(err string) error {
	if e.Status != ExecutionRunning {
		return assert.AnError
	}
	now := time.Now()
	e.Status = ExecutionFailed
	e.CompletedAt = &now
	e.Error = err
	if e.StartedAt != nil {
		e.Duration = now.Sub(*e.StartedAt).Milliseconds()
	}
	return nil
}

func (e *Execution) Cancel() error {
	if e.Status != ExecutionPending && e.Status != ExecutionRunning {
		return assert.AnError
	}
	now := time.Now()
	e.Status = ExecutionCancelled
	e.CompletedAt = &now
	return nil
}

func (e *Execution) CanRetry() bool {
	return e.Status == ExecutionFailed && e.RetryCount < e.MaxRetries
}

func (e *Execution) Retry() (*Execution, error) {
	if !e.CanRetry() {
		return nil, assert.AnError
	}
	retry := NewExecution(e.WorkflowID, e.TriggerType, e.InputData)
	retry.RetryCount = e.RetryCount + 1
	return retry, nil
}

func (e *Execution) AddNodeExecution(nodeExec NodeExecution) {
	e.NodeExecutions = append(e.NodeExecutions, nodeExec)
}

func (e *Execution) GetProgress() float64 {
	if len(e.NodeExecutions) == 0 {
		return 0
	}
	completed := 0
	for _, ne := range e.NodeExecutions {
		if ne.Status == ExecutionCompleted || ne.Status == ExecutionFailed {
			completed++
		}
	}
	return float64(completed) / float64(len(e.NodeExecutions)) * 100
}

// Tests
func TestNewExecution(t *testing.T) {
	t.Run("creates execution with valid inputs", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", map[string]interface{}{"key": "value"})
		require.NotNil(t, exec)
		assert.NotEmpty(t, exec.ID)
		assert.Equal(t, "wf-001", exec.WorkflowID)
		assert.Equal(t, ExecutionPending, exec.Status)
		assert.Equal(t, "manual", exec.TriggerType)
		assert.Equal(t, 3, exec.MaxRetries)
	})
}

func TestExecution_Start(t *testing.T) {
	t.Run("starts pending execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		err := exec.Start()
		assert.NoError(t, err)
		assert.Equal(t, ExecutionRunning, exec.Status)
		assert.NotNil(t, exec.StartedAt)
	})

	t.Run("fails to start non-pending execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		exec.Status = ExecutionRunning
		err := exec.Start()
		assert.Error(t, err)
	})
}

func TestExecution_Complete(t *testing.T) {
	t.Run("completes running execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		_ = exec.Start()
		time.Sleep(10 * time.Millisecond)

		output := map[string]interface{}{"result": "success"}
		err := exec.Complete(output)
		assert.NoError(t, err)
		assert.Equal(t, ExecutionCompleted, exec.Status)
		assert.NotNil(t, exec.CompletedAt)
		assert.Equal(t, output, exec.OutputData)
		assert.Greater(t, exec.Duration, int64(0))
	})

	t.Run("fails to complete non-running execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		err := exec.Complete(nil)
		assert.Error(t, err)
	})
}

func TestExecution_Fail(t *testing.T) {
	t.Run("fails running execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		_ = exec.Start()

		err := exec.Fail("HTTP request failed")
		assert.NoError(t, err)
		assert.Equal(t, ExecutionFailed, exec.Status)
		assert.Equal(t, "HTTP request failed", exec.Error)
	})
}

func TestExecution_Cancel(t *testing.T) {
	t.Run("cancels pending execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		err := exec.Cancel()
		assert.NoError(t, err)
		assert.Equal(t, ExecutionCancelled, exec.Status)
	})

	t.Run("cancels running execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		_ = exec.Start()
		err := exec.Cancel()
		assert.NoError(t, err)
		assert.Equal(t, ExecutionCancelled, exec.Status)
	})

	t.Run("fails to cancel completed execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		exec.Status = ExecutionCompleted
		err := exec.Cancel()
		assert.Error(t, err)
	})
}

func TestExecution_Retry(t *testing.T) {
	t.Run("retries failed execution", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", map[string]interface{}{"key": "value"})
		_ = exec.Start()
		_ = exec.Fail("error")

		retry, err := exec.Retry()
		assert.NoError(t, err)
		assert.NotNil(t, retry)
		assert.Equal(t, exec.WorkflowID, retry.WorkflowID)
		assert.Equal(t, 1, retry.RetryCount)
		assert.Equal(t, ExecutionPending, retry.Status)
	})

	t.Run("fails when max retries reached", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		exec.Status = ExecutionFailed
		exec.RetryCount = 3

		retry, err := exec.Retry()
		assert.Error(t, err)
		assert.Nil(t, retry)
	})
}

func TestExecution_GetProgress(t *testing.T) {
	t.Run("returns 0 for empty executions", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		assert.Equal(t, float64(0), exec.GetProgress())
	})

	t.Run("calculates progress correctly", func(t *testing.T) {
		exec := NewExecution("wf-001", "manual", nil)
		exec.AddNodeExecution(NodeExecution{ID: "ne-1", Status: ExecutionCompleted})
		exec.AddNodeExecution(NodeExecution{ID: "ne-2", Status: ExecutionRunning})
		exec.AddNodeExecution(NodeExecution{ID: "ne-3", Status: ExecutionPending})
		exec.AddNodeExecution(NodeExecution{ID: "ne-4", Status: ExecutionCompleted})

		progress := exec.GetProgress()
		assert.Equal(t, float64(50), progress)
	})
}
