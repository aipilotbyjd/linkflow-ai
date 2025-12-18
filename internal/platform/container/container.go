// Package container provides dependency injection
package container

import (
	"context"
	"fmt"
	"sync"
)

// ServiceContainer manages service dependencies
type ServiceContainer struct {
	services map[string]interface{}
	factories map[string]Factory
	mu       sync.RWMutex
}

// Factory creates service instances
type Factory func(c *ServiceContainer) (interface{}, error)

// New creates a new service container
func New() *ServiceContainer {
	return &ServiceContainer{
		services:  make(map[string]interface{}),
		factories: make(map[string]Factory),
	}
}

// Register registers a service instance
func (c *ServiceContainer) Register(name string, service interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.services[name] = service
}

// RegisterFactory registers a service factory
func (c *ServiceContainer) RegisterFactory(name string, factory Factory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories[name] = factory
}

// Get retrieves a service by name
func (c *ServiceContainer) Get(name string) (interface{}, error) {
	c.mu.RLock()
	if service, ok := c.services[name]; ok {
		c.mu.RUnlock()
		return service, nil
	}
	c.mu.RUnlock()

	// Try factory
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after acquiring write lock
	if service, ok := c.services[name]; ok {
		return service, nil
	}

	factory, ok := c.factories[name]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	service, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create service %s: %w", name, err)
	}

	c.services[name] = service
	return service, nil
}

// MustGet retrieves a service or panics
func (c *ServiceContainer) MustGet(name string) interface{} {
	service, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return service
}

// Has checks if a service exists
func (c *ServiceContainer) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, hasService := c.services[name]
	_, hasFactory := c.factories[name]
	return hasService || hasFactory
}

// ServiceProvider configures services in the container
type ServiceProvider interface {
	Register(c *ServiceContainer) error
	Boot(ctx context.Context, c *ServiceContainer) error
}

// App represents the application with dependency injection
type App struct {
	container *ServiceContainer
	providers []ServiceProvider
}

// NewApp creates a new application
func NewApp() *App {
	return &App{
		container: New(),
		providers: []ServiceProvider{},
	}
}

// Container returns the service container
func (a *App) Container() *ServiceContainer {
	return a.container
}

// AddProvider adds a service provider
func (a *App) AddProvider(provider ServiceProvider) *App {
	a.providers = append(a.providers, provider)
	return a
}

// Boot initializes all providers
func (a *App) Boot(ctx context.Context) error {
	// Register phase
	for _, provider := range a.providers {
		if err := provider.Register(a.container); err != nil {
			return fmt.Errorf("failed to register provider: %w", err)
		}
	}

	// Boot phase
	for _, provider := range a.providers {
		if err := provider.Boot(ctx, a.container); err != nil {
			return fmt.Errorf("failed to boot provider: %w", err)
		}
	}

	return nil
}

// Service names
const (
	ServiceDatabase          = "database"
	ServiceCache             = "cache"
	ServiceWorkflowRepo      = "workflow.repository"
	ServiceWorkflowService   = "workflow.service"
	ServiceExecutionEngine   = "execution.engine"
	ServiceNodeRegistry      = "node.registry"
	ServiceCredentialRepo    = "credential.repository"
	ServiceCredentialService = "credential.service"
	ServiceUserRepo          = "user.repository"
	ServiceUserService       = "user.service"
	ServiceAuthService       = "auth.service"
	ServiceWebhookService    = "webhook.service"
	ServiceBillingService    = "billing.service"
	ServiceNotificationService = "notification.service"
)

// ServiceHealthCheck represents service health
type ServiceHealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthChecker checks service health
type HealthChecker interface {
	HealthCheck(ctx context.Context) *ServiceHealthCheck
}

// CheckHealth checks all service health
func (c *ServiceContainer) CheckHealth(ctx context.Context) []*ServiceHealthCheck {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var checks []*ServiceHealthCheck
	for name, service := range c.services {
		if checker, ok := service.(HealthChecker); ok {
			checks = append(checks, checker.HealthCheck(ctx))
		} else {
			checks = append(checks, &ServiceHealthCheck{
				Name:   name,
				Status: "healthy",
			})
		}
	}
	return checks
}
