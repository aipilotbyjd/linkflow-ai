// Package engine provides worker pool for workflow execution
package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// WorkerPool manages a pool of workers for executing tasks
type WorkerPool struct {
	workers       []*Worker
	taskQueue     chan *Task
	resultQueue   chan *TaskResult
	maxWorkers    int
	activeWorkers int32
	mu            sync.RWMutex
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	engine        *Engine
	metrics       *PoolMetrics
}

// Worker represents a single worker in the pool
type Worker struct {
	ID           string
	Status       WorkerStatus
	CurrentTask  *Task
	TasksHandled int64
	StartedAt    time.Time
	LastActiveAt time.Time
}

// WorkerStatus represents worker status
type WorkerStatus string

const (
	WorkerStatusIdle    WorkerStatus = "idle"
	WorkerStatusBusy    WorkerStatus = "busy"
	WorkerStatusStopped WorkerStatus = "stopped"
)

// Task represents a task to be executed by a worker
type Task struct {
	ID           string
	Type         TaskType
	ExecutionID  string
	WorkflowID   string
	NodeID       string
	Workflow     *WorkflowDefinition
	Options      *ExecutionOptions
	Priority     int
	CreatedAt    time.Time
	StartedAt    *time.Time
	Timeout      time.Duration
	RetryCount   int
	MaxRetries   int
	Metadata     map[string]interface{}
}

// TaskType represents the type of task
type TaskType string

const (
	TaskTypeWorkflowExecution TaskType = "workflow_execution"
	TaskTypeNodeExecution     TaskType = "node_execution"
	TaskTypeWebhookTrigger    TaskType = "webhook_trigger"
	TaskTypeScheduleTrigger   TaskType = "schedule_trigger"
)

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID      string
	ExecutionID string
	Status      TaskStatus
	Output      map[string]interface{}
	Error       error
	StartedAt   time.Time
	CompletedAt time.Time
	WorkerID    string
	Logs        []LogEntry
}

// TaskStatus represents task execution status
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRetrying  TaskStatus = "retrying"
)

// LogEntry represents a log entry
type LogEntry struct {
	Level     string
	Message   string
	Timestamp int64
	TaskID    string
	WorkerID  string
}

// PoolMetrics tracks worker pool metrics
type PoolMetrics struct {
	TotalTasks      int64
	CompletedTasks  int64
	FailedTasks     int64
	ActiveTasks     int64
	QueuedTasks     int64
	AvgExecutionMs  int64
	TotalWorkers    int32
	ActiveWorkers   int32
	IdleWorkers     int32
}

// PoolConfig holds worker pool configuration
type PoolConfig struct {
	MaxWorkers     int
	QueueSize      int
	TaskTimeout    time.Duration
	IdleTimeout    time.Duration
	ScaleUpDelay   time.Duration
	ScaleDownDelay time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxWorkers:     10,
		QueueSize:      1000,
		TaskTimeout:    5 * time.Minute,
		IdleTimeout:    30 * time.Second,
		ScaleUpDelay:   5 * time.Second,
		ScaleDownDelay: 30 * time.Second,
	}
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(engine *Engine, config *PoolConfig) *WorkerPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		workers:     make([]*Worker, 0, config.MaxWorkers),
		taskQueue:   make(chan *Task, config.QueueSize),
		resultQueue: make(chan *TaskResult, config.QueueSize),
		maxWorkers:  config.MaxWorkers,
		ctx:         ctx,
		cancel:      cancel,
		engine:      engine,
		metrics:     &PoolMetrics{},
	}

	return pool
}

// Start starts the worker pool
func (p *WorkerPool) Start(numWorkers int) {
	if numWorkers > p.maxWorkers {
		numWorkers = p.maxWorkers
	}

	// Start initial workers
	for i := 0; i < numWorkers; i++ {
		p.addWorker()
	}

	// Start result processor
	go p.processResults()

	// Start metrics collector
	go p.collectMetrics()
}

// Stop stops the worker pool gracefully
func (p *WorkerPool) Stop() {
	p.cancel()

	// Wait for all workers to finish current tasks
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All workers finished
	case <-time.After(30 * time.Second):
		// Timeout waiting for workers
	}

	close(p.taskQueue)
	close(p.resultQueue)
}

// Submit submits a task to the pool
func (p *WorkerPool) Submit(task *Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()

	select {
	case p.taskQueue <- task:
		atomic.AddInt64(&p.metrics.TotalTasks, 1)
		atomic.AddInt64(&p.metrics.QueuedTasks, 1)
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// SubmitWorkflow submits a workflow for execution
func (p *WorkerPool) SubmitWorkflow(workflow *WorkflowDefinition, options *ExecutionOptions) (string, error) {
	executionID := uuid.New().String()

	task := &Task{
		ID:          uuid.New().String(),
		Type:        TaskTypeWorkflowExecution,
		ExecutionID: executionID,
		WorkflowID:  workflow.ID,
		Workflow:    workflow,
		Options:     options,
		Priority:    5, // Normal priority
		Timeout:     5 * time.Minute,
		MaxRetries:  3,
		Metadata:    make(map[string]interface{}),
	}

	if err := p.Submit(task); err != nil {
		return "", err
	}

	return executionID, nil
}

func (p *WorkerPool) addWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	worker := &Worker{
		ID:        uuid.New().String(),
		Status:    WorkerStatusIdle,
		StartedAt: time.Now(),
	}

	p.workers = append(p.workers, worker)
	p.wg.Add(1)

	go p.runWorker(worker)

	atomic.AddInt32(&p.metrics.TotalWorkers, 1)
	atomic.AddInt32(&p.metrics.IdleWorkers, 1)
}

func (p *WorkerPool) runWorker(worker *Worker) {
	defer p.wg.Done()
	defer func() {
		worker.Status = WorkerStatusStopped
		atomic.AddInt32(&p.metrics.TotalWorkers, -1)
	}()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}

			// Update worker status
			worker.Status = WorkerStatusBusy
			worker.CurrentTask = task
			worker.LastActiveAt = time.Now()
			atomic.AddInt32(&p.activeWorkers, 1)
			atomic.AddInt32(&p.metrics.IdleWorkers, -1)
			atomic.AddInt32(&p.metrics.ActiveWorkers, 1)
			atomic.AddInt64(&p.metrics.QueuedTasks, -1)
			atomic.AddInt64(&p.metrics.ActiveTasks, 1)

			// Execute task
			result := p.executeTask(worker, task)

			// Send result
			select {
			case p.resultQueue <- result:
			default:
				// Result queue full, log error
			}

			// Update worker status
			worker.Status = WorkerStatusIdle
			worker.CurrentTask = nil
			worker.TasksHandled++
			atomic.AddInt32(&p.activeWorkers, -1)
			atomic.AddInt32(&p.metrics.IdleWorkers, 1)
			atomic.AddInt32(&p.metrics.ActiveWorkers, -1)
			atomic.AddInt64(&p.metrics.ActiveTasks, -1)
		}
	}
}

func (p *WorkerPool) executeTask(worker *Worker, task *Task) *TaskResult {
	startTime := time.Now()
	now := startTime
	task.StartedAt = &now

	result := &TaskResult{
		TaskID:      task.ID,
		ExecutionID: task.ExecutionID,
		Status:      TaskStatusRunning,
		StartedAt:   startTime,
		WorkerID:    worker.ID,
		Logs:        []LogEntry{},
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(p.ctx, task.Timeout)
	defer cancel()

	result.Logs = append(result.Logs, LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Task %s started by worker %s", task.ID, worker.ID),
		Timestamp: time.Now().UnixMilli(),
		TaskID:    task.ID,
		WorkerID:  worker.ID,
	})

	// Execute based on task type
	var err error
	var output map[string]interface{}

	switch task.Type {
	case TaskTypeWorkflowExecution:
		output, err = p.executeWorkflowTask(ctx, task)
	case TaskTypeNodeExecution:
		output, err = p.executeNodeTask(ctx, task)
	default:
		err = fmt.Errorf("unknown task type: %s", task.Type)
	}

	result.CompletedAt = time.Now()
	result.Output = output

	if err != nil {
		result.Error = err
		result.Status = TaskStatusFailed

		// Check if should retry
		if task.RetryCount < task.MaxRetries {
			result.Status = TaskStatusRetrying
			task.RetryCount++
			// Resubmit task
			go func() {
				time.Sleep(time.Duration(task.RetryCount) * time.Second) // Exponential backoff
				p.Submit(task)
			}()
		}

		atomic.AddInt64(&p.metrics.FailedTasks, 1)
	} else {
		result.Status = TaskStatusCompleted
		atomic.AddInt64(&p.metrics.CompletedTasks, 1)
	}

	result.Logs = append(result.Logs, LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("Task %s completed with status %s", task.ID, result.Status),
		Timestamp: time.Now().UnixMilli(),
		TaskID:    task.ID,
		WorkerID:  worker.ID,
	})

	return result
}

func (p *WorkerPool) executeWorkflowTask(ctx context.Context, task *Task) (map[string]interface{}, error) {
	state, err := p.engine.Execute(ctx, task.Workflow, task.Options)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"executionId": state.ID,
		"status":      state.Status,
		"outputs":     state.NodeOutputs,
	}, nil
}

func (p *WorkerPool) executeNodeTask(ctx context.Context, task *Task) (map[string]interface{}, error) {
	// For individual node execution (used in parallel execution)
	// This would be implemented for parallel node execution
	return nil, fmt.Errorf("node execution not implemented")
}

func (p *WorkerPool) processResults() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case result, ok := <-p.resultQueue:
			if !ok {
				return
			}
			// Process result - could emit events, update database, etc.
			_ = result
		}
	}
}

func (p *WorkerPool) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			// Update metrics
			p.mu.RLock()
			idleCount := int32(0)
			for _, w := range p.workers {
				if w.Status == WorkerStatusIdle {
					idleCount++
				}
			}
			p.mu.RUnlock()
			atomic.StoreInt32(&p.metrics.IdleWorkers, idleCount)
		}
	}
}

// GetMetrics returns current pool metrics
func (p *WorkerPool) GetMetrics() *PoolMetrics {
	return &PoolMetrics{
		TotalTasks:     atomic.LoadInt64(&p.metrics.TotalTasks),
		CompletedTasks: atomic.LoadInt64(&p.metrics.CompletedTasks),
		FailedTasks:    atomic.LoadInt64(&p.metrics.FailedTasks),
		ActiveTasks:    atomic.LoadInt64(&p.metrics.ActiveTasks),
		QueuedTasks:    atomic.LoadInt64(&p.metrics.QueuedTasks),
		TotalWorkers:   atomic.LoadInt32(&p.metrics.TotalWorkers),
		ActiveWorkers:  atomic.LoadInt32(&p.metrics.ActiveWorkers),
		IdleWorkers:    atomic.LoadInt32(&p.metrics.IdleWorkers),
	}
}

// GetWorkers returns list of workers
func (p *WorkerPool) GetWorkers() []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	workers := make([]*Worker, len(p.workers))
	copy(workers, p.workers)
	return workers
}

// ScaleUp adds more workers
func (p *WorkerPool) ScaleUp(count int) {
	p.mu.Lock()
	currentCount := len(p.workers)
	p.mu.Unlock()

	toAdd := count
	if currentCount+toAdd > p.maxWorkers {
		toAdd = p.maxWorkers - currentCount
	}

	for i := 0; i < toAdd; i++ {
		p.addWorker()
	}
}

// ScaleDown removes idle workers
func (p *WorkerPool) ScaleDown(count int) {
	// Workers will naturally stop when context is cancelled
	// For graceful scale down, we mark workers for shutdown
	p.mu.Lock()
	defer p.mu.Unlock()

	removed := 0
	for i := len(p.workers) - 1; i >= 0 && removed < count; i-- {
		if p.workers[i].Status == WorkerStatusIdle {
			p.workers[i].Status = WorkerStatusStopped
			removed++
		}
	}
}
