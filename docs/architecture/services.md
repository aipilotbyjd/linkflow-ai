# Service Architecture

LinkFlow AI is organized as a modular monolith with clear service boundaries. This document describes the services and their interactions.

## Service Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│         cmd/services/api/main.go, internal/gateway/              │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│   Workflow    │    │   Execution   │    │     Auth      │
│   Service     │    │   Service     │    │   Service     │
└───────┬───────┘    └───────┬───────┘    └───────┬───────┘
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                    ▼                 ▼
           ┌───────────────┐  ┌───────────────┐
           │   PostgreSQL  │  │     Redis     │
           └───────────────┘  └───────────────┘
```

## Core Services

### 1. API Service (`cmd/services/api/`)

**Purpose**: Main HTTP API server

**Responsibilities**:
- REST API endpoints
- Request routing
- Middleware chain (CORS, rate limiting, auth)
- Request validation

**Key Files**:
```
cmd/services/api/
└── main.go              # Server entry point

internal/gateway/
├── handlers/            # Route handlers
├── middleware/          # Gateway middleware
├── realtime/            # WebSocket handling
└── server/              # Server setup
```

### 2. Workflow Service (`cmd/services/workflow/`)

**Purpose**: Workflow management

**Responsibilities**:
- Workflow CRUD operations
- Workflow validation (cycle detection)
- Version management
- Template management
- Workflow sharing

**Key Files**:
```
internal/workflow/
├── domain/model/workflow.go           # Workflow entity
├── adapters/http/handlers/            # HTTP handlers
├── adapters/repository/postgres/      # Database access
├── app/service/workflow_service.go    # Business logic
└── features/                          # Feature modules
    ├── templates.go
    ├── sharing.go
    ├── debug.go
    └── replay.go
```

### 3. Execution Service (`cmd/services/execution/`)

**Purpose**: Workflow execution

**Responsibilities**:
- Execution lifecycle management
- Node execution tracking
- Execution history
- Real-time status updates

**Key Files**:
```
internal/execution/
├── domain/model/            # Execution entities
├── adapters/repository/     # Persistence
└── app/service/             # Execution logic

internal/engine/
├── engine.go                # Execution engine
├── executor.go              # Node executor
├── queue.go                 # Task queue
├── retry.go                 # Retry logic
└── persistence.go           # State persistence
```

### 4. Auth Service (`cmd/services/auth/`)

**Purpose**: Authentication and authorization

**Responsibilities**:
- User authentication (JWT)
- API key management
- OAuth2 flows
- Session management
- Password reset

**Key Files**:
```
internal/auth/
├── domain/model/auth.go              # Auth entities
├── adapters/http/handlers/           # Auth endpoints
├── adapters/repository/postgres/     # User storage
└── app/service/auth_service.go       # Auth logic
```

### 5. Integration Service (`cmd/services/integration/`)

**Purpose**: Third-party integrations

**Responsibilities**:
- OAuth flow handling
- Token management
- Connector registry
- Integration CRUD

**Key Files**:
```
internal/integration/
├── domain/model/              # Integration entities
├── connectors/                # Service connectors
│   └── connector.go           # Base connector
├── oauth/                     # OAuth handling
│   ├── oauth.go
│   └── handler.go
└── app/service/               # Integration logic
```

### 6. Schedule Service (`cmd/services/schedule/`)

**Purpose**: Scheduled workflow execution

**Responsibilities**:
- Cron schedule management
- Schedule evaluation
- Trigger execution

**Key Files**:
```
internal/schedule/
├── domain/model/schedule.go          # Schedule entity
├── app/service/scheduler.go          # Cron scheduler
└── adapters/repository/postgres/     # Schedule storage
```

### 7. Webhook Service (`cmd/services/webhook/`)

**Purpose**: Incoming webhook handling

**Responsibilities**:
- Webhook registration
- Request validation
- Workflow triggering

**Key Files**:
```
internal/webhook/
├── domain/model/webhook.go           # Webhook entity
├── adapters/http/handlers/           # Webhook endpoints
└── app/service/webhook_service.go    # Webhook logic
```

### 8. Notification Service (`cmd/services/notification/`)

**Purpose**: User notifications

**Responsibilities**:
- Email sending (SMTP, SendGrid)
- In-app notifications
- Notification preferences

**Key Files**:
```
internal/notification/
├── domain/model/                     # Notification entities
├── adapters/smtp/                    # SMTP provider
├── adapters/sendgrid/                # SendGrid provider
└── app/service/                      # Notification logic
```

### 9. Billing Service (`cmd/services/billing/`)

**Purpose**: Subscription and billing

**Responsibilities**:
- Stripe integration
- Subscription management
- Usage tracking
- Invoice generation

**Key Files**:
```
internal/billing/
├── domain/model/billing.go           # Billing entities
├── adapters/http/handlers/           # Billing endpoints
└── app/service/billing_service.go    # Billing logic
```

## Supporting Services

### Credential Service (`cmd/services/credential/`)
- Encrypted credential storage
- Credential types management
- OAuth token storage

### Workspace Service (`cmd/services/workspace/`)
- Multi-tenant workspace management
- Member management
- Role-based access

### Node Service (`cmd/services/node/`)
- Node type registry
- Node metadata
- Node execution

## Service Communication

### Synchronous (In-Process)

Services communicate through dependency injection:

```go
// Workflow service depends on execution service
type WorkflowService struct {
    repo             repository.WorkflowRepository
    executionService *execution.Service
}
```

### Asynchronous (Events)

Domain events for loose coupling:

```go
// Event bus for cross-service communication
eventBus.Publish(events.WorkflowExecuted{
    WorkflowID:  workflowID,
    ExecutionID: executionID,
})

// Subscriber in another service
eventBus.Subscribe("workflow.executed", func(event Event) {
    // Handle event
})
```

## Database Schema

Each service has its own tables:

| Service | Tables |
|---------|--------|
| Workflow | `workflows`, `workflow_nodes`, `workflow_connections` |
| Execution | `executions`, `node_executions` |
| Auth | `users`, `sessions`, `api_keys`, `oauth_tokens` |
| Credential | `credentials`, `credential_types` |
| Schedule | `schedules` |
| Webhook | `webhooks` |
| Billing | `plans`, `subscriptions`, `invoices`, `usage` |
| Workspace | `workspaces`, `workspace_members`, `invitations` |
| Notification | `notifications`, `email_templates` |

## Scaling Considerations

### Stateless Services
All services are stateless, enabling horizontal scaling:

```yaml
# Kubernetes deployment
replicas: 3
```

### Database Connection Pooling
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
```

### Worker Pool Scaling
```go
workerPool := executor.NewWorkerPool(
    executor.WithWorkerCount(10),
    executor.WithQueueSize(1000),
)
```

## Monitoring

Each service exposes:
- Health endpoint: `/health`
- Metrics endpoint: `/metrics` (Prometheus)
- Tracing: Jaeger integration

## Next Steps

- [Architecture Overview](overview.md)
- [Domain-Driven Design](domain-driven-design.md)
- [Deployment Guide](../deployment.md)
