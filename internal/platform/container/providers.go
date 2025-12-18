// Package container provides service providers
package container

import (
	"context"

	"github.com/linkflow-ai/linkflow-ai/internal/platform/database"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/cache"
	"github.com/linkflow-ai/linkflow-ai/internal/platform/config"
)

// DatabaseProvider provides database service
type DatabaseProvider struct {
	Config config.DatabaseConfig
}

func (p *DatabaseProvider) Register(c *ServiceContainer) error {
	c.RegisterFactory(ServiceDatabase, func(c *ServiceContainer) (interface{}, error) {
		return database.New(p.Config)
	})
	return nil
}

func (p *DatabaseProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	_, err := c.Get(ServiceDatabase)
	return err
}

// CacheProvider provides cache service
type CacheProvider struct {
	Config cache.Config
}

func (p *CacheProvider) Register(c *ServiceContainer) error {
	c.RegisterFactory(ServiceCache, func(c *ServiceContainer) (interface{}, error) {
		return cache.NewRedisCache(p.Config)
	})
	return nil
}

func (p *CacheProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	_, err := c.Get(ServiceCache)
	return err
}

// WorkflowProvider provides workflow services
type WorkflowProvider struct{}

func (p *WorkflowProvider) Register(c *ServiceContainer) error {
	// Register workflow repository factory
	c.RegisterFactory(ServiceWorkflowRepo, func(c *ServiceContainer) (interface{}, error) {
		// Would get database from container and create repository
		return nil, nil
	})

	// Register workflow service factory
	c.RegisterFactory(ServiceWorkflowService, func(c *ServiceContainer) (interface{}, error) {
		return nil, nil
	})

	return nil
}

func (p *WorkflowProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	return nil
}

// ExecutionProvider provides execution engine
type ExecutionProvider struct{}

func (p *ExecutionProvider) Register(c *ServiceContainer) error {
	c.RegisterFactory(ServiceExecutionEngine, func(c *ServiceContainer) (interface{}, error) {
		return nil, nil
	})
	return nil
}

func (p *ExecutionProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	return nil
}

// AuthProvider provides authentication services
type AuthProvider struct{}

func (p *AuthProvider) Register(c *ServiceContainer) error {
	c.RegisterFactory(ServiceAuthService, func(c *ServiceContainer) (interface{}, error) {
		return nil, nil
	})
	return nil
}

func (p *AuthProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	return nil
}

// NodeProvider provides node registry
type NodeProvider struct{}

func (p *NodeProvider) Register(c *ServiceContainer) error {
	c.RegisterFactory(ServiceNodeRegistry, func(c *ServiceContainer) (interface{}, error) {
		return nil, nil
	})
	return nil
}

func (p *NodeProvider) Boot(ctx context.Context, c *ServiceContainer) error {
	return nil
}
