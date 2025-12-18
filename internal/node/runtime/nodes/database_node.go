// Package nodes provides built-in node implementations
package nodes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

// PostgresNode implements PostgreSQL database operations
type PostgresNode struct{}

// NewPostgresNode creates a new PostgreSQL node
func NewPostgresNode() *PostgresNode {
	return &PostgresNode{}
}

// GetType returns the node type
func (n *PostgresNode) GetType() string {
	return "postgres"
}

// GetMetadata returns node metadata
func (n *PostgresNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "postgres",
		Name:        "PostgreSQL",
		Description: "Execute queries on PostgreSQL databases",
		Category:    "database",
		Icon:        "database",
		Color:       "#336791",
		Version:     "1.0.0",
		Inputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Input data"},
		},
		Outputs: []runtime.PortDefinition{
			{Name: "main", Type: "any", Description: "Query results"},
		},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Default: "select", Description: "Operation", Options: []runtime.PropertyOption{
				{Label: "Execute Query", Value: "query"},
				{Label: "Select", Value: "select"},
				{Label: "Insert", Value: "insert"},
				{Label: "Update", Value: "update"},
				{Label: "Delete", Value: "delete"},
			}},
			{Name: "table", Type: "string", Description: "Table name (for CRUD operations)"},
			{Name: "query", Type: "code", Description: "SQL query (for query operation)"},
			{Name: "columns", Type: "string", Description: "Columns to select, comma-separated (default: *)"},
			{Name: "values", Type: "json", Description: "Values for insert/update"},
			{Name: "where", Type: "json", Description: "WHERE conditions as key-value pairs"},
			{Name: "orderBy", Type: "string", Description: "ORDER BY clause"},
			{Name: "limit", Type: "number", Description: "LIMIT clause"},
			{Name: "returnData", Type: "boolean", Default: true, Description: "Return inserted/updated data"},
		},
		IsTrigger: false,
	}
}

// Validate validates the node configuration
func (n *PostgresNode) Validate(config map[string]interface{}) error {
	operation := getStringConfig(config, "operation", "select")
	
	switch operation {
	case "query":
		if getStringConfig(config, "query", "") == "" {
			return fmt.Errorf("query is required for query operation")
		}
	case "select", "insert", "update", "delete":
		if getStringConfig(config, "table", "") == "" {
			return fmt.Errorf("table is required for %s operation", operation)
		}
	}
	
	return nil
}

// Execute executes the PostgreSQL node
func (n *PostgresNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	startTime := time.Now()
	output := &runtime.ExecutionOutput{
		Data: make(map[string]interface{}),
		Logs: []runtime.LogEntry{},
	}
	
	// Get connection string from credentials
	connStr := ""
	if input.Credentials != nil {
		if cs, ok := input.Credentials["connectionString"].(string); ok {
			connStr = cs
		} else {
			// Build from parts
			host := getCredString(input.Credentials, "host", "localhost")
			port := getCredString(input.Credentials, "port", "5432")
			database := getCredString(input.Credentials, "database", "")
			username := getCredString(input.Credentials, "username", "")
			password := getCredString(input.Credentials, "password", "")
			sslmode := getCredString(input.Credentials, "ssl", "disable")
			
			connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
				host, port, username, password, database, sslmode)
		}
	}
	
	if connStr == "" {
		output.Error = fmt.Errorf("database credentials required")
		return output, nil
	}
	
	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		output.Error = fmt.Errorf("failed to connect: %w", err)
		return output, nil
	}
	defer db.Close()
	
	// Execute operation
	operation := getStringConfig(input.NodeConfig, "operation", "select")
	
	var result interface{}
	switch operation {
	case "query":
		result, err = n.executeQuery(ctx, db, input.NodeConfig)
	case "select":
		result, err = n.executeSelect(ctx, db, input.NodeConfig)
	case "insert":
		result, err = n.executeInsert(ctx, db, input.NodeConfig)
	case "update":
		result, err = n.executeUpdate(ctx, db, input.NodeConfig)
	case "delete":
		result, err = n.executeDelete(ctx, db, input.NodeConfig)
	default:
		err = fmt.Errorf("unknown operation: %s", operation)
	}
	
	if err != nil {
		output.Error = err
		output.Logs = append(output.Logs, runtime.LogEntry{
			Level:     "error",
			Message:   fmt.Sprintf("Database operation failed: %v", err),
			Timestamp: time.Now().UnixMilli(),
			NodeID:    input.NodeID,
		})
		return output, nil
	}
	
	// Set output
	if rows, ok := result.([]map[string]interface{}); ok {
		output.Data["rows"] = rows
		output.Data["rowCount"] = len(rows)
	} else if m, ok := result.(map[string]interface{}); ok {
		output.Data = m
	}
	
	output.Logs = append(output.Logs, runtime.LogEntry{
		Level:     "info",
		Message:   fmt.Sprintf("PostgreSQL %s completed", operation),
		Timestamp: time.Now().UnixMilli(),
		NodeID:    input.NodeID,
	})
	
	output.Metrics = runtime.ExecutionMetrics{
		StartTime:  startTime.UnixMilli(),
		EndTime:    time.Now().UnixMilli(),
		DurationMs: time.Since(startTime).Milliseconds(),
	}
	
	return output, nil
}

func (n *PostgresNode) executeQuery(ctx context.Context, db *sql.DB, config map[string]interface{}) (interface{}, error) {
	query := getStringConfig(config, "query", "")
	
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return scanRows(rows)
}

func (n *PostgresNode) executeSelect(ctx context.Context, db *sql.DB, config map[string]interface{}) (interface{}, error) {
	table := getStringConfig(config, "table", "")
	columns := getStringConfig(config, "columns", "*")
	orderBy := getStringConfig(config, "orderBy", "")
	limit := getIntConfig(config, "limit", 0)
	where := getMapConfig(config, "where")
	
	// Build query
	query := fmt.Sprintf("SELECT %s FROM %s", columns, table)
	
	var args []interface{}
	if len(where) > 0 {
		conditions, whereArgs := buildWhereClause(where)
		query += " WHERE " + conditions
		args = whereArgs
	}
	
	if orderBy != "" {
		query += " ORDER BY " + orderBy
	}
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	return scanRows(rows)
}

func (n *PostgresNode) executeInsert(ctx context.Context, db *sql.DB, config map[string]interface{}) (interface{}, error) {
	table := getStringConfig(config, "table", "")
	values := getMapConfig(config, "values")
	returnData := getBoolConfig(config, "returnData", true)
	
	if len(values) == 0 {
		return nil, fmt.Errorf("values required for insert")
	}
	
	// Build query
	var columns []string
	var placeholders []string
	var args []interface{}
	i := 1
	
	for col, val := range values {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}
	
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	
	if returnData {
		query += " RETURNING *"
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanRows(rows)
	}
	
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	
	affected, _ := result.RowsAffected()
	return map[string]interface{}{
		"success":      true,
		"rowsAffected": affected,
	}, nil
}

func (n *PostgresNode) executeUpdate(ctx context.Context, db *sql.DB, config map[string]interface{}) (interface{}, error) {
	table := getStringConfig(config, "table", "")
	values := getMapConfig(config, "values")
	where := getMapConfig(config, "where")
	returnData := getBoolConfig(config, "returnData", true)
	
	if len(values) == 0 {
		return nil, fmt.Errorf("values required for update")
	}
	
	// Build SET clause
	var setClauses []string
	var args []interface{}
	i := 1
	
	for col, val := range values {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	
	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ", "))
	
	// Add WHERE clause
	if len(where) > 0 {
		var whereClauses []string
		for col, val := range where {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", col, i))
			args = append(args, val)
			i++
		}
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	
	if returnData {
		query += " RETURNING *"
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanRows(rows)
	}
	
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	
	affected, _ := result.RowsAffected()
	return map[string]interface{}{
		"success":      true,
		"rowsAffected": affected,
	}, nil
}

func (n *PostgresNode) executeDelete(ctx context.Context, db *sql.DB, config map[string]interface{}) (interface{}, error) {
	table := getStringConfig(config, "table", "")
	where := getMapConfig(config, "where")
	returnData := getBoolConfig(config, "returnData", true)
	
	query := fmt.Sprintf("DELETE FROM %s", table)
	
	var args []interface{}
	if len(where) > 0 {
		conditions, whereArgs := buildWhereClause(where)
		query += " WHERE " + conditions
		args = whereArgs
	}
	
	if returnData {
		query += " RETURNING *"
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanRows(rows)
	}
	
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	
	affected, _ := result.RowsAffected()
	return map[string]interface{}{
		"success":      true,
		"rowsAffected": affected,
	}, nil
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	
	var results []map[string]interface{}
	
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			
			// Convert byte arrays to strings
			if b, ok := val.([]byte); ok {
				// Try to parse as JSON
				var jsonVal interface{}
				if err := json.Unmarshal(b, &jsonVal); err == nil {
					val = jsonVal
				} else {
					val = string(b)
				}
			}
			
			row[col] = val
		}
		
		results = append(results, row)
	}
	
	return results, rows.Err()
}

func buildWhereClause(where map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	i := 1
	
	for col, val := range where {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}
	
	return strings.Join(conditions, " AND "), args
}

func getCredString(creds map[string]interface{}, key, defaultVal string) string {
	if v, ok := creds[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func init() {
	runtime.Register(NewPostgresNode())
}
