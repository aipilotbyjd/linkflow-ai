// Package postgres provides PostgreSQL implementation of migration repository
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/migration/app/service"
	"github.com/linkflow-ai/linkflow-ai/internal/migration/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
)

// MigrationRepository implements migration persistence using PostgreSQL
type MigrationRepository struct {
	db *database.DB
}

// NewMigrationRepository creates a new PostgreSQL migration repository
func NewMigrationRepository(db *database.DB) service.MigrationRepository {
	return &MigrationRepository{db: db}
}

// EnsureMigrationTable creates the migration history table if it doesn't exist
func (r *MigrationRepository) EnsureMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id VARCHAR(255) PRIMARY KEY,
			version BIGINT NOT NULL UNIQUE,
			name VARCHAR(255) NOT NULL,
			direction VARCHAR(10) NOT NULL,
			status VARCHAR(20) NOT NULL,
			executed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			duration_ms BIGINT NOT NULL DEFAULT 0,
			error TEXT,
			checksum VARCHAR(64),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_version ON schema_migrations(version);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_status ON schema_migrations(status);
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// CreateHistory records a migration execution in the history
func (r *MigrationRepository) CreateHistory(ctx context.Context, history *model.MigrationHistory) error {
	query := `
		INSERT INTO schema_migrations (id, version, name, direction, status, executed_at, duration_ms, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (version) DO UPDATE SET
			status = EXCLUDED.status,
			executed_at = EXCLUDED.executed_at,
			duration_ms = EXCLUDED.duration_ms,
			error = EXCLUDED.error
	`
	
	_, err := r.db.ExecContext(ctx, query,
		history.ID,
		history.Version,
		history.Name,
		string(history.Direction),
		string(history.Status),
		history.ExecutedAt,
		history.DurationMs,
		history.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to create migration history: %w", err)
	}

	return nil
}

// GetCurrentVersion returns the highest applied migration version
func (r *MigrationRepository) GetCurrentVersion(ctx context.Context) (int64, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM schema_migrations
		WHERE status = 'applied' AND direction = 'up'
	`

	var version int64
	err := r.db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get current version: %w", err)
	}

	return version, nil
}

// GetAppliedMigrations returns all applied migrations
func (r *MigrationRepository) GetAppliedMigrations(ctx context.Context) ([]model.MigrationHistory, error) {
	query := `
		SELECT id, version, name, direction, status, executed_at, duration_ms, COALESCE(error, '')
		FROM schema_migrations
		WHERE status = 'applied' AND direction = 'up'
		ORDER BY version ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	var migrations []model.MigrationHistory
	for rows.Next() {
		var h model.MigrationHistory
		var direction, status string

		err := rows.Scan(
			&h.ID,
			&h.Version,
			&h.Name,
			&direction,
			&status,
			&h.ExecutedAt,
			&h.DurationMs,
			&h.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration history: %w", err)
		}

		h.Direction = model.Direction(direction)
		h.Status = model.MigrationStatus(status)
		migrations = append(migrations, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return migrations, nil
}

// Execute runs a SQL statement
func (r *MigrationRepository) Execute(ctx context.Context, sqlStmt string) error {
	_, err := r.db.ExecContext(ctx, sqlStmt)
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

// GetMigrationHistory returns full migration history
func (r *MigrationRepository) GetMigrationHistory(ctx context.Context, limit, offset int) ([]model.MigrationHistory, error) {
	query := `
		SELECT id, version, name, direction, status, executed_at, duration_ms, COALESCE(error, '')
		FROM schema_migrations
		ORDER BY executed_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query migration history: %w", err)
	}
	defer rows.Close()

	var migrations []model.MigrationHistory
	for rows.Next() {
		var h model.MigrationHistory
		var direction, status string

		err := rows.Scan(
			&h.ID,
			&h.Version,
			&h.Name,
			&direction,
			&status,
			&h.ExecutedAt,
			&h.DurationMs,
			&h.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration history: %w", err)
		}

		h.Direction = model.Direction(direction)
		h.Status = model.MigrationStatus(status)
		migrations = append(migrations, h)
	}

	return migrations, nil
}

// MarkMigrationFailed marks a migration as failed
func (r *MigrationRepository) MarkMigrationFailed(ctx context.Context, version int64, errMsg string) error {
	query := `
		UPDATE schema_migrations
		SET status = 'failed', error = $2, executed_at = $3
		WHERE version = $1
	`

	_, err := r.db.ExecContext(ctx, query, version, errMsg, time.Now())
	return err
}

// DeleteMigration removes a migration record (for rollbacks)
func (r *MigrationRepository) DeleteMigration(ctx context.Context, version int64) error {
	query := `DELETE FROM schema_migrations WHERE version = $1`
	_, err := r.db.ExecContext(ctx, query, version)
	return err
}

// IsDirty checks if there are any failed migrations
func (r *MigrationRepository) IsDirty(ctx context.Context) (bool, int64, error) {
	query := `
		SELECT version FROM schema_migrations
		WHERE status = 'failed'
		ORDER BY version DESC
		LIMIT 1
	`

	var version int64
	err := r.db.QueryRowContext(ctx, query).Scan(&version)
	if err == sql.ErrNoRows {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}

	return true, version, nil
}

// ClearDirty clears the dirty state for a version
func (r *MigrationRepository) ClearDirty(ctx context.Context, version int64) error {
	query := `DELETE FROM schema_migrations WHERE version = $1 AND status = 'failed'`
	_, err := r.db.ExecContext(ctx, query, version)
	return err
}
