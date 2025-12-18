package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/execution/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
)

// ExecutionRepository implements repository.ExecutionRepository using PostgreSQL
type ExecutionRepository struct {
	db *database.DB
}

// NewExecutionRepository creates a new PostgreSQL execution repository
func NewExecutionRepository(db *database.DB) repository.ExecutionRepository {
	return &ExecutionRepository{db: db}
}

// Save saves a new execution
func (r *ExecutionRepository) Save(ctx context.Context, execution *model.Execution) error {
	query := `
		INSERT INTO executions (
			id, workflow_id, workflow_version, user_id, trigger_type, trigger_id,
			status, input_data, output_data, context, node_executions, execution_path,
			error, started_at, completed_at, paused_at, duration_ms, metadata,
			created_at, updated_at, version
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18,
			$19, $20, $21
		)`

	// Serialize JSON fields
	inputData, err := json.Marshal(execution.InputData())
	if err != nil {
		return fmt.Errorf("failed to marshal input data: %w", err)
	}

	outputData, err := json.Marshal(execution.OutputData())
	if err != nil {
		return fmt.Errorf("failed to marshal output data: %w", err)
	}

	contextData, err := json.Marshal(execution.Context())
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	nodeExecutions, err := json.Marshal(execution.NodeExecutions())
	if err != nil {
		return fmt.Errorf("failed to marshal node executions: %w", err)
	}

	executionPath, err := json.Marshal(execution.ExecutionPath())
	if err != nil {
		return fmt.Errorf("failed to marshal execution path: %w", err)
	}

	var errorData []byte
	if execution.Error() != nil {
		errorData, err = json.Marshal(execution.Error())
		if err != nil {
			return fmt.Errorf("failed to marshal error: %w", err)
		}
	}

	metadata, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		execution.ID().String(),
		execution.WorkflowID(),
		execution.WorkflowVersion(),
		execution.UserID(),
		string(execution.TriggerType()),
		"", // trigger_id
		string(execution.Status()),
		inputData,
		outputData,
		contextData,
		nodeExecutions,
		executionPath,
		errorData,
		execution.StartedAt(),
		execution.CompletedAt(),
		nil, // paused_at
		execution.DurationMs(),
		metadata,
		execution.CreatedAt(),
		execution.UpdatedAt(),
		execution.Version(),
	)

	if err != nil {
		return fmt.Errorf("failed to save execution: %w", err)
	}

	return nil
}

// Update updates an existing execution
func (r *ExecutionRepository) Update(ctx context.Context, execution *model.Execution) error {
	query := `
		UPDATE executions SET
			workflow_version = $2,
			trigger_type = $3,
			status = $4,
			input_data = $5,
			output_data = $6,
			context = $7,
			node_executions = $8,
			execution_path = $9,
			error = $10,
			started_at = $11,
			completed_at = $12,
			paused_at = $13,
			duration_ms = $14,
			metadata = $15,
			updated_at = $16,
			version = $17
		WHERE id = $1 AND version = $18`

	// Serialize JSON fields
	inputData, _ := json.Marshal(execution.InputData())
	outputData, _ := json.Marshal(execution.OutputData())
	contextData, _ := json.Marshal(execution.Context())
	nodeExecutions, _ := json.Marshal(execution.NodeExecutions())
	executionPath, _ := json.Marshal(execution.ExecutionPath())
	
	var errorData []byte
	if execution.Error() != nil {
		errorData, _ = json.Marshal(execution.Error())
	}
	
	metadata, _ := json.Marshal(map[string]interface{}{})

	result, err := r.db.ExecContext(ctx, query,
		execution.ID().String(),
		execution.WorkflowVersion(),
		string(execution.TriggerType()),
		string(execution.Status()),
		inputData,
		outputData,
		contextData,
		nodeExecutions,
		executionPath,
		errorData,
		execution.StartedAt(),
		execution.CompletedAt(),
		nil, // paused_at
		execution.DurationMs(),
		metadata,
		execution.UpdatedAt(),
		execution.Version()+1,
		execution.Version(), // for optimistic locking
	)

	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrOptimisticLocking
	}

	return nil
}

// FindByID finds an execution by ID
func (r *ExecutionRepository) FindByID(ctx context.Context, id model.ExecutionID) (*model.Execution, error) {
	query := `
		SELECT
			id, workflow_id, workflow_version, user_id, trigger_type, trigger_id,
			status, input_data, output_data, context, node_executions, execution_path,
			error, started_at, completed_at, paused_at, duration_ms, metadata,
			created_at, updated_at, version
		FROM executions
		WHERE id = $1`

	var execution executionRow
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&execution.ID,
		&execution.WorkflowID,
		&execution.WorkflowVersion,
		&execution.UserID,
		&execution.TriggerType,
		&execution.TriggerID,
		&execution.Status,
		&execution.InputData,
		&execution.OutputData,
		&execution.Context,
		&execution.NodeExecutions,
		&execution.ExecutionPath,
		&execution.Error,
		&execution.StartedAt,
		&execution.CompletedAt,
		&execution.PausedAt,
		&execution.DurationMs,
		&execution.Metadata,
		&execution.CreatedAt,
		&execution.UpdatedAt,
		&execution.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	return execution.toDomain()
}

// FindByWorkflowID finds executions by workflow ID
func (r *ExecutionRepository) FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Execution, error) {
	query := `
		SELECT
			id, workflow_id, workflow_version, user_id, trigger_type, trigger_id,
			status, input_data, output_data, context, node_executions, execution_path,
			error, started_at, completed_at, paused_at, duration_ms, metadata,
			created_at, updated_at, version
		FROM executions
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.Execution
	for rows.Next() {
		var row executionRow
		err := rows.Scan(
			&row.ID,
			&row.WorkflowID,
			&row.WorkflowVersion,
			&row.UserID,
			&row.TriggerType,
			&row.TriggerID,
			&row.Status,
			&row.InputData,
			&row.OutputData,
			&row.Context,
			&row.NodeExecutions,
			&row.ExecutionPath,
			&row.Error,
			&row.StartedAt,
			&row.CompletedAt,
			&row.PausedAt,
			&row.DurationMs,
			&row.Metadata,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// FindByUserID finds executions by user ID
func (r *ExecutionRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Execution, error) {
	query := `
		SELECT
			id, workflow_id, workflow_version, user_id, trigger_type, trigger_id,
			status, input_data, output_data, context, node_executions, execution_path,
			error, started_at, completed_at, paused_at, duration_ms, metadata,
			created_at, updated_at, version
		FROM executions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.Execution
	for rows.Next() {
		var row executionRow
		err := rows.Scan(
			&row.ID,
			&row.WorkflowID,
			&row.WorkflowVersion,
			&row.UserID,
			&row.TriggerType,
			&row.TriggerID,
			&row.Status,
			&row.InputData,
			&row.OutputData,
			&row.Context,
			&row.NodeExecutions,
			&row.ExecutionPath,
			&row.Error,
			&row.StartedAt,
			&row.CompletedAt,
			&row.PausedAt,
			&row.DurationMs,
			&row.Metadata,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// FindByStatus finds executions by status
func (r *ExecutionRepository) FindByStatus(ctx context.Context, status model.ExecutionStatus, offset, limit int) ([]*model.Execution, error) {
	query := `
		SELECT
			id, workflow_id, workflow_version, user_id, trigger_type, trigger_id,
			status, input_data, output_data, context, node_executions, execution_path,
			error, started_at, completed_at, paused_at, duration_ms, metadata,
			created_at, updated_at, version
		FROM executions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, string(status), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find executions: %w", err)
	}
	defer rows.Close()

	var executions []*model.Execution
	for rows.Next() {
		var row executionRow
		err := rows.Scan(
			&row.ID,
			&row.WorkflowID,
			&row.WorkflowVersion,
			&row.UserID,
			&row.TriggerType,
			&row.TriggerID,
			&row.Status,
			&row.InputData,
			&row.OutputData,
			&row.Context,
			&row.NodeExecutions,
			&row.ExecutionPath,
			&row.Error,
			&row.StartedAt,
			&row.CompletedAt,
			&row.PausedAt,
			&row.DurationMs,
			&row.Metadata,
			&row.CreatedAt,
			&row.UpdatedAt,
			&row.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		execution, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// CountByUserID counts executions for a user
func (r *ExecutionRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM executions WHERE user_id = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return count, nil
}

// CountByWorkflowID counts executions for a workflow
func (r *ExecutionRepository) CountByWorkflowID(ctx context.Context, workflowID string) (int64, error) {
	query := `SELECT COUNT(*) FROM executions WHERE workflow_id = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, workflowID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return count, nil
}

// Delete deletes an execution
func (r *ExecutionRepository) Delete(ctx context.Context, id model.ExecutionID) error {
	query := `DELETE FROM executions WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

// executionRow represents a database row for execution
type executionRow struct {
	ID              string
	WorkflowID      string
	WorkflowVersion int
	UserID          string
	TriggerType     string
	TriggerID       sql.NullString
	Status          string
	InputData       []byte
	OutputData      []byte
	Context         []byte
	NodeExecutions  []byte
	ExecutionPath   []byte
	Error           []byte
	StartedAt       sql.NullTime
	CompletedAt     sql.NullTime
	PausedAt        sql.NullTime
	DurationMs      int64
	Metadata        []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Version         int
}

// toDomain converts a database row to domain model
func (r *executionRow) toDomain() (*model.Execution, error) {
	var inputData map[string]interface{}
	if err := json.Unmarshal(r.InputData, &inputData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input data: %w", err)
	}

	var outputData map[string]interface{}
	if err := json.Unmarshal(r.OutputData, &outputData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal output data: %w", err)
	}

	var context model.ExecutionContext
	if err := json.Unmarshal(r.Context, &context); err != nil {
		return nil, fmt.Errorf("failed to unmarshal context: %w", err)
	}

	var nodeExecutions map[string]*model.NodeExecution
	if err := json.Unmarshal(r.NodeExecutions, &nodeExecutions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node executions: %w", err)
	}

	var executionPath []string
	if err := json.Unmarshal(r.ExecutionPath, &executionPath); err != nil {
		return nil, fmt.Errorf("failed to unmarshal execution path: %w", err)
	}

	var executionError *model.ExecutionError
	if len(r.Error) > 0 {
		if err := json.Unmarshal(r.Error, &executionError); err != nil {
			return nil, fmt.Errorf("failed to unmarshal error: %w", err)
		}
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	var startedAt *time.Time
	if r.StartedAt.Valid {
		startedAt = &r.StartedAt.Time
	}

	var completedAt *time.Time
	if r.CompletedAt.Valid {
		completedAt = &r.CompletedAt.Time
	}

	var pausedAt *time.Time
	if r.PausedAt.Valid {
		pausedAt = &r.PausedAt.Time
	}

	triggerID := ""
	if r.TriggerID.Valid {
		triggerID = r.TriggerID.String
	}

	return model.ReconstructExecution(
		model.ExecutionID(r.ID),
		r.WorkflowID,
		r.WorkflowVersion,
		r.UserID,
		model.TriggerType(r.TriggerType),
		triggerID,
		model.ExecutionStatus(r.Status),
		inputData,
		outputData,
		context,
		nodeExecutions,
		executionPath,
		executionError,
		startedAt,
		completedAt,
		pausedAt,
		r.DurationMs,
		metadata,
		r.CreatedAt,
		r.UpdatedAt,
		r.Version,
	), nil
}
