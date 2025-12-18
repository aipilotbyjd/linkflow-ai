// Package service provides executor business logic
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/executor/domain/model"
)

// WorkerRepository defines worker persistence operations
type WorkerRepository interface {
	Create(ctx context.Context, worker *model.Worker) error
	FindByID(ctx context.Context, id string) (*model.Worker, error)
	Update(ctx context.Context, worker *model.Worker) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, offset, limit int) ([]*model.Worker, int64, error)
	FindAvailable(ctx context.Context) ([]*model.Worker, error)
}

// TaskRepository defines task persistence operations
type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	FindByID(ctx context.Context, id string) (*model.Task, error)
	Update(ctx context.Context, task *model.Task) error
	FindPending(ctx context.Context, limit int) ([]*model.Task, error)
	FindByExecutionID(ctx context.Context, executionID string) ([]*model.Task, error)
}

// ExecutorService manages workflow execution
type ExecutorService struct {
	workerRepo WorkerRepository
	taskRepo   TaskRepository
	workers    map[string]*model.Worker
	taskQueue  chan *model.Task
	mu         sync.RWMutex
	stopCh     chan struct{}
}

// NewExecutorService creates a new executor service
func NewExecutorService(workerRepo WorkerRepository, taskRepo TaskRepository) *ExecutorService {
	return &ExecutorService{
		workerRepo: workerRepo,
		taskRepo:   taskRepo,
		workers:    make(map[string]*model.Worker),
		taskQueue:  make(chan *model.Task, 1000),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the executor service
func (s *ExecutorService) Start(ctx context.Context) error {
	// Load existing workers
	workers, _, err := s.workerRepo.List(ctx, 0, 100)
	if err != nil {
		return fmt.Errorf("failed to load workers: %w", err)
	}

	s.mu.Lock()
	for _, w := range workers {
		s.workers[w.ID] = w
	}
	s.mu.Unlock()

	// Start task dispatcher
	go s.dispatchTasks(ctx)

	// Start worker health checker
	go s.healthCheck(ctx)

	return nil
}

// Stop stops the executor service
func (s *ExecutorService) Stop() {
	close(s.stopCh)
}

// RegisterWorker registers a new worker
func (s *ExecutorService) RegisterWorker(ctx context.Context, input RegisterWorkerInput) (*model.Worker, error) {
	worker := &model.Worker{
		ID:           generateID("worker"),
		Name:         input.Name,
		Host:         input.Host,
		Port:         input.Port,
		Status:       model.WorkerStatusIdle,
		Capacity:     input.Capacity,
		CurrentLoad:  0,
		Tags:         input.Tags,
		LastHeartbeat: time.Now(),
		RegisteredAt:  time.Now(),
	}

	if err := s.workerRepo.Create(ctx, worker); err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	s.mu.Lock()
	s.workers[worker.ID] = worker
	s.mu.Unlock()

	return worker, nil
}

// RegisterWorkerInput represents input for registering a worker
type RegisterWorkerInput struct {
	Name     string
	Host     string
	Port     int
	Capacity int
	Tags     []string
}

// UnregisterWorker removes a worker
func (s *ExecutorService) UnregisterWorker(ctx context.Context, workerID string) error {
	s.mu.Lock()
	delete(s.workers, workerID)
	s.mu.Unlock()

	return s.workerRepo.Delete(ctx, workerID)
}

// Heartbeat updates worker heartbeat
func (s *ExecutorService) Heartbeat(ctx context.Context, workerID string, status model.WorkerStatus, currentLoad int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	worker, exists := s.workers[workerID]
	if !exists {
		return fmt.Errorf("worker not found")
	}

	worker.LastHeartbeat = time.Now()
	worker.Status = status
	worker.CurrentLoad = currentLoad

	return s.workerRepo.Update(ctx, worker)
}

// SubmitTask submits a task for execution
func (s *ExecutorService) SubmitTask(ctx context.Context, input SubmitTaskInput) (*model.Task, error) {
	task := &model.Task{
		ID:          generateID("task"),
		ExecutionID: input.ExecutionID,
		NodeID:      input.NodeID,
		Type:        input.Type,
		Status:      model.TaskStatusPending,
		Priority:    input.Priority,
		Input:       input.Input,
		Retries:     0,
		MaxRetries:  input.MaxRetries,
		CreatedAt:   time.Now(),
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Add to queue
	select {
	case s.taskQueue <- task:
	default:
		return nil, fmt.Errorf("task queue is full")
	}

	return task, nil
}

// SubmitTaskInput represents input for submitting a task
type SubmitTaskInput struct {
	ExecutionID string
	NodeID      string
	Type        string
	Priority    int
	Input       map[string]interface{}
	MaxRetries  int
}

// GetTask retrieves a task by ID
func (s *ExecutorService) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	return s.taskRepo.FindByID(ctx, taskID)
}

// CompleteTask marks a task as completed
func (s *ExecutorService) CompleteTask(ctx context.Context, taskID string, output map[string]interface{}) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}

	now := time.Now()
	task.Status = model.TaskStatusCompleted
	task.Output = output
	task.CompletedAt = &now

	// Release worker
	if task.WorkerID != "" {
		s.releaseWorker(task.WorkerID)
	}

	return s.taskRepo.Update(ctx, task)
}

// FailTask marks a task as failed
func (s *ExecutorService) FailTask(ctx context.Context, taskID string, errMsg string) error {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Release worker
	if task.WorkerID != "" {
		s.releaseWorker(task.WorkerID)
	}

	// Check if we should retry
	if task.Retries < task.MaxRetries {
		task.Retries++
		task.Status = model.TaskStatusPending
		task.WorkerID = ""
		task.Error = errMsg

		if err := s.taskRepo.Update(ctx, task); err != nil {
			return err
		}

		// Re-queue the task
		select {
		case s.taskQueue <- task:
		default:
		}
		return nil
	}

	now := time.Now()
	task.Status = model.TaskStatusFailed
	task.Error = errMsg
	task.CompletedAt = &now

	return s.taskRepo.Update(ctx, task)
}

// GetWorkers returns all registered workers
func (s *ExecutorService) GetWorkers(ctx context.Context) ([]*model.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make([]*model.Worker, 0, len(s.workers))
	for _, w := range s.workers {
		workers = append(workers, w)
	}
	return workers, nil
}

// GetWorker returns a worker by ID
func (s *ExecutorService) GetWorker(ctx context.Context, id string) (*model.Worker, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	worker, exists := s.workers[id]
	if !exists {
		return nil, fmt.Errorf("worker not found")
	}
	return worker, nil
}

func (s *ExecutorService) dispatchTasks(ctx context.Context) {
	for {
		select {
		case <-s.stopCh:
			return
		case task := <-s.taskQueue:
			worker := s.selectWorker(task)
			if worker == nil {
				// No available worker, re-queue
				time.Sleep(100 * time.Millisecond)
				select {
				case s.taskQueue <- task:
				default:
				}
				continue
			}

			// Assign task to worker
			task.WorkerID = worker.ID
			task.Status = model.TaskStatusRunning
			now := time.Now()
			task.StartedAt = &now

			if err := s.taskRepo.Update(ctx, task); err != nil {
				continue
			}

			// Update worker load
			s.mu.Lock()
			worker.CurrentLoad++
			if worker.CurrentLoad >= worker.Capacity {
				worker.Status = model.WorkerStatusBusy
			}
			s.mu.Unlock()
		}
	}
}

func (s *ExecutorService) selectWorker(task *model.Task) *model.Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestWorker *model.Worker
	var lowestLoad int = -1

	for _, worker := range s.workers {
		if worker.Status == model.WorkerStatusOffline {
			continue
		}
		if worker.CurrentLoad >= worker.Capacity {
			continue
		}

		// Check tags if task requires specific capabilities
		if len(task.Tags) > 0 {
			if !hasAllTags(worker.Tags, task.Tags) {
				continue
			}
		}

		// Select worker with lowest load
		if lowestLoad == -1 || worker.CurrentLoad < lowestLoad {
			bestWorker = worker
			lowestLoad = worker.CurrentLoad
		}
	}

	return bestWorker
}

func (s *ExecutorService) releaseWorker(workerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	worker, exists := s.workers[workerID]
	if !exists {
		return
	}

	worker.CurrentLoad--
	if worker.CurrentLoad < worker.Capacity {
		worker.Status = model.WorkerStatusIdle
	}
}

func (s *ExecutorService) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for _, worker := range s.workers {
				if now.Sub(worker.LastHeartbeat) > 60*time.Second {
					worker.Status = model.WorkerStatusOffline
				}
			}
			s.mu.Unlock()
		}
	}
}

func hasAllTags(workerTags, requiredTags []string) bool {
	tagSet := make(map[string]bool)
	for _, t := range workerTags {
		tagSet[t] = true
	}
	for _, t := range requiredTags {
		if !tagSet[t] {
			return false
		}
	}
	return true
}

func generateID(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}
