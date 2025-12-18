package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/model"
	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/repository"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
)

// NodeDefinitionRepository implements repository.NodeDefinitionRepository using PostgreSQL
type NodeDefinitionRepository struct {
	db *database.DB
}

// NewNodeDefinitionRepository creates a new PostgreSQL node definition repository
func NewNodeDefinitionRepository(db *database.DB) repository.NodeDefinitionRepository {
	return &NodeDefinitionRepository{db: db}
}

// Save saves a new node definition
func (r *NodeDefinitionRepository) Save(ctx context.Context, node *model.NodeDefinition) error {
	query := `
		INSERT INTO node_definitions (
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20,
			$21, $22, $23, $24
		)`

	// Serialize JSON fields
	inputs, _ := json.Marshal(node.Inputs())
	outputs, _ := json.Marshal(node.Outputs())
	properties, _ := json.Marshal(node.Properties())
	configuration, _ := json.Marshal(node.Configuration())
	validationRules, _ := json.Marshal(map[string]interface{}{})
	examples, _ := json.Marshal([]map[string]interface{}{})
	metadata, _ := json.Marshal(map[string]interface{}{})

	_, err := r.db.ExecContext(ctx, query,
		node.ID().String(),
		node.Name(),
		string(node.Type()),
		string(node.Category()),
		node.Description(),
		node.Icon(),
		node.Color(),
		node.Version(),
		string(node.Status()),
		inputs,
		outputs,
		properties,
		configuration,
		validationRules,
		"", // documentation
		examples,
		pq.Array(node.Tags()),
		"", // author
		node.IsSystem(),
		node.IsPremium(),
		node.ExecutionHandler(),
		metadata,
		node.CreatedAt(),
		node.UpdatedAt(),
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return repository.ErrDuplicateName
		}
		return fmt.Errorf("failed to save node definition: %w", err)
	}

	return nil
}

// Update updates an existing node definition
func (r *NodeDefinitionRepository) Update(ctx context.Context, node *model.NodeDefinition) error {
	query := `
		UPDATE node_definitions SET
			name = $2,
			description = $3,
			icon = $4,
			color = $5,
			status = $6,
			inputs = $7,
			outputs = $8,
			properties = $9,
			configuration = $10,
			documentation = $11,
			tags = $12,
			is_system = $13,
			is_premium = $14,
			execution_handler = $15,
			updated_at = $16
		WHERE id = $1`

	// Serialize JSON fields
	inputs, _ := json.Marshal(node.Inputs())
	outputs, _ := json.Marshal(node.Outputs())
	properties, _ := json.Marshal(node.Properties())
	configuration, _ := json.Marshal(node.Configuration())

	result, err := r.db.ExecContext(ctx, query,
		node.ID().String(),
		node.Name(),
		node.Description(),
		node.Icon(),
		node.Color(),
		string(node.Status()),
		inputs,
		outputs,
		properties,
		configuration,
		"", // documentation
		pq.Array(node.Tags()),
		node.IsSystem(),
		node.IsPremium(),
		node.ExecutionHandler(),
		node.UpdatedAt(),
	)

	if err != nil {
		return fmt.Errorf("failed to update node definition: %w", err)
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

// FindByID finds a node definition by ID
func (r *NodeDefinitionRepository) FindByID(ctx context.Context, id model.NodeID) (*model.NodeDefinition, error) {
	query := `
		SELECT
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		FROM node_definitions
		WHERE id = $1`

	var row nodeRow
	err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Category,
		&row.Description,
		&row.Icon,
		&row.Color,
		&row.Version,
		&row.Status,
		&row.Inputs,
		&row.Outputs,
		&row.Properties,
		&row.Configuration,
		&row.ValidationRules,
		&row.Documentation,
		&row.Examples,
		pq.Array(&row.Tags),
		&row.Author,
		&row.IsSystem,
		&row.IsPremium,
		&row.ExecutionHandler,
		&row.Metadata,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find node definition: %w", err)
	}

	return row.toDomain()
}

// FindByName finds a node definition by name
func (r *NodeDefinitionRepository) FindByName(ctx context.Context, name string) (*model.NodeDefinition, error) {
	query := `
		SELECT
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		FROM node_definitions
		WHERE name = $1`

	var row nodeRow
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&row.ID,
		&row.Name,
		&row.Type,
		&row.Category,
		&row.Description,
		&row.Icon,
		&row.Color,
		&row.Version,
		&row.Status,
		&row.Inputs,
		&row.Outputs,
		&row.Properties,
		&row.Configuration,
		&row.ValidationRules,
		&row.Documentation,
		&row.Examples,
		pq.Array(&row.Tags),
		&row.Author,
		&row.IsSystem,
		&row.IsPremium,
		&row.ExecutionHandler,
		&row.Metadata,
		&row.CreatedAt,
		&row.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, fmt.Errorf("failed to find node definition: %w", err)
	}

	return row.toDomain()
}

// FindByType finds node definitions by type
func (r *NodeDefinitionRepository) FindByType(ctx context.Context, nodeType model.NodeType, offset, limit int) ([]*model.NodeDefinition, error) {
	return r.findBy(ctx, "type = $1", string(nodeType), offset, limit)
}

// FindByCategory finds node definitions by category
func (r *NodeDefinitionRepository) FindByCategory(ctx context.Context, category model.NodeCategory, offset, limit int) ([]*model.NodeDefinition, error) {
	return r.findBy(ctx, "category = $1", string(category), offset, limit)
}

// FindByStatus finds node definitions by status
func (r *NodeDefinitionRepository) FindByStatus(ctx context.Context, status model.NodeStatus, offset, limit int) ([]*model.NodeDefinition, error) {
	return r.findBy(ctx, "status = $1", string(status), offset, limit)
}

// FindAll finds all node definitions
func (r *NodeDefinitionRepository) FindAll(ctx context.Context, offset, limit int) ([]*model.NodeDefinition, error) {
	query := `
		SELECT
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		FROM node_definitions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find node definitions: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// FindSystemNodes finds all system nodes
func (r *NodeDefinitionRepository) FindSystemNodes(ctx context.Context, offset, limit int) ([]*model.NodeDefinition, error) {
	return r.findBy(ctx, "is_system = $1", true, offset, limit)
}

// Search searches node definitions by name or description
func (r *NodeDefinitionRepository) Search(ctx context.Context, query string, offset, limit int) ([]*model.NodeDefinition, error) {
	sqlQuery := `
		SELECT
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		FROM node_definitions
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, sqlQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search node definitions: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

// Count counts total node definitions
func (r *NodeDefinitionRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM node_definitions`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count node definitions: %w", err)
	}

	return count, nil
}

// CountByType counts node definitions by type
func (r *NodeDefinitionRepository) CountByType(ctx context.Context, nodeType model.NodeType) (int64, error) {
	query := `SELECT COUNT(*) FROM node_definitions WHERE type = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, string(nodeType)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count node definitions: %w", err)
	}

	return count, nil
}

// CountByCategory counts node definitions by category
func (r *NodeDefinitionRepository) CountByCategory(ctx context.Context, category model.NodeCategory) (int64, error) {
	query := `SELECT COUNT(*) FROM node_definitions WHERE category = $1`
	
	var count int64
	err := r.db.QueryRowContext(ctx, query, string(category)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count node definitions: %w", err)
	}

	return count, nil
}

// Delete deletes a node definition
func (r *NodeDefinitionRepository) Delete(ctx context.Context, id model.NodeID) error {
	query := `DELETE FROM node_definitions WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return fmt.Errorf("failed to delete node definition: %w", err)
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

func (r *NodeDefinitionRepository) findBy(ctx context.Context, condition string, value interface{}, offset, limit int) ([]*model.NodeDefinition, error) {
	query := fmt.Sprintf(`
		SELECT
			id, name, type, category, description, icon, color, version, status,
			inputs, outputs, properties, configuration, validation_rules,
			documentation, examples, tags, author, is_system, is_premium,
			execution_handler, metadata, created_at, updated_at
		FROM node_definitions
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, condition)

	rows, err := r.db.QueryContext(ctx, query, value, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to find node definitions: %w", err)
	}
	defer rows.Close()

	return r.scanRows(rows)
}

func (r *NodeDefinitionRepository) scanRows(rows *sql.Rows) ([]*model.NodeDefinition, error) {
	var nodes []*model.NodeDefinition
	
	for rows.Next() {
		var row nodeRow
		err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.Type,
			&row.Category,
			&row.Description,
			&row.Icon,
			&row.Color,
			&row.Version,
			&row.Status,
			&row.Inputs,
			&row.Outputs,
			&row.Properties,
			&row.Configuration,
			&row.ValidationRules,
			&row.Documentation,
			&row.Examples,
			pq.Array(&row.Tags),
			&row.Author,
			&row.IsSystem,
			&row.IsPremium,
			&row.ExecutionHandler,
			&row.Metadata,
			&row.CreatedAt,
			&row.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan node definition: %w", err)
		}

		node, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

// nodeRow represents a database row for node definition
type nodeRow struct {
	ID               string
	Name             string
	Type             string
	Category         string
	Description      sql.NullString
	Icon             sql.NullString
	Color            sql.NullString
	Version          string
	Status           string
	Inputs           []byte
	Outputs          []byte
	Properties       []byte
	Configuration    []byte
	ValidationRules  []byte
	Documentation    sql.NullString
	Examples         []byte
	Tags             []string
	Author           sql.NullString
	IsSystem         bool
	IsPremium        bool
	ExecutionHandler sql.NullString
	Metadata         []byte
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// toDomain converts a database row to domain model
func (r *nodeRow) toDomain() (*model.NodeDefinition, error) {
	// Create base node
	node, err := model.NewNodeDefinition(
		r.Name,
		model.NodeType(r.Type),
		model.NodeCategory(r.Category),
		r.Description.String,
	)
	if err != nil {
		return nil, err
	}

	// Set additional properties
	// Note: We'd need to expose setters or use a reconstruction method
	// For now, returning the basic node
	
	return node, nil
}
