// Package di provides dependency injection container
package di

import (
	"context"
	"fmt"
	"sync"
)

// Container manages service dependencies
type Container struct {
	services map[string]interface{}
	factories map[string]Factory
	mu       sync.RWMutex
}

// Factory creates a service instance
type Factory func(c *Container) (interface{}, error)

// New creates a new DI container
func New() *Container {
	return &Container{
		services:  make(map[string]interface{}),
		factories: make(map[string]Factory),
	}
}

// Register registers a singleton service
func (c *Container) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = service
}

// RegisterFactory registers a factory function for lazy initialization
func (c *Container) RegisterFactory(name string, factory Factory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories[name] = factory
}

// Get retrieves a service by name
func (c *Container) Get(name string) (interface{}, error) {
	c.mu.RLock()
	if service, ok := c.services[name]; ok {
		c.mu.RUnlock()
		return service, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if service, ok := c.services[name]; ok {
		return service, nil
	}

	// Try factory
	if factory, ok := c.factories[name]; ok {
		service, err := factory(c)
		if err != nil {
			return nil, fmt.Errorf("factory error for %s: %w", name, err)
		}
		c.services[name] = service
		return service, nil
	}

	return nil, fmt.Errorf("service not found: %s", name)
}

// MustGet retrieves a service or panics
func (c *Container) MustGet(name string) interface{} {
	service, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return service
}

// Has checks if a service is registered
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.services[name]
	if !ok {
		_, ok = c.factories[name]
	}
	return ok
}

// Close closes all closeable services
func (c *Container) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs []error
	for name, service := range c.services {
		if closer, ok := service.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
			}
		}
		if closerCtx, ok := service.(interface{ Close(context.Context) error }); ok {
			if err := closerCtx.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("failed to close %s: %w", name, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing services: %v", errs)
	}
	return nil
}

// Common service names
const (
	ServiceDB             = "db"
	ServiceLogger         = "logger"
	ServiceConfig         = "config"
	ServiceAuthRepo       = "auth.repository"
	ServiceAuthService    = "auth.service"
	ServiceWorkflowRepo   = "workflow.repository"
	ServiceWorkflowService = "workflow.service"
	ServiceExecutionRepo  = "execution.repository"
	ServiceExecutionService = "execution.service"
	ServiceCredentialRepo = "credential.repository"
	ServiceCredentialService = "credential.service"
	ServiceNotificationRepo = "notification.repository"
	ServiceNotificationService = "notification.service"
	ServiceWebhookRepo    = "webhook.repository"
	ServiceWebhookService = "webhook.service"
	ServiceScheduleRepo   = "schedule.repository"
	ServiceScheduleService = "schedule.service"
	ServiceBillingRepo    = "billing.repository"
	ServiceBillingService = "billing.service"
	ServiceWorkspaceRepo  = "workspace.repository"
	ServiceWorkspaceService = "workspace.service"
	ServiceIntegrationRepo = "integration.repository"
	ServiceIntegrationService = "integration.service"
	ServiceNodeRegistry   = "node.registry"
	ServiceEngine         = "engine"
	ServiceWorkerPool     = "worker.pool"
	ServiceEventBus       = "event.bus"
	ServiceCache          = "cache"
)

// ServiceRegistry provides typed service accessors
type ServiceRegistry struct {
	container *Container
}

// NewServiceRegistry wraps a container with typed accessors
func NewServiceRegistry(c *Container) *ServiceRegistry {
	return &ServiceRegistry{container: c}
}

// Container returns the underlying container
func (r *ServiceRegistry) Container() *Container {
	return r.container
}

// Builder helps construct a container with common services
type Builder struct {
	container *Container
	errors    []error
}

// NewBuilder creates a new container builder
func NewBuilder() *Builder {
	return &Builder{
		container: New(),
	}
}

// WithService adds a service to the container
func (b *Builder) WithService(name string, service interface{}) *Builder {
	b.container.Register(name, service)
	return b
}

// WithFactory adds a factory to the container
func (b *Builder) WithFactory(name string, factory Factory) *Builder {
	b.container.RegisterFactory(name, factory)
	return b
}

// Build returns the container or an error
func (b *Builder) Build() (*Container, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("builder errors: %v", b.errors)
	}
	return b.container, nil
}

// MustBuild returns the container or panics
func (b *Builder) MustBuild() *Container {
	c, err := b.Build()
	if err != nil {
		panic(err)
	}
	return c
}

// ServiceProvider interface for services that can register themselves
type ServiceProvider interface {
	Register(c *Container) error
}

// RegisterProviders registers multiple service providers
func (c *Container) RegisterProviders(providers ...ServiceProvider) error {
	for _, p := range providers {
		if err := p.Register(c); err != nil {
			return err
		}
	}
	return nil
}
