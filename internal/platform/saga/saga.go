// Package saga implements the Saga pattern for distributed transactions
package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// StepStatus represents the status of a saga step
type StepStatus string

const (
	StepPending     StepStatus = "pending"
	StepRunning     StepStatus = "running"
	StepCompleted   StepStatus = "completed"
	StepFailed      StepStatus = "failed"
	StepCompensated StepStatus = "compensated"
)

// SagaStatus represents the overall saga status
type SagaStatus string

const (
	SagaPending      SagaStatus = "pending"
	SagaRunning      SagaStatus = "running"
	SagaCompleted    SagaStatus = "completed"
	SagaFailed       SagaStatus = "failed"
	SagaCompensating SagaStatus = "compensating"
	SagaCompensated  SagaStatus = "compensated"
)

// Step represents a single step in a saga
type Step struct {
	Name       string
	Execute    func(ctx context.Context, data interface{}) (interface{}, error)
	Compensate func(ctx context.Context, data interface{}) error
	Status     StepStatus
	Result     interface{}
	Error      error
	StartedAt  *time.Time
	EndedAt    *time.Time
}

// Saga represents a saga transaction
type Saga struct {
	ID          string
	Name        string
	Steps       []*Step
	Status      SagaStatus
	Data        interface{}
	Error       error
	StartedAt   *time.Time
	CompletedAt *time.Time
	mu          sync.RWMutex
}

// Builder is a saga builder
type Builder struct {
	name  string
	steps []*Step
}

// NewBuilder creates a new saga builder
func NewBuilder(name string) *Builder {
	return &Builder{
		name:  name,
		steps: make([]*Step, 0),
	}
}

// AddStep adds a step to the saga
func (b *Builder) AddStep(name string, execute func(ctx context.Context, data interface{}) (interface{}, error), compensate func(ctx context.Context, data interface{}) error) *Builder {
	b.steps = append(b.steps, &Step{
		Name:       name,
		Execute:    execute,
		Compensate: compensate,
		Status:     StepPending,
	})
	return b
}

// Build creates the saga
func (b *Builder) Build() *Saga {
	return &Saga{
		ID:     fmt.Sprintf("saga-%d", time.Now().UnixNano()),
		Name:   b.name,
		Steps:  b.steps,
		Status: SagaPending,
	}
}

// Execute runs the saga
func (s *Saga) Execute(ctx context.Context, initialData interface{}) error {
	s.mu.Lock()
	s.Status = SagaRunning
	s.Data = initialData
	now := time.Now()
	s.StartedAt = &now
	s.mu.Unlock()

	currentData := initialData
	var lastCompletedStep int = -1

	for i, step := range s.Steps {
		select {
		case <-ctx.Done():
			s.compensate(ctx, lastCompletedStep)
			return ctx.Err()
		default:
		}

		s.mu.Lock()
		step.Status = StepRunning
		stepStart := time.Now()
		step.StartedAt = &stepStart
		s.mu.Unlock()

		result, err := step.Execute(ctx, currentData)

		s.mu.Lock()
		stepEnd := time.Now()
		step.EndedAt = &stepEnd
		s.mu.Unlock()

		if err != nil {
			s.mu.Lock()
			step.Status = StepFailed
			step.Error = err
			s.Error = err
			s.mu.Unlock()

			// Compensate all completed steps
			s.compensate(ctx, lastCompletedStep)
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		s.mu.Lock()
		step.Status = StepCompleted
		step.Result = result
		s.mu.Unlock()

		currentData = result
		lastCompletedStep = i
	}

	s.mu.Lock()
	s.Status = SagaCompleted
	s.Data = currentData
	endTime := time.Now()
	s.CompletedAt = &endTime
	s.mu.Unlock()

	return nil
}

// compensate runs compensation for completed steps in reverse order
func (s *Saga) compensate(ctx context.Context, lastCompletedStep int) {
	s.mu.Lock()
	s.Status = SagaCompensating
	s.mu.Unlock()

	var compensationErrors []error

	// Run compensation in reverse order
	for i := lastCompletedStep; i >= 0; i-- {
		step := s.Steps[i]
		if step.Compensate == nil {
			continue
		}

		err := step.Compensate(ctx, step.Result)
		if err != nil {
			compensationErrors = append(compensationErrors, fmt.Errorf("compensation for %s failed: %w", step.Name, err))
			continue
		}

		s.mu.Lock()
		step.Status = StepCompensated
		s.mu.Unlock()
	}

	s.mu.Lock()
	if len(compensationErrors) > 0 {
		s.Error = errors.Join(compensationErrors...)
	}
	s.Status = SagaCompensated
	s.mu.Unlock()
}

// GetStatus returns the current saga status
func (s *Saga) GetStatus() SagaStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

// GetStepStatus returns status of all steps
func (s *Saga) GetStepStatus() []struct {
	Name   string
	Status StepStatus
} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]struct {
		Name   string
		Status StepStatus
	}, len(s.Steps))

	for i, step := range s.Steps {
		result[i] = struct {
			Name   string
			Status StepStatus
		}{
			Name:   step.Name,
			Status: step.Status,
		}
	}
	return result
}

// Orchestrator manages saga execution
type Orchestrator struct {
	sagas map[string]*Saga
	mu    sync.RWMutex
}

// NewOrchestrator creates a new saga orchestrator
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		sagas: make(map[string]*Saga),
	}
}

// Execute runs a saga and tracks it
func (o *Orchestrator) Execute(ctx context.Context, saga *Saga, data interface{}) error {
	o.mu.Lock()
	o.sagas[saga.ID] = saga
	o.mu.Unlock()

	return saga.Execute(ctx, data)
}

// GetSaga returns a saga by ID
func (o *Orchestrator) GetSaga(id string) (*Saga, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	saga, ok := o.sagas[id]
	return saga, ok
}

// Example: Workflow Execution Saga
func NewWorkflowExecutionSaga(workflowID string) *Saga {
	return NewBuilder("workflow-execution").
		AddStep("validate-workflow",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				// Validate workflow exists and is active
				return data, nil
			},
			nil, // No compensation needed
		).
		AddStep("allocate-resources",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				// Allocate execution resources
				return data, nil
			},
			func(ctx context.Context, data interface{}) error {
				// Release allocated resources
				return nil
			},
		).
		AddStep("start-execution",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				// Start the workflow execution
				return data, nil
			},
			func(ctx context.Context, data interface{}) error {
				// Cancel the execution
				return nil
			},
		).
		AddStep("notify-start",
			func(ctx context.Context, data interface{}) (interface{}, error) {
				// Send execution started notification
				return data, nil
			},
			nil, // Notification doesn't need compensation
		).
		Build()
}
