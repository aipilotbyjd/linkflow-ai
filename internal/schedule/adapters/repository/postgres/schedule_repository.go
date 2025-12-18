package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/schedule/domain/repository"
)

// ScheduleRepository implements repository.ScheduleRepository using PostgreSQL
type ScheduleRepository struct {
	db *database.DB
}

// NewScheduleRepository creates a new PostgreSQL schedule repository
func NewScheduleRepository(db *database.DB) repository.ScheduleRepository {
	return &ScheduleRepository{db: db}
}

// Save saves a new schedule
func (r *ScheduleRepository) Save(ctx context.Context, schedule *model.Schedule) error {
	query := `
		INSERT INTO schedules (
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19
		)`

	// Serialize metadata
	metadata, _ := json.Marshal(schedule.Metadata())

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID().String(),
		schedule.UserID(),
		schedule.OrganizationID(),
		schedule.WorkflowID(),
		schedule.Name(),
		schedule.Description(),
		schedule.CronExpression(),
		schedule.Timezone(),
		schedule.StartDate(),
		schedule.EndDate(),
		string(schedule.Status()),
		schedule.LastRunAt(),
		schedule.NextRunAt(),
		schedule.RunCount(),
		schedule.SuccessCount(),
		schedule.FailureCount(),
		metadata,
		schedule.CreatedAt(),
		schedule.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to save schedule: %w", err)
	}

	return nil
}

// Update updates an existing schedule
func (r *ScheduleRepository) Update(ctx context.Context, schedule *model.Schedule) error {
	query := `
		UPDATE schedules SET
			name = $2,
			description = $3,
			cron_expression = $4,
			timezone = $5,
			start_date = $6,
			end_date = $7,
			status = $8,
			last_run_at = $9,
			next_run_at = $10,
			run_count = $11,
			success_count = $12,
			failure_count = $13,
			metadata = $14,
			updated_at = $15
		WHERE id = $1 AND updated_at = $16`

	// Serialize metadata
	metadata, _ := json.Marshal(schedule.Metadata())

	result, err := r.db.ExecContext(ctx, query,
		schedule.ID().String(),
		schedule.Name(),
		schedule.Description(),
		schedule.CronExpression(),
		schedule.Timezone(),
		schedule.StartDate(),
		schedule.EndDate(),
		string(schedule.Status()),
		schedule.LastRunAt(),
		schedule.NextRunAt(),
		schedule.RunCount(),
		schedule.SuccessCount(),
		schedule.FailureCount(),
		metadata,
		time.Now(),
		schedule.UpdatedAt(), // for optimistic locking
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
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

// FindByID finds a schedule by ID
func (r *ScheduleRepository) FindByID(ctx context.Context, id model.ScheduleID) (*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE id = $1`

	var row scheduleRow
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&row.ID,
		&row.UserID,
		&row.OrganizationID,
		&row.WorkflowID,
		&row.Name,
		&row.Description,
		&row.CronExpression,
		&row.Timezone,
		&row.StartDate,
		&row.EndDate,
		&row.Status,
		&row.LastRunAt,
		&row.NextRunAt,
		&row.RunCount,
		&row.SuccessCount,
		&row.FailureCount,
		&row.Metadata,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find schedule: %w", err)
	}

	return row.toDomain()
}

// FindByUserID finds schedules by user ID
func (r *ScheduleRepository) FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find schedules: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// FindByWorkflowID finds schedules by workflow ID
func (r *ScheduleRepository) FindByWorkflowID(ctx context.Context, workflowID string, offset, limit int) ([]*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find schedules: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// FindByOrganizationID finds schedules by organization ID
func (r *ScheduleRepository) FindByOrganizationID(ctx context.Context, orgID string, offset, limit int) ([]*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find schedules: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// FindByStatus finds schedules by status
func (r *ScheduleRepository) FindByStatus(ctx context.Context, status model.ScheduleStatus, offset, limit int) ([]*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, string(status), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find schedules: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// FindDueSchedules finds schedules that are due to run
func (r *ScheduleRepository) FindDueSchedules(ctx context.Context, before time.Time, limit int) ([]*model.Schedule, error) {
	query := `
		SELECT
			id, user_id, organization_id, workflow_id, name, description,
			cron_expression, timezone, start_date, end_date, status,
			last_run_at, next_run_at, run_count, success_count, failure_count,
			metadata, created_at, updated_at
		FROM schedules
		WHERE status = 'active' 
		AND next_run_at <= $1
		ORDER BY next_run_at ASC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, before, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to find due schedules: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// CountByUserID counts schedules for a user
func (r *ScheduleRepository) CountByUserID(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COUNT(*) FROM schedules WHERE user_id = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	return count, nil
}

// CountByWorkflowID counts schedules for a workflow
func (r *ScheduleRepository) CountByWorkflowID(ctx context.Context, workflowID string) (int64, error) {
	query := `SELECT COUNT(*) FROM schedules WHERE workflow_id = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, workflowID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	return count, nil
}

// Delete deletes a schedule
func (r *ScheduleRepository) Delete(ctx context.Context, id model.ScheduleID) error {
	query := `DELETE FROM schedules WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
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

// Helper methods

func (r *ScheduleRepository) scanRows(rows *sql.Rows) ([]*model.Schedule, error) {
	var schedules []*model.Schedule
	
	for rows.Next() {
		var row scheduleRow
		err := rows.Scan(
			&row.ID,
			&row.UserID,
			&row.OrganizationID,
			&row.WorkflowID,
			&row.Name,
			&row.Description,
			&row.CronExpression,
			&row.Timezone,
			&row.StartDate,
			&row.EndDate,
			&row.Status,
			&row.LastRunAt,
			&row.NextRunAt,
			&row.RunCount,
			&row.SuccessCount,
			&row.FailureCount,
			&row.Metadata,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		schedule, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// scheduleRow represents a database row for schedule
type scheduleRow struct {
	ID             string
	UserID         string
	OrganizationID sql.NullString
	WorkflowID     string
	Name           string
	Description    sql.NullString
	CronExpression string
	Timezone       string
	StartDate      sql.NullTime
	EndDate        sql.NullTime
	Status         string
	LastRunAt      sql.NullTime
	NextRunAt      sql.NullTime
	RunCount       int64
	SuccessCount   int64
	FailureCount   int64
	Metadata       []byte
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// toDomain converts a database row to domain model
func (r *scheduleRow) toDomain() (*model.Schedule, error) {
	// Create base schedule
	schedule, err := model.NewSchedule(
		r.UserID,
		r.WorkflowID,
		r.Name,
		r.CronExpression,
		r.Timezone,
	)
	if err != nil {
		// If we can't create a new schedule, try to reconstruct from stored data
		// This might happen if cron expression has changed format
		// For now, return a basic schedule with error
		return nil, fmt.Errorf("failed to reconstruct schedule: %w", err)
	}

	// Set additional fields
	if r.OrganizationID.Valid {
		schedule.SetOrganizationID(r.OrganizationID.String)
	}
	if r.Description.Valid {
		schedule.SetDescription(r.Description.String)
	}
	
	// Set date range if present
	var startDate, endDate *time.Time
	if r.StartDate.Valid {
		startDate = &r.StartDate.Time
	}
	if r.EndDate.Valid {
		endDate = &r.EndDate.Time
	}
	if startDate != nil || endDate != nil {
		_ = schedule.SetDateRange(startDate, endDate)
	}

	return schedule, nil
}
