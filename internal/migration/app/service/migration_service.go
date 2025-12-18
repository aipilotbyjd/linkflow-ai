// Package service provides migration business logic
package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/migration/domain/model"
)

// MigrationRepository defines migration persistence operations
type MigrationRepository interface {
	CreateHistory(ctx context.Context, history *model.MigrationHistory) error
	GetCurrentVersion(ctx context.Context) (int64, error)
	GetAppliedMigrations(ctx context.Context) ([]model.MigrationHistory, error)
	Execute(ctx context.Context, sql string) error
}

// MigrationService handles migration business logic
type MigrationService struct {
	repo           MigrationRepository
	migrationsPath string
}

// NewMigrationService creates a new migration service
func NewMigrationService(repo MigrationRepository, migrationsPath string) *MigrationService {
	return &MigrationService{
		repo:           repo,
		migrationsPath: migrationsPath,
	}
}

// Migration represents a migration for API responses
type Migration struct {
	Version    int64
	Name       string
	Direction  model.Direction
	Status     model.MigrationStatus
	ExecutedAt *time.Time
	DurationMs int64
}

// MigrationStatus represents overall migration status
type MigrationStatus struct {
	CurrentVersion int64
	Pending        int
	Applied        int
	LastApplied    string
	Migrations     []Migration
}

// GetStatus returns the current migration status
func (s *MigrationService) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	currentVersion, err := s.repo.GetCurrentVersion(ctx)
	if err != nil {
		currentVersion = 0
	}

	applied, err := s.repo.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Load all migrations from files
	allMigrations, err := s.loadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	appliedSet := make(map[int64]model.MigrationHistory)
	for _, m := range applied {
		appliedSet[m.Version] = m
	}

	var migrations []Migration
	pending := 0
	lastApplied := ""

	for _, m := range allMigrations {
		status := model.MigrationStatusPending
		var executedAt *time.Time
		var durationMs int64

		if h, ok := appliedSet[m.Version]; ok {
			status = h.Status
			executedAt = &h.ExecutedAt
			durationMs = h.DurationMs
			if h.ExecutedAt.After(time.Time{}) {
				lastApplied = h.ExecutedAt.Format("2006-01-02T15:04:05Z")
			}
		} else {
			pending++
		}

		migrations = append(migrations, Migration{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  m.Direction,
			Status:     status,
			ExecutedAt: executedAt,
			DurationMs: durationMs,
		})
	}

	return &MigrationStatus{
		CurrentVersion: currentVersion,
		Pending:        pending,
		Applied:        len(applied),
		LastApplied:    lastApplied,
		Migrations:     migrations,
	}, nil
}

// ListMigrations lists migrations with optional status filter
func (s *MigrationService) ListMigrations(ctx context.Context, status string) ([]Migration, error) {
	allStatus, err := s.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	if status == "" {
		return allStatus.Migrations, nil
	}

	var filtered []Migration
	for _, m := range allStatus.Migrations {
		if string(m.Status) == status {
			filtered = append(filtered, m)
		}
	}

	return filtered, nil
}

// GetMigration returns a specific migration
func (s *MigrationService) GetMigration(ctx context.Context, version int64) (*Migration, error) {
	migrations, err := s.loadMigrations()
	if err != nil {
		return nil, err
	}

	for _, m := range migrations {
		if m.Version == version {
			return &Migration{
				Version:   m.Version,
				Name:      m.Name,
				Direction: m.Direction,
				Status:    m.Status,
			}, nil
		}
	}

	return nil, fmt.Errorf("migration not found")
}

// MigrateInput represents migration input
type MigrateInput struct {
	Steps  int
	DryRun bool
}

// MigrateResult represents migration result
type MigrateResult struct {
	Applied []model.MigrationResult
	Errors  []string
}

// MigrateUp runs pending up migrations
func (s *MigrationService) MigrateUp(ctx context.Context, input MigrateInput) (*MigrateResult, error) {
	currentVersion, _ := s.repo.GetCurrentVersion(ctx)

	migrations, err := s.loadMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Filter pending migrations
	var pending []*model.Migration
	for _, m := range migrations {
		if m.Version > currentVersion && m.Direction == model.DirectionUp {
			pending = append(pending, m)
		}
	}

	// Sort by version ascending
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	// Limit steps if specified
	if input.Steps > 0 && input.Steps < len(pending) {
		pending = pending[:input.Steps]
	}

	result := &MigrateResult{
		Applied: make([]model.MigrationResult, 0, len(pending)),
	}

	for _, m := range pending {
		start := time.Now()

		if !input.DryRun {
			if err := s.repo.Execute(ctx, m.Content); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("migration %d failed: %v", m.Version, err))
				result.Applied = append(result.Applied, model.MigrationResult{
					Version:   m.Version,
					Name:      m.Name,
					Direction: model.DirectionUp,
					Status:    model.MigrationStatusFailed,
					Error:     err.Error(),
				})
				break
			}

			// Record history
			_ = s.repo.CreateHistory(ctx, &model.MigrationHistory{
				ID:         fmt.Sprintf("mig-%d", time.Now().UnixNano()),
				Version:    m.Version,
				Name:       m.Name,
				Direction:  model.DirectionUp,
				Status:     model.MigrationStatusApplied,
				ExecutedAt: time.Now(),
				DurationMs: time.Since(start).Milliseconds(),
			})
		}

		result.Applied = append(result.Applied, model.MigrationResult{
			Version:    m.Version,
			Name:       m.Name,
			Direction:  model.DirectionUp,
			Status:     model.MigrationStatusApplied,
			DurationMs: time.Since(start).Milliseconds(),
		})
	}

	return result, nil
}

// MigrateDown runs down migrations
func (s *MigrationService) MigrateDown(ctx context.Context, input MigrateInput) (*MigrateResult, error) {
	applied, err := s.repo.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return &MigrateResult{Applied: []model.MigrationResult{}}, nil
	}

	// Sort by version descending
	sort.Slice(applied, func(i, j int) bool {
		return applied[i].Version > applied[j].Version
	})

	// Limit steps
	if input.Steps > 0 && input.Steps < len(applied) {
		applied = applied[:input.Steps]
	}

	result := &MigrateResult{
		Applied: make([]model.MigrationResult, 0, len(applied)),
	}

	for _, h := range applied {
		// Find corresponding down migration
		downMigration, err := s.findDownMigration(h.Version)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("down migration %d not found: %v", h.Version, err))
			continue
		}

		start := time.Now()

		if !input.DryRun {
			if err := s.repo.Execute(ctx, downMigration.Content); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("migration %d rollback failed: %v", h.Version, err))
				result.Applied = append(result.Applied, model.MigrationResult{
					Version:   h.Version,
					Name:      h.Name,
					Direction: model.DirectionDown,
					Status:    model.MigrationStatusFailed,
					Error:     err.Error(),
				})
				break
			}

			// Record history
			_ = s.repo.CreateHistory(ctx, &model.MigrationHistory{
				ID:         fmt.Sprintf("mig-%d", time.Now().UnixNano()),
				Version:    h.Version,
				Name:       h.Name,
				Direction:  model.DirectionDown,
				Status:     model.MigrationStatusRolledBack,
				ExecutedAt: time.Now(),
				DurationMs: time.Since(start).Milliseconds(),
			})
		}

		result.Applied = append(result.Applied, model.MigrationResult{
			Version:    h.Version,
			Name:       h.Name,
			Direction:  model.DirectionDown,
			Status:     model.MigrationStatusRolledBack,
			DurationMs: time.Since(start).Milliseconds(),
		})
	}

	return result, nil
}

// Reset resets the database
func (s *MigrationService) Reset(ctx context.Context, dryRun bool) (*MigrateResult, error) {
	// First migrate down all
	downResult, err := s.MigrateDown(ctx, MigrateInput{Steps: 0, DryRun: dryRun})
	if err != nil {
		return nil, err
	}

	// Then migrate up all
	upResult, err := s.MigrateUp(ctx, MigrateInput{Steps: 0, DryRun: dryRun})
	if err != nil {
		return nil, err
	}

	// Combine results
	result := &MigrateResult{
		Applied: append(downResult.Applied, upResult.Applied...),
		Errors:  append(downResult.Errors, upResult.Errors...),
	}

	return result, nil
}

// SeedInput represents seed input
type SeedInput struct {
	Tables []string
	Force  bool
}

// Seed seeds the database with initial data
func (s *MigrationService) Seed(ctx context.Context, input SeedInput) error {
	seedPath := filepath.Join(s.migrationsPath, "seeds")
	
	files, err := os.ReadDir(seedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No seeds directory
		}
		return fmt.Errorf("failed to read seeds directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		tableName := strings.TrimSuffix(file.Name(), ".sql")

		// Filter by table if specified
		if len(input.Tables) > 0 {
			found := false
			for _, t := range input.Tables {
				if t == tableName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		content, err := os.ReadFile(filepath.Join(seedPath, file.Name()))
		if err != nil {
			return fmt.Errorf("failed to read seed file %s: %w", file.Name(), err)
		}

		if err := s.repo.Execute(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to seed %s: %w", tableName, err)
		}
	}

	return nil
}

func (s *MigrationService) loadMigrations() ([]*model.Migration, error) {
	files, err := os.ReadDir(s.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []*model.Migration

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// Parse filename: 001_create_users.up.sql
		parts := strings.Split(name, "_")
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		direction := model.DirectionUp
		if strings.Contains(name, ".down.") {
			direction = model.DirectionDown
		}

		content, err := os.ReadFile(filepath.Join(s.migrationsPath, name))
		if err != nil {
			continue
		}

		checksum := fmt.Sprintf("%x", md5.Sum(content))

		migrationName := strings.TrimSuffix(strings.TrimPrefix(name, parts[0]+"_"), ".up.sql")
		migrationName = strings.TrimSuffix(migrationName, ".down.sql")

		migrations = append(migrations, &model.Migration{
			ID:        fmt.Sprintf("mig-%d-%s", version, direction),
			Version:   version,
			Name:      migrationName,
			Content:   string(content),
			Direction: direction,
			Status:    model.MigrationStatusPending,
			Checksum:  checksum,
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (s *MigrationService) findDownMigration(version int64) (*model.Migration, error) {
	migrations, err := s.loadMigrations()
	if err != nil {
		return nil, err
	}

	for _, m := range migrations {
		if m.Version == version && m.Direction == model.DirectionDown {
			return m, nil
		}
	}

	return nil, fmt.Errorf("down migration not found")
}
