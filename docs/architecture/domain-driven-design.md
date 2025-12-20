# Domain-Driven Design

LinkFlow AI follows Domain-Driven Design (DDD) principles to organize code around business domains. This document explains our DDD implementation.

## Directory Structure

Each domain follows a consistent structure:

```
internal/<domain>/
├── domain/                 # Core business logic (innermost layer)
│   ├── model/              # Entities, Value Objects, Aggregates
│   │   └── <entity>.go
│   └── repository/         # Repository interfaces
│       └── interfaces.go
├── adapters/               # External interfaces (outermost layer)
│   ├── http/               # HTTP handlers
│   │   ├── handlers/
│   │   │   └── <domain>_handler.go
│   │   └── dto/
│   │       └── <domain>_dto.go
│   └── repository/         # Repository implementations
│       └── postgres/
│           └── <domain>_repository.go
├── app/                    # Application services (middle layer)
│   └── service/
│       └── <domain>_service.go
├── ports/                  # Port interfaces (optional)
│   └── ports.go
└── server/                 # Server setup (optional)
    └── server.go
```

## Layer Responsibilities

### Domain Layer (`domain/`)

The innermost layer containing pure business logic with no external dependencies.

**Entities** - Objects with identity:
```go
// internal/workflow/domain/model/workflow.go
type Workflow struct {
    id          WorkflowID
    userID      string
    name        string
    description string
    nodes       []Node
    connections []Connection
    status      WorkflowStatus
    createdAt   time.Time
    updatedAt   time.Time
    version     int
}

func NewWorkflow(userID, name, description string) (*Workflow, error) {
    if userID == "" {
        return nil, errors.New("user ID is required")
    }
    return &Workflow{
        id:        NewWorkflowID(),
        userID:    userID,
        name:      name,
        status:    StatusDraft,
        createdAt: time.Now(),
        version:   1,
    }, nil
}
```

**Value Objects** - Immutable objects without identity:
```go
type WorkflowID string

func NewWorkflowID() WorkflowID {
    return WorkflowID(uuid.New().String())
}

type Position struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}
```

**Repository Interfaces**:
```go
// internal/workflow/domain/repository/interfaces.go
type WorkflowRepository interface {
    Save(ctx context.Context, workflow *model.Workflow) error
    FindByID(ctx context.Context, id model.WorkflowID) (*model.Workflow, error)
    FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*model.Workflow, error)
    Update(ctx context.Context, workflow *model.Workflow) error
    Delete(ctx context.Context, id model.WorkflowID) error
}
```

### Application Layer (`app/`)

Orchestrates domain objects to perform use cases.

```go
// internal/workflow/app/service/workflow_service.go
type WorkflowService struct {
    repo       repository.WorkflowRepository
    eventBus   events.EventBus
}

func NewWorkflowService(repo repository.WorkflowRepository, eventBus events.EventBus) *WorkflowService {
    return &WorkflowService{repo: repo, eventBus: eventBus}
}

func (s *WorkflowService) CreateWorkflow(ctx context.Context, userID, name, description string) (*model.Workflow, error) {
    // Business logic orchestration
    workflow, err := model.NewWorkflow(userID, name, description)
    if err != nil {
        return nil, err
    }

    if err := s.repo.Save(ctx, workflow); err != nil {
        return nil, err
    }

    // Publish domain event
    s.eventBus.Publish(events.WorkflowCreated{
        WorkflowID: workflow.ID().String(),
        UserID:     userID,
    })

    return workflow, nil
}
```

### Adapters Layer (`adapters/`)

Implements interfaces defined in the domain layer.

**HTTP Handlers**:
```go
// internal/workflow/adapters/http/handlers/workflow_handler.go
type WorkflowHandler struct {
    service *service.WorkflowService
}

func NewWorkflowHandler(service *service.WorkflowService) *WorkflowHandler {
    return &WorkflowHandler{service: service}
}

func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req dto.CreateWorkflowRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    userID := middleware.GetUserID(r.Context())
    workflow, err := h.service.CreateWorkflow(r.Context(), userID, req.Name, req.Description)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(dto.WorkflowResponse{
        ID:   workflow.ID().String(),
        Name: workflow.Name(),
    })
}
```

**Repository Implementations**:
```go
// internal/workflow/adapters/repository/postgres/workflow_repository.go
type WorkflowRepository struct {
    db *database.DB
}

func NewWorkflowRepository(db *database.DB) repository.WorkflowRepository {
    return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) Save(ctx context.Context, workflow *model.Workflow) error {
    query := `INSERT INTO workflows (id, user_id, name, ...) VALUES ($1, $2, $3, ...)`
    _, err := r.db.ExecContext(ctx, query, workflow.ID(), workflow.UserID(), workflow.Name())
    return err
}
```

## Dependency Rule

Dependencies point inward:

```
┌─────────────────────────────────────────────────┐
│                  Adapters                        │
│  ┌─────────────────────────────────────────┐    │
│  │              Application                 │    │
│  │  ┌─────────────────────────────────┐    │    │
│  │  │            Domain               │    │    │
│  │  │                                 │    │    │
│  │  │  • Entities                     │    │    │
│  │  │  • Value Objects                │    │    │
│  │  │  • Repository Interfaces        │    │    │
│  │  │  • Domain Services              │    │    │
│  │  │                                 │    │    │
│  │  └─────────────────────────────────┘    │    │
│  │                                         │    │
│  │  • Application Services                 │    │
│  │  • Use Cases                            │    │
│  │                                         │    │
│  └─────────────────────────────────────────┘    │
│                                                  │
│  • HTTP Handlers                                 │
│  • Repository Implementations                    │
│  • External Service Clients                      │
│                                                  │
└─────────────────────────────────────────────────┘
```

## Domain Events

Domain events communicate changes across bounded contexts:

```go
// internal/shared/events/events.go
type WorkflowCreated struct {
    WorkflowID string
    UserID     string
    Timestamp  time.Time
}

type WorkflowExecuted struct {
    WorkflowID  string
    ExecutionID string
    Status      string
    Timestamp   time.Time
}
```

## Bounded Contexts

Each domain is a bounded context with its own:
- Ubiquitous language
- Domain model
- Data storage

| Context | Responsibility | Key Entities |
|---------|---------------|--------------|
| Workflow | Workflow definitions | Workflow, Node, Connection |
| Execution | Runtime execution | Execution, NodeExecution |
| Credential | Secret storage | Credential, CredentialType |
| Integration | External services | Integration, OAuthToken |
| Schedule | Time-based triggers | Schedule |
| Auth | Identity management | User, Session, APIKey |
| Billing | Subscription management | Plan, Subscription, Invoice |

## Best Practices

1. **Keep domain layer pure**: No framework dependencies
2. **Use interfaces**: Define contracts in domain, implement in adapters
3. **Validate in domain**: Business rules belong in entities
4. **Use value objects**: For concepts without identity
5. **Emit domain events**: For cross-context communication
6. **Test domain logic**: Unit test without infrastructure

## Next Steps

- [Service Architecture](services.md)
- [API Documentation](../api/overview.md)
- [Adding New Domains](../architecture/domain-driven-design.md)
