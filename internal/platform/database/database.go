package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
)

// DB wraps the SQL database connection
type DB struct {
	*sql.DB
	cfg config.DatabaseConfig
}

// New creates a new database connection
func New(cfg config.DatabaseConfig) (*DB, error) {
	dsn := cfg.DSN()
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create schema if specified
	if cfg.Schema != "" {
		if err := createSchema(db, cfg.Schema); err != nil {
			return nil, fmt.Errorf("failed to create schema: %w", err)
		}

		// Set search_path to use the schema
		_, err = db.Exec(fmt.Sprintf("SET search_path TO %s", cfg.Schema))
		if err != nil {
			return nil, fmt.Errorf("failed to set search_path: %w", err)
		}
	}

	return &DB{
		DB:  db,
		cfg: cfg,
	}, nil
}

// createSchema creates the schema if it doesn't exist
func createSchema(db *sql.DB, schema string) error {
	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema)
	_, err := db.Exec(query)
	return err
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check if we can execute a simple query
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query check failed: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// QueryBuilder helps build SQL queries
type QueryBuilder struct {
	query  string
	args   []interface{}
	offset int
	limit  int
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(base string) *QueryBuilder {
	return &QueryBuilder{
		query: base,
		args:  []interface{}{},
	}
}

// Where adds a WHERE clause
func (q *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	if len(q.args) == 0 {
		q.query += " WHERE " + condition
	} else {
		q.query += " AND " + condition
	}
	q.args = append(q.args, args...)
	return q
}

// OrderBy adds an ORDER BY clause
func (q *QueryBuilder) OrderBy(column string, desc bool) *QueryBuilder {
	q.query += fmt.Sprintf(" ORDER BY %s", column)
	if desc {
		q.query += " DESC"
	}
	return q
}

// Limit adds LIMIT and OFFSET
func (q *QueryBuilder) Limit(limit, offset int) *QueryBuilder {
	q.limit = limit
	q.offset = offset
	q.query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
	return q
}

// Build returns the query and arguments
func (q *QueryBuilder) Build() (string, []interface{}) {
	return q.query, q.args
}

// NullString handles nullable strings
func NullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

// NullTime handles nullable time
func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{
		Time:  t,
		Valid: !t.IsZero(),
	}
}

// Scanner interface for custom types
type Scanner interface {
	Scan(src interface{}) error
}

// Valuer interface for custom types
type Valuer interface {
	Value() (driver.Value, error)
}
