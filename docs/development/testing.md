# Testing Guide

Comprehensive guide to testing in LinkFlow AI.

## Test Structure

```
tests/
├── unit/                   # Unit tests
│   ├── workflow_test.go
│   ├── execution_test.go
│   └── node_test.go
├── integration/            # Integration tests
│   ├── auth_test.go
│   └── workflow_test.go
├── e2e/                    # End-to-end tests
│   ├── workflow_e2e_test.go
│   └── execution_e2e_test.go
├── security/               # Security tests
│   └── auth_security_test.go
└── helpers/                # Test utilities
    └── test_helpers.go
```

## Running Tests

### All Tests

```bash
go test ./...
```

### With Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# Coverage summary
go test -cover ./...
```

### Specific Package

```bash
go test ./internal/workflow/...
go test ./internal/execution/app/service/...
```

### Verbose Output

```bash
go test -v ./...
```

### Run Specific Test

```bash
go test -run TestWorkflowService_Create ./internal/workflow/...
go test -run "TestWorkflow.*" ./...
```

### Race Detection

```bash
go test -race ./...
```

### Benchmarks

```bash
go test -bench=. ./...
go test -bench=BenchmarkExecution ./internal/engine/...
```

## Unit Tests

Unit tests test individual functions in isolation.

### Structure

```go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/linkflow-ai/linkflow-ai/internal/workflow/app/service"
)

func TestWorkflowService_Create(t *testing.T) {
    // Arrange
    ctx := context.Background()
    mockRepo := new(MockWorkflowRepository)
    svc := service.NewWorkflowService(mockRepo)

    mockRepo.On("Save", mock.Anything, mock.Anything).Return(nil)

    // Act
    workflow, err := svc.Create(ctx, "user-123", "Test Workflow", "Description")

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, workflow)
    assert.Equal(t, "Test Workflow", workflow.Name())
    mockRepo.AssertExpectations(t)
}
```

### Table-Driven Tests

```go
func TestWorkflow_Validate(t *testing.T) {
    tests := []struct {
        name    string
        nodes   []Node
        conns   []Connection
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid workflow",
            nodes:   []Node{{ID: "1", Type: "trigger"}},
            conns:   []Connection{},
            wantErr: false,
        },
        {
            name:    "empty nodes",
            nodes:   []Node{},
            conns:   []Connection{},
            wantErr: true,
            errMsg:  "at least one node required",
        },
        {
            name: "circular dependency",
            nodes: []Node{
                {ID: "1", Type: "action"},
                {ID: "2", Type: "action"},
            },
            conns: []Connection{
                {Source: "1", Target: "2"},
                {Source: "2", Target: "1"},
            },
            wantErr: true,
            errMsg:  "cycle detected",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            workflow := &Workflow{nodes: tt.nodes, connections: tt.conns}
            err := workflow.Validate()

            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Mocking

Using testify/mock:

```go
// Mock definition
type MockWorkflowRepository struct {
    mock.Mock
}

func (m *MockWorkflowRepository) Save(ctx context.Context, w *Workflow) error {
    args := m.Called(ctx, w)
    return args.Error(0)
}

func (m *MockWorkflowRepository) FindByID(ctx context.Context, id WorkflowID) (*Workflow, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*Workflow), args.Error(1)
}

// Usage in test
func TestService_Get(t *testing.T) {
    mockRepo := new(MockWorkflowRepository)
    
    expectedWorkflow := &Workflow{id: "wf-123"}
    mockRepo.On("FindByID", mock.Anything, WorkflowID("wf-123")).
        Return(expectedWorkflow, nil)
    
    svc := NewService(mockRepo)
    result, err := svc.Get(ctx, "wf-123")
    
    assert.NoError(t, err)
    assert.Equal(t, expectedWorkflow, result)
}
```

## Integration Tests

Integration tests test components with real dependencies.

### Database Setup

```go
package integration_test

import (
    "context"
    "database/sql"
    "testing"

    _ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
    // Setup
    var err error
    testDB, err = sql.Open("postgres", os.Getenv("TEST_DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    // Run migrations
    runMigrations(testDB)

    // Run tests
    code := m.Run()

    // Cleanup
    testDB.Close()
    os.Exit(code)
}

func setupTest(t *testing.T) func() {
    // Start transaction
    tx, _ := testDB.Begin()

    // Return cleanup function
    return func() {
        tx.Rollback()
    }
}
```

### Integration Test Example

```go
func TestWorkflowRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    cleanup := setupTest(t)
    defer cleanup()

    repo := postgres.NewWorkflowRepository(testDB)
    ctx := context.Background()

    // Test Create
    workflow, _ := model.NewWorkflow("user-123", "Test", "")
    err := repo.Save(ctx, workflow)
    assert.NoError(t, err)

    // Test Read
    found, err := repo.FindByID(ctx, workflow.ID())
    assert.NoError(t, err)
    assert.Equal(t, workflow.Name(), found.Name())

    // Test Update
    workflow.UpdateName("Updated")
    err = repo.Update(ctx, workflow)
    assert.NoError(t, err)

    // Test Delete
    err = repo.Delete(ctx, workflow.ID())
    assert.NoError(t, err)
}
```

## E2E Tests

End-to-end tests test complete workflows.

```go
func TestWorkflowExecution_E2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    // Setup server
    srv := setupTestServer(t)
    defer srv.Close()

    client := &http.Client{}

    // Create workflow
    workflow := `{
        "name": "Test Workflow",
        "nodes": [
            {"id": "1", "type": "manual_trigger", "config": {}},
            {"id": "2", "type": "set", "config": {"values": {"result": "success"}}}
        ],
        "connections": [
            {"source": "1", "target": "2"}
        ]
    }`

    resp, err := client.Post(
        srv.URL+"/api/v1/workflows",
        "application/json",
        strings.NewReader(workflow),
    )
    assert.NoError(t, err)
    assert.Equal(t, 201, resp.StatusCode)

    var createResp map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&createResp)
    workflowID := createResp["data"].(map[string]interface{})["id"].(string)

    // Execute workflow
    execResp, err := client.Post(
        srv.URL+"/api/v1/execute",
        "application/json",
        strings.NewReader(fmt.Sprintf(`{"workflowId": "%s"}`, workflowID)),
    )
    assert.NoError(t, err)
    assert.Equal(t, 200, execResp.StatusCode)

    var execResult map[string]interface{}
    json.NewDecoder(execResp.Body).Decode(&execResult)
    assert.Equal(t, "completed", execResult["status"])
}
```

## Test Helpers

### Common Helpers

```go
// tests/helpers/test_helpers.go
package helpers

import (
    "context"
    "testing"
    "time"
)

// TestContext returns a context with timeout
func TestContext(t *testing.T) context.Context {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    t.Cleanup(cancel)
    return ctx
}

// MustCreateWorkflow creates a workflow or fails the test
func MustCreateWorkflow(t *testing.T, svc *WorkflowService, userID, name string) *Workflow {
    t.Helper()
    workflow, err := svc.Create(TestContext(t), userID, name, "")
    if err != nil {
        t.Fatalf("failed to create workflow: %v", err)
    }
    return workflow
}

// AssertEventually retries assertion until timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(100 * time.Millisecond)
    }
    t.Fatalf("condition not met within %v: %s", timeout, msg)
}
```

### Test Fixtures

```go
// tests/fixtures/fixtures.go
package fixtures

import "github.com/linkflow-ai/linkflow-ai/internal/workflow/domain/model"

func SimpleWorkflow() *model.Workflow {
    workflow, _ := model.NewWorkflow("test-user", "Test Workflow", "")
    workflow.AddNode(model.Node{
        ID:   "trigger",
        Type: "manual_trigger",
        Name: "Start",
    })
    return workflow
}

func ComplexWorkflow() *model.Workflow {
    workflow, _ := model.NewWorkflow("test-user", "Complex Workflow", "")
    // Add multiple nodes and connections
    return workflow
}
```

## Benchmarks

```go
func BenchmarkWorkflowExecution(b *testing.B) {
    engine := NewEngine()
    workflow := fixtures.SimpleWorkflow()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.Execute(context.Background(), workflow, nil)
    }
}

func BenchmarkNodeExecution(b *testing.B) {
    executor := NewHTTPRequestExecutor()
    node := &Node{
        Config: map[string]interface{}{
            "url":    "http://localhost:8080/health",
            "method": "GET",
        },
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        executor.Execute(context.Background(), node, nil)
    }
}
```

## Test Coverage Goals

| Package | Target Coverage |
|---------|-----------------|
| `domain/model` | 90%+ |
| `app/service` | 80%+ |
| `adapters/repository` | 70%+ |
| `adapters/http` | 60%+ |

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: postgres
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run tests
        env:
          DATABASE_URL: postgres://postgres:postgres@localhost:5432/test?sslmode=disable
        run: |
          go test -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

## Best Practices

1. **Test behavior, not implementation**
2. **Use table-driven tests for multiple cases**
3. **Mock external dependencies**
4. **Use meaningful test names**
5. **Keep tests fast**
6. **Test edge cases**
7. **Maintain test coverage**
8. **Clean up test data**

## Next Steps

- [Coding Standards](coding-standards.md)
- [Contributing Guide](contributing.md)
- [Architecture Overview](../architecture/overview.md)
