// Package postgres provides PostgreSQL repository implementations
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/linkflow-ai/linkflow-ai/internal/executor/domain/model"
)

// WorkerRepository implements worker persistence with PostgreSQL
type WorkerRepository struct {
	db *sql.DB
}

// NewWorkerRepository creates a new worker repository
func NewWorkerRepository(db *sql.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

// Create creates a new worker
func (r *WorkerRepository) Create(ctx context.Context, worker *model.Worker) error {
	tagsJSON, err := json.Marshal(worker.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO executor_workers (id, name, host, port, status, capacity, current_load, tags, last_heartbeat, registered_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.ExecContext(ctx, query,
		worker.ID,
		worker.Name,
		worker.Host,
		worker.Port,
		worker.Status,
		worker.Capacity,
		worker.CurrentLoad,
		tagsJSON,
		worker.LastHeartbeat,
		worker.RegisteredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create worker: %w", err)
	}

	return nil
}

// FindByID finds a worker by ID
func (r *WorkerRepository) FindByID(ctx context.Context, id string) (*model.Worker, error) {
	query := `
		SELECT id, name, host, port, status, capacity, current_load, tags, last_heartbeat, registered_at
		FROM executor_workers
		WHERE id = $1
	`

	var worker model.Worker
	var tagsJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&worker.ID,
		&worker.Name,
		&worker.Host,
		&worker.Port,
		&worker.Status,
		&worker.Capacity,
		&worker.CurrentLoad,
		&tagsJSON,
		&worker.LastHeartbeat,
		&worker.RegisteredAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker not found")
		}
		return nil, fmt.Errorf("failed to find worker: %w", err)
	}

	if err := json.Unmarshal(tagsJSON, &worker.Tags); err != nil {
		worker.Tags = []string{}
	}

	return &worker, nil
}

// Update updates a worker
func (r *WorkerRepository) Update(ctx context.Context, worker *model.Worker) error {
	query := `
		UPDATE executor_workers 
		SET status = $2, capacity = $3, current_load = $4, last_heartbeat = $5
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		worker.ID,
		worker.Status,
		worker.Capacity,
		worker.CurrentLoad,
		worker.LastHeartbeat,
	)
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}

	return nil
}

// Delete deletes a worker
func (r *WorkerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM executor_workers WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}
	return nil
}

// List lists workers with pagination
func (r *WorkerRepository) List(ctx context.Context, offset, limit int) ([]*model.Worker, int64, error) {
	countQuery := `SELECT COUNT(*) FROM executor_workers`
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count workers: %w", err)
	}

	query := `
		SELECT id, name, host, port, status, capacity, current_load, tags, last_heartbeat, registered_at
		FROM executor_workers
		ORDER BY registered_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list workers: %w", err)
	}
	defer rows.Close()

	var workers []*model.Worker
	for rows.Next() {
		var worker model.Worker
		var tagsJSON []byte

		err := rows.Scan(
			&worker.ID,
			&worker.Name,
			&worker.Host,
			&worker.Port,
			&worker.Status,
			&worker.Capacity,
			&worker.CurrentLoad,
			&tagsJSON,
			&worker.LastHeartbeat,
			&worker.RegisteredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan worker: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &worker.Tags); err != nil {
			worker.Tags = []string{}
		}

		workers = append(workers, &worker)
	}

	return workers, total, nil
}

// FindAvailable finds available workers
func (r *WorkerRepository) FindAvailable(ctx context.Context) ([]*model.Worker, error) {
	query := `
		SELECT id, name, host, port, status, capacity, current_load, tags, last_heartbeat, registered_at
		FROM executor_workers
		WHERE status != 'offline' AND current_load < capacity
		ORDER BY current_load ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find available workers: %w", err)
	}
	defer rows.Close()

	var workers []*model.Worker
	for rows.Next() {
		var worker model.Worker
		var tagsJSON []byte

		err := rows.Scan(
			&worker.ID,
			&worker.Name,
			&worker.Host,
			&worker.Port,
			&worker.Status,
			&worker.Capacity,
			&worker.CurrentLoad,
			&tagsJSON,
			&worker.LastHeartbeat,
			&worker.RegisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan worker: %w", err)
		}

		if err := json.Unmarshal(tagsJSON, &worker.Tags); err != nil {
			worker.Tags = []string{}
		}

		workers = append(workers, &worker)
	}

	return workers, nil
}

// TaskRepository implements task persistence with PostgreSQL
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	inputJSON, err := json.Marshal(task.Input)
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	tagsJSON, err := json.Marshal(task.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO execution_tasks (id, execution_id, node_id, worker_id, type, status, priority, input, tags, retries, max_retries, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = r.db.ExecContext(ctx, query,
		task.ID,
		task.ExecutionID,
		task.NodeID,
		task.WorkerID,
		task.Type,
		task.Status,
		task.Priority,
		inputJSON,
		tagsJSON,
		task.Retries,
		task.MaxRetries,
		task.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// FindByID finds a task by ID
func (r *TaskRepository) FindByID(ctx context.Context, id string) (*model.Task, error) {
	query := `
		SELECT id, execution_id, node_id, worker_id, type, status, priority, input, output, error, tags, retries, max_retries, created_at, started_at, completed_at
		FROM execution_tasks
		WHERE id = $1
	`

	var task model.Task
	var inputJSON, outputJSON, tagsJSON []byte
	var workerID sql.NullString
	var errorMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.ExecutionID,
		&task.NodeID,
		&workerID,
		&task.Type,
		&task.Status,
		&task.Priority,
		&inputJSON,
		&outputJSON,
		&errorMsg,
		&tagsJSON,
		&task.Retries,
		&task.MaxRetries,
		&task.CreatedAt,
		&task.StartedAt,
		&task.CompletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	if workerID.Valid {
		task.WorkerID = workerID.String
	}
	if errorMsg.Valid {
		task.Error = errorMsg.String
	}

	if err := json.Unmarshal(inputJSON, &task.Input); err != nil {
		task.Input = make(map[string]interface{})
	}
	if outputJSON != nil {
		if err := json.Unmarshal(outputJSON, &task.Output); err != nil {
			task.Output = make(map[string]interface{})
		}
	}
	if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
		task.Tags = []string{}
	}

	return &task, nil
}

// Update updates a task
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	outputJSON, err := json.Marshal(task.Output)
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	query := `
		UPDATE execution_tasks 
		SET worker_id = $2, status = $3, output = $4, error = $5, retries = $6, started_at = $7, completed_at = $8
		WHERE id = $1
	`

	var workerID interface{}
	if task.WorkerID != "" {
		workerID = task.WorkerID
	}

	_, err = r.db.ExecContext(ctx, query,
		task.ID,
		workerID,
		task.Status,
		outputJSON,
		task.Error,
		task.Retries,
		task.StartedAt,
		task.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// FindPending finds pending tasks
func (r *TaskRepository) FindPending(ctx context.Context, limit int) ([]*model.Task, error) {
	query := `
		SELECT id, execution_id, node_id, worker_id, type, status, priority, input, tags, retries, max_retries, created_at
		FROM execution_tasks
		WHERE status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find pending tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		var task model.Task
		var inputJSON, tagsJSON []byte
		var workerID sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ExecutionID,
			&task.NodeID,
			&workerID,
			&task.Type,
			&task.Status,
			&task.Priority,
			&inputJSON,
			&tagsJSON,
			&task.Retries,
			&task.MaxRetries,
			&task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if workerID.Valid {
			task.WorkerID = workerID.String
		}
		if err := json.Unmarshal(inputJSON, &task.Input); err != nil {
			task.Input = make(map[string]interface{})
		}
		if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
			task.Tags = []string{}
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// FindByExecutionID finds tasks by execution ID
func (r *TaskRepository) FindByExecutionID(ctx context.Context, executionID string) ([]*model.Task, error) {
	query := `
		SELECT id, execution_id, node_id, worker_id, type, status, priority, input, output, error, tags, retries, max_retries, created_at, started_at, completed_at
		FROM execution_tasks
		WHERE execution_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		var task model.Task
		var inputJSON, outputJSON, tagsJSON []byte
		var workerID, errorMsg sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ExecutionID,
			&task.NodeID,
			&workerID,
			&task.Type,
			&task.Status,
			&task.Priority,
			&inputJSON,
			&outputJSON,
			&errorMsg,
			&tagsJSON,
			&task.Retries,
			&task.MaxRetries,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if workerID.Valid {
			task.WorkerID = workerID.String
		}
		if errorMsg.Valid {
			task.Error = errorMsg.String
		}
		if err := json.Unmarshal(inputJSON, &task.Input); err != nil {
			task.Input = make(map[string]interface{})
		}
		if outputJSON != nil {
			if err := json.Unmarshal(outputJSON, &task.Output); err != nil {
				task.Output = make(map[string]interface{})
			}
		}
		if err := json.Unmarshal(tagsJSON, &task.Tags); err != nil {
			task.Tags = []string{}
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}
