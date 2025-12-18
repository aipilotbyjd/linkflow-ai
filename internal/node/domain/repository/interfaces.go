package repository

import (
	"context"
	"errors"

	"github.com/linkflow-ai/linkflow-ai/internal/node/domain/model"
)

var (
	// ErrNotFound is returned when a node is not found
	ErrNotFound = errors.New("node not found")
	
	// ErrDuplicateName is returned when a node with the same name already exists
	ErrDuplicateName = errors.New("node with this name already exists")
)

// NodeDefinitionRepository defines the interface for node definition persistence
type NodeDefinitionRepository interface {
	// Save saves a new node definition
	Save(ctx context.Context, node *model.NodeDefinition) error
	
	// Update updates an existing node definition
	Update(ctx context.Context, node *model.NodeDefinition) error
	
	// FindByID finds a node definition by ID
	FindByID(ctx context.Context, id model.NodeID) (*model.NodeDefinition, error)
	
	// FindByName finds a node definition by name
	FindByName(ctx context.Context, name string) (*model.NodeDefinition, error)
	
	// FindByType finds node definitions by type
	FindByType(ctx context.Context, nodeType model.NodeType, offset, limit int) ([]*model.NodeDefinition, error)
	
	// FindByCategory finds node definitions by category
	FindByCategory(ctx context.Context, category model.NodeCategory, offset, limit int) ([]*model.NodeDefinition, error)
	
	// FindByStatus finds node definitions by status
	FindByStatus(ctx context.Context, status model.NodeStatus, offset, limit int) ([]*model.NodeDefinition, error)
	
	// FindAll finds all node definitions
	FindAll(ctx context.Context, offset, limit int) ([]*model.NodeDefinition, error)
	
	// FindSystemNodes finds all system nodes
	FindSystemNodes(ctx context.Context, offset, limit int) ([]*model.NodeDefinition, error)
	
	// Search searches node definitions by name or description
	Search(ctx context.Context, query string, offset, limit int) ([]*model.NodeDefinition, error)
	
	// Count counts total node definitions
	Count(ctx context.Context) (int64, error)
	
	// CountByType counts node definitions by type
	CountByType(ctx context.Context, nodeType model.NodeType) (int64, error)
	
	// CountByCategory counts node definitions by category
	CountByCategory(ctx context.Context, category model.NodeCategory) (int64, error)
	
	// Delete deletes a node definition
	Delete(ctx context.Context, id model.NodeID) error
}
