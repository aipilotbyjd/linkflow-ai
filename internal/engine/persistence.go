// Package engine provides execution persistence
package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecutionRepository defines execution persistence
type ExecutionRepository interface {
	// Create creates a new execution record
	Create(ctx context.Context, execution *ExecutionRecord) error
	
	// FindByID finds an execution by ID
	FindByID(ctx context.Context, id string) (*ExecutionRecord, error)
	
	// Update updates an execution
	Update(ctx context.Context, execution *ExecutionRecord) error
	
	// Delete deletes an execution
	Delete(ctx context.Context, id string) error
	
	// ListByWorkflow lists executions by workflow ID
	ListByWorkflow(ctx context.Context, workflowID string, limit, offset int) ([]*ExecutionRecord, error)
	
	// ListByStatus lists executions by status
	ListByStatus(ctx context.Context, status ExecutionStatus, limit int) ([]*ExecutionRecord, error)
	
	// ListRecent lists recent executions
	ListRecent(ctx context.Context, limit int) ([]*ExecutionRecord, error)
	
	// CountByWorkflow counts executions for a workflow
	CountByWorkflow(ctx context.Context, workflowID string) (int64, error)
	
	// GetStats returns execution statistics
	GetStats(ctx context.Context, workflowID string, period time.Duration) (*ExecutionStats, error)
}

// ExecutionRecord represents a persisted execution
type ExecutionRecord struct {
	ID            string
	WorkflowID    string
	WorkflowName  string
	Status        ExecutionStatus
	Mode          string
	StartedAt     time.Time
	CompletedAt   *time.Time
	DurationMs    int64
	TriggerData   map[string]interface{}
	NodeOutputs   map[string]interface{}
	Error         string
	RetryCount    int
	ParentID      *string
	UserID        string
	WorkspaceID   string
	Metadata      map[string]interface{}
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ExecutionStats holds execution statistics
type ExecutionStats struct {
	TotalExecutions   int64
	SuccessCount      int64
	FailureCount      int64
	AvgDurationMs     int64
	MinDurationMs     int64
	MaxDurationMs     int64
	SuccessRate       float64
	ExecutionsPerHour float64
}

// InMemoryExecutionRepository implements ExecutionRepository in memory
type InMemoryExecutionRepository struct {
	executions map[string]*ExecutionRecord
	mu         sync.RWMutex
}

// NewInMemoryExecutionRepository creates a new in-memory repository
func NewInMemoryExecutionRepository() *InMemoryExecutionRepository {
	return &InMemoryExecutionRepository{
		executions: make(map[string]*ExecutionRecord),
	}
}

// Create creates a new execution record
func (r *InMemoryExecutionRepository) Create(ctx context.Context, execution *ExecutionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = time.Now()

	r.executions[execution.ID] = execution
	return nil
}

// FindByID finds an execution by ID
func (r *InMemoryExecutionRepository) FindByID(ctx context.Context, id string) (*ExecutionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	execution, ok := r.executions[id]
	if !ok {
		return nil, fmt.Errorf("execution %s not found", id)
	}

	return execution, nil
}

// Update updates an execution
func (r *InMemoryExecutionRepository) Update(ctx context.Context, execution *ExecutionRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	execution.UpdatedAt = time.Now()
	r.executions[execution.ID] = execution
	return nil
}

// Delete deletes an execution
func (r *InMemoryExecutionRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.executions, id)
	return nil
}

// ListByWorkflow lists executions by workflow ID
func (r *InMemoryExecutionRepository) ListByWorkflow(ctx context.Context, workflowID string, limit, offset int) ([]*ExecutionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var executions []*ExecutionRecord
	for _, exec := range r.executions {
		if exec.WorkflowID == workflowID {
			executions = append(executions, exec)
		}
	}

	// Sort by created at desc
	// Simple sort for now
	if offset < len(executions) {
		executions = executions[offset:]
	} else {
		executions = nil
	}

	if limit > 0 && len(executions) > limit {
		executions = executions[:limit]
	}

	return executions, nil
}

// ListByStatus lists executions by status
func (r *InMemoryExecutionRepository) ListByStatus(ctx context.Context, status ExecutionStatus, limit int) ([]*ExecutionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var executions []*ExecutionRecord
	for _, exec := range r.executions {
		if exec.Status == status {
			executions = append(executions, exec)
		}
	}

	if limit > 0 && len(executions) > limit {
		executions = executions[:limit]
	}

	return executions, nil
}

// ListRecent lists recent executions
func (r *InMemoryExecutionRepository) ListRecent(ctx context.Context, limit int) ([]*ExecutionRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var executions []*ExecutionRecord
	for _, exec := range r.executions {
		executions = append(executions, exec)
	}

	if limit > 0 && len(executions) > limit {
		executions = executions[:limit]
	}

	return executions, nil
}

// CountByWorkflow counts executions for a workflow
func (r *InMemoryExecutionRepository) CountByWorkflow(ctx context.Context, workflowID string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var count int64
	for _, exec := range r.executions {
		if exec.WorkflowID == workflowID {
			count++
		}
	}

	return count, nil
}

// GetStats returns execution statistics
func (r *InMemoryExecutionRepository) GetStats(ctx context.Context, workflowID string, period time.Duration) (*ExecutionStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	stats := &ExecutionStats{}
	var totalDuration int64

	for _, exec := range r.executions {
		if workflowID != "" && exec.WorkflowID != workflowID {
			continue
		}
		if exec.StartedAt.Before(cutoff) {
			continue
		}

		stats.TotalExecutions++
		if exec.Status == ExecutionStatusCompleted {
			stats.SuccessCount++
		} else if exec.Status == ExecutionStatusFailed {
			stats.FailureCount++
		}

		totalDuration += exec.DurationMs
		if stats.MinDurationMs == 0 || exec.DurationMs < stats.MinDurationMs {
			stats.MinDurationMs = exec.DurationMs
		}
		if exec.DurationMs > stats.MaxDurationMs {
			stats.MaxDurationMs = exec.DurationMs
		}
	}

	if stats.TotalExecutions > 0 {
		stats.AvgDurationMs = totalDuration / stats.TotalExecutions
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions)
		stats.ExecutionsPerHour = float64(stats.TotalExecutions) / period.Hours()
	}

	return stats, nil
}

// PostgresExecutionRepository implements ExecutionRepository with PostgreSQL
type PostgresExecutionRepository struct {
	db *sql.DB
}

// NewPostgresExecutionRepository creates a new PostgreSQL repository
func NewPostgresExecutionRepository(db *sql.DB) *PostgresExecutionRepository {
	return &PostgresExecutionRepository{db: db}
}

// Create creates a new execution record
func (r *PostgresExecutionRepository) Create(ctx context.Context, execution *ExecutionRecord) error {
	if execution.ID == "" {
		execution.ID = uuid.New().String()
	}
	execution.CreatedAt = time.Now()
	execution.UpdatedAt = time.Now()

	triggerData, _ := json.Marshal(execution.TriggerData)
	nodeOutputs, _ := json.Marshal(execution.NodeOutputs)
	metadata, _ := json.Marshal(execution.Metadata)

	query := `
		INSERT INTO executions (
			id, workflow_id, workflow_name, status, mode,
			started_at, completed_at, duration_ms,
			trigger_data, node_outputs, error,
			retry_count, parent_id, user_id, workspace_id,
			metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.WorkflowID, execution.WorkflowName, execution.Status, execution.Mode,
		execution.StartedAt, execution.CompletedAt, execution.DurationMs,
		triggerData, nodeOutputs, execution.Error,
		execution.RetryCount, execution.ParentID, execution.UserID, execution.WorkspaceID,
		metadata, execution.CreatedAt, execution.UpdatedAt,
	)

	return err
}

// FindByID finds an execution by ID
func (r *PostgresExecutionRepository) FindByID(ctx context.Context, id string) (*ExecutionRecord, error) {
	query := `
		SELECT id, workflow_id, workflow_name, status, mode,
			started_at, completed_at, duration_ms,
			trigger_data, node_outputs, error,
			retry_count, parent_id, user_id, workspace_id,
			metadata, created_at, updated_at
		FROM executions WHERE id = $1`

	var execution ExecutionRecord
	var triggerData, nodeOutputs, metadata []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&execution.ID, &execution.WorkflowID, &execution.WorkflowName, &execution.Status, &execution.Mode,
		&execution.StartedAt, &execution.CompletedAt, &execution.DurationMs,
		&triggerData, &nodeOutputs, &execution.Error,
		&execution.RetryCount, &execution.ParentID, &execution.UserID, &execution.WorkspaceID,
		&metadata, &execution.CreatedAt, &execution.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution %s not found", id)
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(triggerData, &execution.TriggerData)
	json.Unmarshal(nodeOutputs, &execution.NodeOutputs)
	json.Unmarshal(metadata, &execution.Metadata)

	return &execution, nil
}

// Update updates an execution
func (r *PostgresExecutionRepository) Update(ctx context.Context, execution *ExecutionRecord) error {
	execution.UpdatedAt = time.Now()

	triggerData, _ := json.Marshal(execution.TriggerData)
	nodeOutputs, _ := json.Marshal(execution.NodeOutputs)
	metadata, _ := json.Marshal(execution.Metadata)

	query := `
		UPDATE executions SET
			status = $2, completed_at = $3, duration_ms = $4,
			trigger_data = $5, node_outputs = $6, error = $7,
			retry_count = $8, metadata = $9, updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.Status, execution.CompletedAt, execution.DurationMs,
		triggerData, nodeOutputs, execution.Error,
		execution.RetryCount, metadata, execution.UpdatedAt,
	)

	return err
}

// Delete deletes an execution
func (r *PostgresExecutionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM executions WHERE id = $1", id)
	return err
}

// ListByWorkflow lists executions by workflow ID
func (r *PostgresExecutionRepository) ListByWorkflow(ctx context.Context, workflowID string, limit, offset int) ([]*ExecutionRecord, error) {
	query := `
		SELECT id, workflow_id, workflow_name, status, mode,
			started_at, completed_at, duration_ms,
			trigger_data, node_outputs, error,
			retry_count, parent_id, user_id, workspace_id,
			metadata, created_at, updated_at
		FROM executions 
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// ListByStatus lists executions by status
func (r *PostgresExecutionRepository) ListByStatus(ctx context.Context, status ExecutionStatus, limit int) ([]*ExecutionRecord, error) {
	query := `
		SELECT id, workflow_id, workflow_name, status, mode,
			started_at, completed_at, duration_ms,
			trigger_data, node_outputs, error,
			retry_count, parent_id, user_id, workspace_id,
			metadata, created_at, updated_at
		FROM executions 
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// ListRecent lists recent executions
func (r *PostgresExecutionRepository) ListRecent(ctx context.Context, limit int) ([]*ExecutionRecord, error) {
	query := `
		SELECT id, workflow_id, workflow_name, status, mode,
			started_at, completed_at, duration_ms,
			trigger_data, node_outputs, error,
			retry_count, parent_id, user_id, workspace_id,
			metadata, created_at, updated_at
		FROM executions 
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *PostgresExecutionRepository) scanRows(rows *sql.Rows) ([]*ExecutionRecord, error) {
	var executions []*ExecutionRecord

	for rows.Next() {
		var execution ExecutionRecord
		var triggerData, nodeOutputs, metadata []byte

		err := rows.Scan(
			&execution.ID, &execution.WorkflowID, &execution.WorkflowName, &execution.Status, &execution.Mode,
			&execution.StartedAt, &execution.CompletedAt, &execution.DurationMs,
			&triggerData, &nodeOutputs, &execution.Error,
			&execution.RetryCount, &execution.ParentID, &execution.UserID, &execution.WorkspaceID,
			&metadata, &execution.CreatedAt, &execution.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(triggerData, &execution.TriggerData)
		json.Unmarshal(nodeOutputs, &execution.NodeOutputs)
		json.Unmarshal(metadata, &execution.Metadata)

		executions = append(executions, &execution)
	}

	return executions, rows.Err()
}

// CountByWorkflow counts executions for a workflow
func (r *PostgresExecutionRepository) CountByWorkflow(ctx context.Context, workflowID string) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM executions WHERE workflow_id = $1", workflowID).Scan(&count)
	return count, err
}

// GetStats returns execution statistics
func (r *PostgresExecutionRepository) GetStats(ctx context.Context, workflowID string, period time.Duration) (*ExecutionStats, error) {
	cutoff := time.Now().Add(-period)

	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as success,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(AVG(duration_ms), 0) as avg_duration,
			COALESCE(MIN(duration_ms), 0) as min_duration,
			COALESCE(MAX(duration_ms), 0) as max_duration
		FROM executions 
		WHERE ($1 = '' OR workflow_id = $1)
		AND started_at >= $2`

	var stats ExecutionStats
	err := r.db.QueryRowContext(ctx, query, workflowID, cutoff).Scan(
		&stats.TotalExecutions, &stats.SuccessCount, &stats.FailureCount,
		&stats.AvgDurationMs, &stats.MinDurationMs, &stats.MaxDurationMs,
	)
	if err != nil {
		return nil, err
	}

	if stats.TotalExecutions > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalExecutions)
		stats.ExecutionsPerHour = float64(stats.TotalExecutions) / period.Hours()
	}

	return &stats, nil
}
