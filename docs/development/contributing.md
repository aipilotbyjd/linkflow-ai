# Contributing Guide

Thank you for your interest in contributing to LinkFlow AI!

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Git

### Development Setup

1. **Fork and clone the repository**
```bash
git clone https://github.com/your-username/linkflow-ai.git
cd linkflow-ai
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up the database**
```bash
# Create database
createdb linkflow_dev

# Run migrations
export DATABASE_URL="postgres://localhost:5432/linkflow_dev?sslmode=disable"
go run ./cmd/tools/migrate up
```

4. **Create environment file**
```bash
cp .env.example .env
# Edit .env with your settings
```

5. **Run the server**
```bash
go run ./cmd/services/api
```

## Development Workflow

### Branch Naming

- `feature/` - New features
- `fix/` - Bug fixes
- `refactor/` - Code refactoring
- `docs/` - Documentation updates
- `test/` - Test additions/updates

Example: `feature/add-discord-integration`

### Commit Messages

Follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation
- `refactor` - Code refactoring
- `test` - Tests
- `chore` - Maintenance

Examples:
```
feat(nodes): add Discord integration node
fix(execution): handle timeout properly in worker pool
docs(api): update authentication documentation
refactor(workflow): extract validation logic
test(nodes): add unit tests for HTTP request node
```

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes
3. Write/update tests
4. Ensure all tests pass
5. Update documentation if needed
6. Submit pull request

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
- [ ] Commit messages follow convention
- [ ] Branch is up to date with main

## Code Organization

### Adding a New Domain

1. Create directory structure:
```
internal/newdomain/
├── domain/
│   ├── model/
│   │   └── newdomain.go
│   └── repository/
│       └── interfaces.go
├── adapters/
│   ├── http/
│   │   ├── handlers/
│   │   └── dto/
│   └── repository/
│       └── postgres/
├── app/
│   └── service/
└── server/
```

2. Implement domain model
3. Define repository interface
4. Implement repository
5. Create application service
6. Add HTTP handlers
7. Wire in main

### Adding a New Node

1. Create node file in `internal/node/runtime/nodes/`
2. Implement `NodeExecutor` interface
3. Register in `init()` function
4. Add tests
5. Update documentation

See [Adding Nodes Guide](../guides/adding-nodes.md) for details.

### Adding a New Integration

1. Create connector in `internal/integration/connectors/`
2. Implement `Connector` interface
3. Add OAuth configuration if needed
4. Register connector
5. Add tests

## Testing

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/workflow/...

# Verbose output
go test -v ./...
```

### Writing Tests

```go
func TestWorkflowService_Create(t *testing.T) {
    // Arrange
    repo := mocks.NewWorkflowRepository()
    service := NewWorkflowService(repo)

    // Act
    workflow, err := service.Create(ctx, "user-1", "Test", "Description")

    // Assert
    assert.NoError(t, err)
    assert.NotEmpty(t, workflow.ID)
    assert.Equal(t, "Test", workflow.Name)
}
```

### Test Categories

- **Unit tests**: Test individual functions/methods
- **Integration tests**: Test with real database
- **E2E tests**: Test complete workflows

## Code Style

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` for formatting
- Run `go vet` before committing

### Naming Conventions

```go
// Package names: lowercase, single word
package workflow

// Interfaces: verb or noun describing behavior
type Repository interface { ... }
type Executor interface { ... }

// Structs: noun
type Workflow struct { ... }
type ExecutionService struct { ... }

// Methods: verb phrase
func (s *Service) CreateWorkflow(...) { ... }
func (w *Workflow) Activate() { ... }

// Constants: MixedCaps or ALL_CAPS
const MaxRetries = 3
const DEFAULT_TIMEOUT = "30m"
```

### Error Handling

```go
// Return errors, don't panic
func (s *Service) Create(ctx context.Context, name string) (*Model, error) {
    if name == "" {
        return nil, ErrNameRequired
    }
    // ...
}

// Wrap errors with context
if err != nil {
    return nil, fmt.Errorf("failed to create workflow: %w", err)
}

// Define domain errors
var (
    ErrNotFound = errors.New("not found")
    ErrInvalidInput = errors.New("invalid input")
)
```

### Comments

```go
// Package workflow provides workflow management functionality.
package workflow

// Workflow represents an automation workflow.
// It contains nodes and connections that define the automation logic.
type Workflow struct {
    // ...
}

// Create creates a new workflow for the given user.
// It validates the input and returns an error if invalid.
func (s *Service) Create(ctx context.Context, userID, name string) (*Workflow, error) {
    // ...
}
```

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use complete sentences
- Include examples for complex functions

### API Documentation

- Update OpenAPI specs in `api/`
- Update markdown docs in `docs/api/`

### User Documentation

- Update guides in `docs/guides/`
- Update reference docs in `docs/reference/`

## Review Process

### What We Look For

1. **Correctness**: Does it work correctly?
2. **Testing**: Are there adequate tests?
3. **Performance**: Any performance concerns?
4. **Security**: Any security implications?
5. **Maintainability**: Is the code clean and maintainable?
6. **Documentation**: Is it documented?

### Common Feedback

- Add error handling
- Add context to errors
- Add tests for edge cases
- Simplify complex logic
- Use existing utilities
- Follow naming conventions

## Getting Help

- Open an issue for bugs or features
- Join our Discord for discussions
- Read existing code for examples
- Check documentation

## License

By contributing, you agree that your contributions will be licensed under the project's license.

## Thank You!

We appreciate your contributions to LinkFlow AI!
