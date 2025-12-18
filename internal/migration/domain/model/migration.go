// Package model defines migration domain models
package model

import "time"

// Migration represents a database migration
type Migration struct {
	ID        string
	Version   int64
	Name      string
	Content   string
	Direction Direction
	Status    MigrationStatus
	ExecutedAt *time.Time
	DurationMs int64
	Checksum  string
	CreatedAt time.Time
}

// Direction represents migration direction
type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// MigrationStatus represents migration status
type MigrationStatus string

const (
	MigrationStatusPending  MigrationStatus = "pending"
	MigrationStatusRunning  MigrationStatus = "running"
	MigrationStatusApplied  MigrationStatus = "applied"
	MigrationStatusFailed   MigrationStatus = "failed"
	MigrationStatusRolledBack MigrationStatus = "rolled_back"
)

// MigrationHistory represents migration history entry
type MigrationHistory struct {
	ID         string
	Version    int64
	Name       string
	Direction  Direction
	Status     MigrationStatus
	ExecutedAt time.Time
	DurationMs int64
	Error      string
}

// MigrationPlan represents a migration plan
type MigrationPlan struct {
	Migrations []Migration
	Direction  Direction
	DryRun     bool
}

// MigrationResult represents migration execution result
type MigrationResult struct {
	Version    int64
	Name       string
	Direction  Direction
	Status     MigrationStatus
	DurationMs int64
	Error      string
}

// NewMigration creates a new migration
func NewMigration(version int64, name, content string, direction Direction) *Migration {
	return &Migration{
		Version:   version,
		Name:      name,
		Content:   content,
		Direction: direction,
		Status:    MigrationStatusPending,
		CreatedAt: time.Now(),
	}
}

// IsPending returns true if migration is pending
func (m *Migration) IsPending() bool {
	return m.Status == MigrationStatusPending
}

// IsApplied returns true if migration is applied
func (m *Migration) IsApplied() bool {
	return m.Status == MigrationStatusApplied
}

// IsFailed returns true if migration failed
func (m *Migration) IsFailed() bool {
	return m.Status == MigrationStatusFailed
}

// MarkApplied marks the migration as applied
func (m *Migration) MarkApplied(durationMs int64) {
	now := time.Now()
	m.ExecutedAt = &now
	m.DurationMs = durationMs
	m.Status = MigrationStatusApplied
}

// MarkFailed marks the migration as failed
func (m *Migration) MarkFailed() {
	m.Status = MigrationStatusFailed
}

// MarkRolledBack marks the migration as rolled back
func (m *Migration) MarkRolledBack() {
	m.Status = MigrationStatusRolledBack
}

// DataMigration represents a data migration (not schema)
type DataMigration struct {
	ID          string
	Name        string
	Description string
	Query       string
	BatchSize   int
	Status      MigrationStatus
	Progress    float64
	TotalRows   int64
	ProcessedRows int64
	StartedAt   *time.Time
	CompletedAt *time.Time
	Error       string
	CreatedAt   time.Time
}

// SeedData represents seed data for a table
type SeedData struct {
	ID        string
	TableName string
	Data      []map[string]interface{}
	Condition string // Only seed if this condition is met
	Status    MigrationStatus
	CreatedAt time.Time
}

// SchemaSnapshot represents a database schema snapshot
type SchemaSnapshot struct {
	ID        string
	Version   int64
	Tables    []TableSchema
	CreatedAt time.Time
}

// TableSchema represents a table schema
type TableSchema struct {
	Name        string
	Columns     []ColumnSchema
	Indexes     []IndexSchema
	Constraints []ConstraintSchema
}

// ColumnSchema represents a column schema
type ColumnSchema struct {
	Name         string
	Type         string
	Nullable     bool
	Default      string
	IsPrimaryKey bool
}

// IndexSchema represents an index schema
type IndexSchema struct {
	Name     string
	Columns  []string
	IsUnique bool
	Type     string
}

// ConstraintSchema represents a constraint schema
type ConstraintSchema struct {
	Name       string
	Type       string // PRIMARY KEY, FOREIGN KEY, UNIQUE, CHECK
	Columns    []string
	RefTable   string
	RefColumns []string
	OnDelete   string
	OnUpdate   string
}
