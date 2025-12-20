package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// GormDB wraps the GORM database connection
type GormDB struct {
	*gorm.DB
	cfg config.DatabaseConfig
}

// NewGorm creates a new GORM database connection
func NewGorm(cfg config.DatabaseConfig) (*GormDB, error) {
	dsn := cfg.DSN()

	// Configure GORM logger based on environment
	var gormLogger logger.Interface
	if cfg.Host == "localhost" || os.Getenv("ENVIRONMENT") == "development" {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Info,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)
	} else {
		gormLogger = logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             500 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true, // Better performance
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set search_path if schema is specified
	if cfg.Schema != "" {
		if err := db.Exec(fmt.Sprintf("SET search_path TO %s", cfg.Schema)).Error; err != nil {
			return nil, fmt.Errorf("failed to set search_path: %w", err)
		}
	}

	return &GormDB{
		DB:  db,
		cfg: cfg,
	}, nil
}

// NewGormFromExisting creates a GORM instance from existing *sql.DB
func NewGormFromExisting(sqlDB *DB) (*GormDB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB.DB,
	}), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GORM from existing connection: %w", err)
	}

	return &GormDB{
		DB:  db,
		cfg: sqlDB.cfg,
	}, nil
}

// HealthCheck performs a health check on the database
func (db *GormDB) HealthCheck(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	var result int
	if err := db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error; err != nil {
		return fmt.Errorf("database query check failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *GormDB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction executes a function within a database transaction
func (db *GormDB) Transaction(fn func(*gorm.DB) error) error {
	return db.DB.Transaction(fn)
}
