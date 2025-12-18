// Package nodes provides MySQL node implementation
package nodes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/linkflow-ai/linkflow-ai/internal/node/runtime"
)

func init() {
	runtime.Register(&MySQLNode{})
}

// MySQLNode implements MySQL database operations
type MySQLNode struct{}

func (n *MySQLNode) GetType() string { return "mysql" }
func (n *MySQLNode) Validate(config map[string]interface{}) error { return nil }

func (n *MySQLNode) GetMetadata() runtime.NodeMetadata {
	return runtime.NodeMetadata{
		Type:        "mysql",
		Name:        "MySQL",
		Description: "Execute queries on MySQL database",
		Category:    "integration",
		Version:     "1.0.0",
		Icon:        "mysql",
		Inputs:      []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Outputs:     []runtime.PortDefinition{{Name: "main", Type: "main"}},
		Properties: []runtime.PropertyDefinition{
			{Name: "operation", Type: "select", Required: true, Options: []runtime.PropertyOption{
				{Label: "Query", Value: "query"}, {Label: "Select", Value: "select"},
				{Label: "Insert", Value: "insert"}, {Label: "Update", Value: "update"}, {Label: "Delete", Value: "delete"},
			}},
			{Name: "query", Type: "string"},
			{Name: "table", Type: "string"},
			{Name: "columns", Type: "json"},
			{Name: "values", Type: "json"},
			{Name: "where", Type: "string"},
			{Name: "limit", Type: "number"},
		},
	}
}

func (n *MySQLNode) Execute(ctx context.Context, input *runtime.ExecutionInput) (*runtime.ExecutionOutput, error) {
	host, _ := input.Credentials["host"].(string)
	port, _ := input.Credentials["port"].(string)
	user, _ := input.Credentials["user"].(string)
	password, _ := input.Credentials["password"].(string)
	database, _ := input.Credentials["database"].(string)

	if port == "" {
		port = "3306"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer db.Close()

	operation, _ := input.NodeConfig["operation"].(string)
	var result map[string]interface{}

	switch operation {
	case "query":
		query, _ := input.NodeConfig["query"].(string)
		params, _ := input.NodeConfig["params"].([]interface{})
		result, err = n.executeQuery(ctx, db, query, params)

	case "select":
		result, err = n.executeSelect(ctx, db, input.NodeConfig)

	case "insert":
		result, err = n.executeInsert(ctx, db, input.NodeConfig)

	case "update":
		result, err = n.executeUpdate(ctx, db, input.NodeConfig)

	case "delete":
		result, err = n.executeDelete(ctx, db, input.NodeConfig)

	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}

	if err != nil {
		return &runtime.ExecutionOutput{Error: err}, nil
	}

	return &runtime.ExecutionOutput{Data: result}, nil
}

func (n *MySQLNode) executeQuery(ctx context.Context, db *sql.DB, query string, params []interface{}) (map[string]interface{}, error) {
	// Determine if it's a SELECT or modification query
	queryUpper := strings.ToUpper(strings.TrimSpace(query))
	if strings.HasPrefix(queryUpper, "SELECT") {
		rows, err := db.QueryContext(ctx, query, params...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return n.scanRows(rows)
	}

	result, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()

	return map[string]interface{}{
		"affectedRows": affected,
		"lastInsertId": lastID,
	}, nil
}

func (n *MySQLNode) executeSelect(ctx context.Context, db *sql.DB, config map[string]interface{}) (map[string]interface{}, error) {
	table, _ := config["table"].(string)
	columns, _ := config["columns"].([]interface{})
	where, _ := config["where"].(string)
	orderBy, _ := config["orderBy"].(string)
	limit, _ := config["limit"].(float64)

	// Build column list
	colList := "*"
	if len(columns) > 0 {
		colStrs := make([]string, len(columns))
		for i, c := range columns {
			colStrs[i], _ = c.(string)
		}
		colList = strings.Join(colStrs, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", colList, table)
	if where != "" {
		query += " WHERE " + where
	}
	if orderBy != "" {
		query += " ORDER BY " + orderBy
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", int(limit))
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return n.scanRows(rows)
}

func (n *MySQLNode) executeInsert(ctx context.Context, db *sql.DB, config map[string]interface{}) (map[string]interface{}, error) {
	table, _ := config["table"].(string)
	values, _ := config["values"].(map[string]interface{})

	if len(values) == 0 {
		return nil, fmt.Errorf("values required for insert")
	}

	columns := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	params := make([]interface{}, 0, len(values))

	for col, val := range values {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		params = append(params, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	result, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()
	lastID, _ := result.LastInsertId()

	return map[string]interface{}{
		"affectedRows": affected,
		"lastInsertId": lastID,
	}, nil
}

func (n *MySQLNode) executeUpdate(ctx context.Context, db *sql.DB, config map[string]interface{}) (map[string]interface{}, error) {
	table, _ := config["table"].(string)
	values, _ := config["values"].(map[string]interface{})
	where, _ := config["where"].(string)

	if len(values) == 0 {
		return nil, fmt.Errorf("values required for update")
	}

	setClauses := make([]string, 0, len(values))
	params := make([]interface{}, 0, len(values))

	for col, val := range values {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", col))
		params = append(params, val)
	}

	query := fmt.Sprintf("UPDATE %s SET %s", table, strings.Join(setClauses, ", "))
	if where != "" {
		query += " WHERE " + where
	}

	result, err := db.ExecContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()
	return map[string]interface{}{
		"affectedRows": affected,
	}, nil
}

func (n *MySQLNode) executeDelete(ctx context.Context, db *sql.DB, config map[string]interface{}) (map[string]interface{}, error) {
	table, _ := config["table"].(string)
	where, _ := config["where"].(string)

	query := fmt.Sprintf("DELETE FROM %s", table)
	if where != "" {
		query += " WHERE " + where
	}

	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return nil, err
	}

	affected, _ := result.RowsAffected()
	return map[string]interface{}{
		"affectedRows": affected,
	}, nil
}

func (n *MySQLNode) scanRows(rows *sql.Rows) (map[string]interface{}, error) {
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
			if b, ok := val.([]byte); ok {
				// Try to parse as JSON first
				var jsonVal interface{}
				if err := json.Unmarshal(b, &jsonVal); err == nil {
					row[col] = jsonVal
				} else {
					row[col] = string(b)
				}
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return map[string]interface{}{
		"rows":  results,
		"count": len(results),
	}, nil
}
