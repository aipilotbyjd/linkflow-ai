# Coding Standards

Code style and conventions for LinkFlow AI development.

## Go Style Guide

### Formatting

All code must be formatted with `gofmt`:

```bash
gofmt -w .
```

### Imports

Group imports in this order:
1. Standard library
2. External packages
3. Internal packages

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/mux"

    "github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"
    "github.com/linkflow-ai/linkflow-ai/pkg/middleware"
)
```

### Naming Conventions

#### Packages

```go
// Good - lowercase, single word
package workflow
package execution
package auth

// Bad
package workflowService
package WorkFlow
package workflow_service
```

#### Variables

```go
// Good - camelCase
userID := "123"
workflowName := "My Workflow"
maxRetries := 3

// Bad
UserID := "123"
workflow_name := "My Workflow"
MAXRETRIES := 3
```

#### Constants

```go
// Exported - MixedCaps
const MaxRetries = 3
const DefaultTimeout = 30 * time.Second

// Unexported - camelCase or MixedCaps
const defaultBufferSize = 1024
const internalPrefix = "_internal_"
```

#### Functions and Methods

```go
// Good - verb phrase describing action
func CreateWorkflow(ctx context.Context, name string) (*Workflow, error)
func (w *Workflow) Activate() error
func (s *Service) GetByID(ctx context.Context, id string) (*Model, error)

// Bad
func Workflow(ctx context.Context, name string) (*Workflow, error)
func (w *Workflow) DoActivate() error
```

#### Interfaces

```go
// Good - behavior description
type Reader interface { Read(p []byte) (n int, err error) }
type WorkflowRepository interface { Save(ctx context.Context, w *Workflow) error }
type Executor interface { Execute(ctx context.Context, node *Node) error }

// Bad
type IWorkflowRepository interface { ... }
type WorkflowRepositoryInterface interface { ... }
```

#### Structs

```go
// Good - noun describing entity
type Workflow struct { ... }
type ExecutionService struct { ... }
type Config struct { ... }

// Bad
type WorkflowData struct { ... }
type ServiceExecution struct { ... }
```

### Error Handling

#### Return Errors

```go
// Good - return errors to caller
func (s *Service) Create(ctx context.Context, name string) (*Model, error) {
    if name == "" {
        return nil, ErrNameRequired
    }
    return s.repo.Save(ctx, &Model{Name: name})
}

// Bad - panic
func (s *Service) Create(ctx context.Context, name string) *Model {
    if name == "" {
        panic("name required")
    }
    // ...
}
```

#### Wrap Errors

```go
// Good - add context to errors
if err := s.repo.Save(ctx, workflow); err != nil {
    return fmt.Errorf("failed to save workflow %s: %w", workflow.ID, err)
}

// Bad - lose context
if err := s.repo.Save(ctx, workflow); err != nil {
    return err
}
```

#### Define Domain Errors

```go
// internal/workflow/domain/repository/errors.go
package repository

import "errors"

var (
    ErrNotFound       = errors.New("workflow not found")
    ErrDuplicateName  = errors.New("workflow name already exists")
    ErrInvalidStatus  = errors.New("invalid workflow status")
)
```

#### Check Errors Immediately

```go
// Good
result, err := doSomething()
if err != nil {
    return err
}
// Use result

// Bad
result, err := doSomething()
// Use result before checking error
fmt.Println(result)
if err != nil {
    return err
}
```

### Context

#### Always Accept Context

```go
// Good
func (s *Service) Get(ctx context.Context, id string) (*Model, error)
func (r *Repository) FindByID(ctx context.Context, id ModelID) (*Model, error)

// Bad
func (s *Service) Get(id string) (*Model, error)
```

#### Pass Context Down

```go
func (s *Service) Create(ctx context.Context, input CreateInput) (*Model, error) {
    // Pass context to repository
    return s.repo.Save(ctx, model)
}
```

### Structs

#### Use Constructor Functions

```go
// Good
func NewWorkflow(userID, name, description string) (*Workflow, error) {
    if userID == "" {
        return nil, errors.New("user ID required")
    }
    return &Workflow{
        id:          NewWorkflowID(),
        userID:      userID,
        name:        name,
        description: description,
        status:      StatusDraft,
        createdAt:   time.Now(),
    }, nil
}

// Usage
workflow, err := NewWorkflow("user-123", "My Workflow", "")
```

#### Keep Fields Private

```go
// Good - encapsulated
type Workflow struct {
    id          WorkflowID
    name        string
    status      Status
}

func (w *Workflow) ID() WorkflowID { return w.id }
func (w *Workflow) Name() string { return w.name }
func (w *Workflow) UpdateName(name string) error {
    if name == "" {
        return errors.New("name required")
    }
    w.name = name
    return nil
}

// Bad - exposed fields
type Workflow struct {
    ID     string
    Name   string
    Status string
}
```

### Functions

#### Keep Functions Short

Functions should do one thing and be < 40 lines.

```go
// Good - single responsibility
func (s *Service) validateWorkflow(w *Workflow) error {
    if w.Name() == "" {
        return ErrNameRequired
    }
    if len(w.Nodes()) == 0 {
        return ErrNoNodes
    }
    return s.detectCycles(w)
}

func (s *Service) detectCycles(w *Workflow) error {
    // Separate function for cycle detection
}
```

#### Limit Parameters

```go
// Good - use options struct for many parameters
type CreateOptions struct {
    Name        string
    Description string
    FolderID    string
    Tags        []string
}

func (s *Service) Create(ctx context.Context, userID string, opts CreateOptions) (*Workflow, error)

// Bad - too many parameters
func (s *Service) Create(ctx context.Context, userID, name, description, folderID string, tags []string) (*Workflow, error)
```

### Comments

#### Document Exported Items

```go
// WorkflowService provides workflow management operations.
type WorkflowService struct {
    repo repository.WorkflowRepository
}

// Create creates a new workflow for the specified user.
// It validates the input and returns ErrNameRequired if name is empty.
func (s *WorkflowService) Create(ctx context.Context, userID, name string) (*Workflow, error) {
    // ...
}
```

#### Explain Why, Not What

```go
// Good - explains why
// Use buffered channel to prevent blocking the sender
// when receivers are slow
ch := make(chan Event, 100)

// Bad - explains what (obvious from code)
// Create a channel with buffer size 100
ch := make(chan Event, 100)
```

### Testing

#### Test File Location

Place tests next to the code:

```
internal/workflow/
├── app/
│   └── service/
│       ├── workflow_service.go
│       └── workflow_service_test.go
└── domain/
    └── model/
        ├── workflow.go
        └── workflow_test.go
```

#### Test Naming

```go
// Function tests
func TestWorkflowService_Create(t *testing.T)
func TestWorkflowService_Create_InvalidInput(t *testing.T)

// Method tests
func TestWorkflow_Activate(t *testing.T)
func TestWorkflow_Activate_AlreadyActive(t *testing.T)
```

### Concurrency

#### Use sync.Mutex for Simple Cases

```go
type Counter struct {
    mu    sync.Mutex
    value int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}
```

#### Use Channels for Communication

```go
// Good - channel for coordination
results := make(chan Result, len(items))
for _, item := range items {
    go func(i Item) {
        results <- process(i)
    }(item)
}

// Collect results
for range items {
    result := <-results
    // ...
}
```

### Project Structure

#### Domain-Driven Design

```
internal/<domain>/
├── domain/           # Core business logic
│   ├── model/        # Entities, value objects
│   └── repository/   # Repository interfaces
├── adapters/         # External interfaces
│   ├── http/         # HTTP handlers
│   └── repository/   # Repository implementations
├── app/              # Application services
│   └── service/
└── server/           # Server setup
```

#### Dependency Direction

- Domain layer has no external dependencies
- Application layer depends on domain
- Adapters depend on application and domain
- Main wires everything together

## Linting

### golangci-lint Configuration

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell

linters-settings:
  errcheck:
    check-type-assertions: true
  govet:
    check-shadowing: true
```

### Run Linter

```bash
golangci-lint run ./...
```

## Pre-commit Checks

```bash
# Format code
gofmt -w .

# Run vet
go vet ./...

# Run linter
golangci-lint run ./...

# Run tests
go test ./...
```

## Next Steps

- [Testing Guide](testing.md)
- [Contributing Guide](contributing.md)
- [Architecture Overview](../architecture/overview.md)
