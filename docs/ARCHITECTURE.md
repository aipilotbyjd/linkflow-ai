# LinkFlow Go - Production-Ready Microservices Architecture

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Service Decomposition](#service-decomposition)
3. [Project Structure](#project-structure)
4. [Communication Patterns](#communication-patterns)
5. [Data Architecture](#data-architecture)
6. [Security Architecture](#security-architecture)
7. [Observability Stack](#observability-stack)
8. [Deployment Strategy](#deployment-strategy)
9. [Developer Experience](#developer-experience)
10. [Performance Optimization](#performance-optimization)

## Architecture Overview

### Core Principles
- **Domain-Driven Design (DDD)**: Clear bounded contexts per service
- **Clean/Hexagonal Architecture**: Business logic independent of frameworks
- **Event-Driven Architecture**: Asynchronous communication via Kafka/NATS
- **CQRS Pattern**: Separate read/write models for complex domains
- **Event Sourcing**: Complete audit trail and time-travel debugging
- **API-First Design**: OpenAPI specifications before implementation
- **Security by Design**: Zero-trust architecture, defense in depth
- **Cloud-Native**: Kubernetes-native, 12-factor app principles
- **GitOps**: Declarative infrastructure and deployments

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│                         External Clients                         │
│           (Web App, Mobile App, CLI, API Consumers)             │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                          CDN Layer                               │
│                    (CloudFlare/Fastly)                          │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Load Balancer Layer                         │
│                    (AWS ALB/NLB, GCP GLB)                       │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                     (Kong/Envoy Gateway)                         │
│  • Rate Limiting  • Auth  • Routing  • Load Balancing           │
└─────────────────────────────────────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        ▼                        ▼                        ▼
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│   GraphQL    │        │   REST API   │        │  WebSocket   │
│   Gateway    │        │   Services   │        │   Gateway    │
└──────────────┘        └──────────────┘        └──────────────┘
        │                        │                        │
        └────────────────────────┼────────────────────────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
         ┌──────────────────────────────────────────┐
         │          Service Mesh (Istio)            │
         │   • mTLS  • Circuit Breaking  • Retry    │
         └──────────────────────────────────────────┘
                                 │
    ┌────────────────────────────┼────────────────────────────┐
    ▼                            ▼                            ▼
┌─────────────┐          ┌─────────────┐          ┌─────────────┐
│   Core      │          │   Business   │          │   Support   │
│  Services   │          │   Services   │          │  Services   │
├─────────────┤          ├─────────────┤          ├─────────────┤
│ • Auth      │          │ • Workflow   │          │ • Search    │
│ • User      │          │ • Execution  │          │ • Analytics │
│ • Billing   │          │ • Node       │          │ • Audit     │
│ • Tenant    │          │ • Schedule   │          │ • Storage   │
└─────────────┘          └─────────────┘          └─────────────┘
        │                        │                        │
        └────────────────────────┼────────────────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         ▼                       ▼                       ▼
┌──────────────┐        ┌──────────────┐        ┌──────────────┐
│   Message    │        │   Database   │        │    Cache     │
│     Bus      │        │    Layer     │        │    Layer     │
├──────────────┤        ├──────────────┤        ├──────────────┤
│ • Kafka      │        │ • PostgreSQL │        │ • Redis      │
│ • NATS       │        │ • MongoDB    │        │ • Memcached  │
│ • RabbitMQ   │        │ • Cassandra  │        │ • Hazelcast  │
└──────────────┘        └──────────────┘        └──────────────┘
```

## Service Decomposition

### Core Domain Services

#### 1. Authentication Service (Port 8001)
**Responsibilities:**
- JWT token generation/validation
- OAuth2/OIDC integration
- API key management
- Session management
- MFA/2FA support
- Password policies
- Account lockout/security

**Technology Stack:**
- JWT with RS256
- Redis for session storage
- Argon2id for password hashing
- TOTP for 2FA

#### 2. User Service (Port 8002)
**Responsibilities:**
- User profile management
- Team/organization management
- Role-based access control (RBAC)
- User preferences
- User activity tracking
- Invitation system

#### 3. Tenant Service (Port 8006)
**Responsibilities:**
- Multi-tenancy management
- Tenant isolation
- Feature flags per tenant
- Tenant-specific configurations
- Resource quotas
- Billing tier management

### Business Domain Services

#### 4. Workflow Service (Port 8004)
**Responsibilities:**
- Workflow CRUD operations
- Version control
- Template management
- Workflow sharing/collaboration
- Import/export
- Workflow validation
- DAG management

#### 5. Execution Service (Port 8003)
**Responsibilities:**
- Execution orchestration
- State management
- Execution history
- Retry logic
- Error handling
- Execution metrics
- Resource allocation

#### 6. Node Service (Port 8005)
**Responsibilities:**
- Node registry
- Custom node management
- Node marketplace
- Node versioning
- Node validation
- Node documentation
- Community nodes

#### 7. Executor Service (Port 8007)
**Responsibilities:**
- Actual node execution
- Sandboxed environments (V8 Isolates / WebAssembly / Firecracker)
- Resource limits (CPU, Memory, Network)
- Timeout management
- Parallel execution
- Node communication
- Data transformation

**Sandboxing Strategy:**
- **Code Nodes (JS/Python):** Executed in isolated runtime environments (e.g., V8 Isolates or gVisor-sandboxed containers) to prevent unauthorized access to host resources.
- **System Nodes:** Executed within the service context but with strict timeout and memory contexts.

#### 8. Webhook Service (Port 8008)
**Responsibilities:**
- Webhook registration
- Payload validation
- Request signing
- Retry mechanism
- Webhook testing
- Rate limiting
- Event filtering

#### 9. Schedule Service (Port 8009)
**Responsibilities:**
- Cron job management
- Timezone handling
- Schedule conflict detection
- Recurring executions
- Schedule history
- Holiday calendars
- Schedule optimization

#### 10. Credential Service (Port 8010)
**Responsibilities:**
- Secure credential storage
- Encryption/decryption
- OAuth token refresh
- Credential sharing
- Audit logging
- Key rotation
- HSM integration

### Support Services

#### 11. Notification Service (Port 8011)
**Responsibilities:**
- Email notifications
- SMS notifications
- Push notifications
- In-app notifications
- Notification templates
- User preferences
- Delivery tracking

#### 12. Audit Service (Port 8012)
**Responsibilities:**
- Activity logging
- Compliance reporting
- Data retention
- GDPR compliance
- Security events
- Access logs
- Change tracking

#### 13. Analytics Service (Port 8013)
**Responsibilities:**
- Usage analytics
- Performance metrics
- Business intelligence
- Custom dashboards
- Report generation
- Data aggregation
- Predictive analytics

#### 14. Search Service (Port 8014)
**Responsibilities:**
- Full-text search
- Faceted search
- Search indexing
- Query optimization
- Search analytics
- Autocomplete
- Fuzzy matching

#### 15. Storage Service (Port 8015)
**Responsibilities:**
- File storage
- Object storage
- CDN integration
- File versioning
- Access control
- Virus scanning
- Compression

#### 16. Billing Service (Port 8016)
**Responsibilities:**
- Subscription management
- Payment processing
- Invoice generation
- Usage tracking
- Metering
- Pricing plans
- Payment gateway integration

#### 17. Variable Service (Port 8017)
**Responsibilities:**
- Environment variables
- Secrets management
- Configuration management
- Variable scoping
- Variable encryption
- Variable versioning
- Variable validation

#### 18. WebSocket Service (Port 8018)
**Responsibilities:**
- Real-time updates
- Live collaboration
- Execution streaming
- Presence detection
- Connection management
- Message broadcasting
- Room management

## Project Structure

### Complete Repository Structure

```
linkflow-go/
├── cmd/                                 # Service entry points
│   ├── services/
│   │   ├── auth/                       # Each service follows same pattern
│   │   │   └── main.go
│   │   ├── workflow/
│   │   ├── execution/
│   │   └── [other-services]/
│   ├── tools/                          # CLI tools and utilities
│   │   ├── migrate/
│   │   ├── seed/
│   │   └── generate/
│   └── workers/                        # Background workers
│       ├── cleaner/
│       ├── scheduler/
│       └── mailer/
│
├── internal/                           # Private application code
│   ├── [service-name]/                # Per-service internal structure
│   │   ├── domain/                    # Domain layer (entities, value objects)
│   │   │   ├── model/
│   │   │   ├── repository/
│   │   │   └── service/
│   │   ├── app/                       # Application layer (use cases)
│   │   │   ├── command/
│   │   │   ├── query/
│   │   │   └── service/
│   │   ├── adapters/                  # Infrastructure layer
│   │   │   ├── grpc/
│   │   │   ├── http/
│   │   │   ├── repository/
│   │   │   ├── messaging/
│   │   │   └── cache/
│   │   ├── ports/                     # Interfaces (inbound/outbound)
│   │   │   ├── inbound/
│   │   │   └── outbound/
│   │   └── server/                    # Server setup
│   │       ├── http.go
│   │       ├── grpc.go
│   │       └── routes.go
│   │
│   ├── platform/                       # Shared platform code
│   │   ├── auth/                      # Authentication middleware
│   │   ├── database/                  # Database utilities
│   │   ├── cache/                     # Caching layer
│   │   ├── messaging/                 # Event bus
│   │   ├── tracing/                   # Distributed tracing
│   │   ├── metrics/                   # Prometheus metrics
│   │   └── logger/                    # Structured logging
│   │
│   └── shared/                        # Shared business logic
│       ├── dto/                       # Data transfer objects
│       ├── events/                    # Event definitions
│       ├── errors/                    # Error types
│       └── utils/                     # Utility functions
│
├── pkg/                               # Public packages
│   ├── api/                          # API client libraries
│   │   ├── rest/
│   │   ├── grpc/
│   │   └── graphql/
│   ├── sdk/                          # SDK for external consumers
│   ├── middleware/                   # Reusable middleware
│   └── validators/                   # Input validators
│
├── api/                               # API definitions
│   ├── openapi/                      # OpenAPI specs
│   │   ├── auth.yaml
│   │   ├── workflow.yaml
│   │   └── [service].yaml
│   ├── grpc/                         # Proto files
│   │   ├── auth.proto
│   │   └── [service].proto
│   └── graphql/                      # GraphQL schemas
│       ├── schema.graphql
│       └── resolvers/
│
├── deployments/                       # Deployment configurations
│   ├── docker/                       # Dockerfiles
│   │   ├── Dockerfile
│   │   ├── Dockerfile.dev
│   │   └── Dockerfile.worker
│   ├── kubernetes/                   # K8s manifests
│   │   ├── base/
│   │   ├── overlays/
│   │   │   ├── dev/
│   │   │   ├── staging/
│   │   │   └── production/
│   │   └── kustomization.yaml
│   ├── helm/                         # Helm charts
│   │   ├── linkflow/
│   │   │   ├── charts/
│   │   │   ├── templates/
│   │   │   ├── values.yaml
│   │   │   └── Chart.yaml
│   │   └── dependencies/
│   ├── terraform/                    # Infrastructure as Code
│   │   ├── modules/
│   │   ├── environments/
│   │   └── backend.tf
│   └── istio/                        # Service mesh configs
│       ├── gateway.yaml
│       ├── virtual-services.yaml
│       └── destination-rules.yaml
│
├── migrations/                        # Database migrations
│   ├── auth/
│   │   └── 000001_initial_schema.sql
│   ├── workflow/
│   └── [service]/
│
├── configs/                          # Configuration files
│   ├── envs/                        # Environment configs
│   │   ├── .env.example
│   │   ├── .env.development
│   │   └── .env.test
│   ├── kong/                        # API Gateway configs
│   ├── prometheus/                  # Metrics configs
│   └── grafana/                     # Dashboard configs
│
├── scripts/                          # Build and utility scripts
│   ├── build/
│   ├── test/
│   ├── deploy/
│   └── hooks/                       # Git hooks
│
├── tests/                            # Test suites
│   ├── unit/                        # Unit tests
│   ├── integration/                 # Integration tests
│   ├── e2e/                         # End-to-end tests
│   ├── load/                        # Load tests
│   ├── security/                    # Security tests
│   └── fixtures/                    # Test data
│
├── docs/                             # Documentation
│   ├── architecture/                # Architecture decisions
│   ├── api/                         # API documentation
│   ├── guides/                      # User guides
│   └── adr/                         # Architecture Decision Records
│
├── tools/                            # Development tools
│   ├── generators/                  # Code generators
│   ├── analyzers/                   # Static analyzers
│   └── linters/                     # Custom linters
│
├── .github/                          # GitHub specific
│   ├── workflows/                   # GitHub Actions
│   ├── ISSUE_TEMPLATE/
│   └── PULL_REQUEST_TEMPLATE.md
│
├── docker-compose.yml                # Local development
├── docker-compose.override.yml       # Local overrides
├── Makefile                         # Build commands
├── go.mod                           # Go modules
├── go.work                          # Go workspace
└── README.md                        # Project documentation
```

### Service Internal Structure (Clean Architecture)

Each service follows this structure:

```
internal/[service-name]/
├── domain/                          # Core business logic
│   ├── model/
│   │   ├── workflow.go             # Entities
│   │   ├── node.go                 # Value objects
│   │   └── events.go               # Domain events
│   ├── repository/
│   │   └── interfaces.go           # Repository interfaces
│   └── service/
│       └── workflow_service.go     # Domain services
│
├── app/                            # Application layer
│   ├── command/                    # Command handlers (CQRS)
│   │   ├── create_workflow.go
│   │   └── update_workflow.go
│   ├── query/                      # Query handlers (CQRS)
│   │   ├── get_workflow.go
│   │   └── list_workflows.go
│   └── service/
│       └── application_service.go  # Use case orchestration
│
├── adapters/                       # Infrastructure layer
│   ├── http/
│   │   ├── handlers/              # HTTP handlers
│   │   ├── middleware/            # HTTP middleware
│   │   └── dto/                   # Request/Response DTOs
│   ├── grpc/
│   │   ├── server/               # gRPC server
│   │   └── client/               # gRPC clients
│   ├── repository/
│   │   ├── postgres/             # PostgreSQL implementation
│   │   ├── mongodb/              # MongoDB implementation
│   │   └── cache/                # Cache implementation
│   └── messaging/
│       ├── publisher/            # Event publishers
│       └── subscriber/           # Event subscribers
│
├── ports/                         # Interfaces/contracts
│   ├── inbound/
│   │   ├── http.go               # HTTP port interface
│   │   └── grpc.go               # gRPC port interface
│   └── outbound/
│       ├── repository.go         # Repository port
│       ├── cache.go              # Cache port
│       └── messaging.go          # Messaging port
│
└── server/                        # Server bootstrapping
    ├── server.go                  # Main server setup
    ├── routes.go                  # Route registration
    ├── dependencies.go            # Dependency injection
    └── config.go                  # Configuration loading
```

## Communication Patterns

### Synchronous Communication

#### REST API
```yaml
Pattern: Request-Response
Use Cases:
  - CRUD operations
  - Real-time queries
  - User-facing operations
Implementation:
  - OpenAPI 3.0 specification
  - JSON API standard
  - HAL for hypermedia
  - Content negotiation
```

#### gRPC
```yaml
Pattern: RPC
Use Cases:
  - Service-to-service communication
  - High-performance requirements
  - Streaming data
Implementation:
  - Protocol Buffers v3
  - Bidirectional streaming
  - Deadline propagation
  - Load balancing
```

### Asynchronous Communication

#### Event-Driven Architecture
```yaml
Message Broker: Apache Kafka
Patterns:
  - Publish-Subscribe
  - Event Sourcing
  - CQRS
  - Saga Pattern
Event Format:
  - CloudEvents specification
  - Schema Registry
  - Avro/Protobuf serialization
```

#### Event Types
```go
// Base event structure
type Event struct {
    ID              string                 `json:"id"`
    Source          string                 `json:"source"`
    Type            string                 `json:"type"`
    Subject         string                 `json:"subject"`
    Time            time.Time              `json:"time"`
    DataContentType string                 `json:"datacontenttype"`
    Data            json.RawMessage        `json:"data"`
    Metadata        map[string]interface{} `json:"metadata"`
    CorrelationID   string                 `json:"correlationid"`
    CausationID     string                 `json:"causationid"`
}

// Event naming convention: {domain}.{entity}.{action}.{version}
// Examples:
// - workflow.workflow.created.v1
// - execution.execution.started.v1
// - auth.user.logged_in.v1
```

### API Gateway Pattern

```yaml
Kong Configuration:
  Plugins:
    - Rate Limiting:
        - Free tier: 100 req/hour
        - Basic tier: 1000 req/hour
        - Pro tier: 10000 req/hour
        - Enterprise: Unlimited
    - Authentication:
        - JWT validation (RS256)
        - API Key
        - OAuth 2.0
        - mTLS
    - Security:
        - CORS
        - IP Restriction
        - Bot Detection
        - Request Size Limiting
    - Transformation:
        - Request/Response transformation
        - GraphQL proxy
        - gRPC-Web proxy
    - Observability:
        - Prometheus metrics
        - Request logging
        - Distributed tracing
```

## Data Architecture

### Database Strategy

#### Primary Database: PostgreSQL
```yaml
Configuration:
  - Version: 15+
  - Connection Pooling: PgBouncer
  - Read Replicas: 2 minimum
  - Partitioning: By tenant_id and created_at
  - Extensions:
    - UUID (uuid-ossp)
    - JSONB operations
    - Full-text search
    - TimescaleDB for time-series
```

#### Database Per Service Pattern
```sql
-- Each service has its own schema
CREATE SCHEMA auth_service;
CREATE SCHEMA workflow_service;
CREATE SCHEMA execution_service;

-- Shared read models for CQRS
CREATE SCHEMA read_models;

-- Event store for event sourcing
CREATE SCHEMA event_store;
```

### Data Flow Strategy (Large Payloads)

#### Claim Check Pattern
To handle large workflow data (e.g., JSON arrays > 1MB, binary files) efficiently:
1. **Payload > Threshold (e.g., 1MB):**
   - Store payload in Object Storage (MinIO/S3).
   - Generate a reference ID (Claim Check).
   - Pass the reference ID in the Event/Message.
2. **Payload < Threshold:**
   - Pass payload directly in the Event/Message.

This ensures the Message Bus (Kafka) and Database (Postgres) remain performant and are not clogged with blob data.

### CQRS Implementation

#### Write Model (Commands)
```go
type WorkflowCommandHandler struct {
    repo      WorkflowRepository
    eventBus  EventPublisher
    validator Validator
}

func (h *WorkflowCommandHandler) CreateWorkflow(cmd CreateWorkflowCommand) error {
    // Validate command
    if err := h.validator.Validate(cmd); err != nil {
        return err
    }
    
    // Create aggregate
    workflow := domain.NewWorkflow(cmd)
    
    // Save to write store
    if err := h.repo.Save(workflow); err != nil {
        return err
    }
    
    // Publish domain events
    for _, event := range workflow.GetUncommittedEvents() {
        h.eventBus.Publish(event)
    }
    
    return nil
}
```

#### Read Model (Queries)
```go
type WorkflowQueryHandler struct {
    readDB ReadDatabase
    cache  CacheService
}

func (h *WorkflowQueryHandler) GetWorkflowDetails(query GetWorkflowQuery) (*WorkflowDTO, error) {
    // Check cache first
    if cached, found := h.cache.Get(query.ID); found {
        return cached.(*WorkflowDTO), nil
    }
    
    // Query optimized read model
    dto, err := h.readDB.GetWorkflow(query.ID)
    if err != nil {
        return nil, err
    }
    
    // Update cache
    h.cache.Set(query.ID, dto, 5*time.Minute)
    
    return dto, nil
}
```

### Event Sourcing

```go
type EventStore interface {
    SaveEvents(aggregateID string, events []Event, expectedVersion int) error
    GetEvents(aggregateID string, fromVersion int) ([]Event, error)
    GetSnapshot(aggregateID string) (*Snapshot, error)
    SaveSnapshot(aggregateID string, snapshot Snapshot) error
}

type AggregateRoot struct {
    ID               string
    Version          int
    uncommittedEvents []Event
}

func (a *AggregateRoot) ApplyEvent(event Event) {
    // Apply event to aggregate state
    a.Version++
}

func (a *AggregateRoot) GetUncommittedEvents() []Event {
    return a.uncommittedEvents
}
```

### Caching Strategy

```yaml
Redis Cluster:
  - Cache-aside pattern for read-heavy data
  - Write-through for critical data
  - TTL Strategy:
    - User sessions: 1 hour
    - Workflow definitions: 5 minutes
    - Execution state: 30 seconds
    - Static content: 24 hours
  - Eviction Policy: allkeys-lru
  - Persistence: AOF with fsync every second
```

## Security Architecture

### Zero-Trust Security Model

#### Authentication & Authorization
```yaml
Authentication:
  - JWT tokens (RS256 signing)
  - Token rotation every hour
  - Refresh token with rotation
  - MFA/2FA required for admin
  - Device fingerprinting
  - Session management

Authorization:
  - RBAC with fine-grained permissions
  - Attribute-based access control (ABAC)
  - Policy Decision Point (PDP)
  - Policy Enforcement Point (PEP)
  - Resource-level permissions
```

#### API Security
```yaml
Rate Limiting:
  - Per-user limits
  - Per-IP limits
  - Adaptive rate limiting
  - Cost-based throttling

Input Validation:
  - Request schema validation
  - SQL injection prevention
  - XSS protection
  - CSRF tokens
  - Content-Type validation

Encryption:
  - TLS 1.3 minimum
  - mTLS for service-to-service
  - Field-level encryption
  - At-rest encryption (AES-256-GCM)
  - Key rotation every 90 days
```

#### Secrets Management
```yaml
HashiCorp Vault:
  - Dynamic secrets
  - Encryption as a service
  - PKI certificates
  - SSH key management
  - Database credentials rotation
  - Audit logging
```

## Observability Stack

### Metrics (Prometheus + Grafana)

```yaml
Key Metrics:
  Business Metrics:
    - workflow_executions_total
    - workflow_success_rate
    - execution_duration_seconds
    - active_users_count
    - api_usage_by_endpoint
    
  Technical Metrics:
    - http_requests_total
    - http_request_duration_seconds
    - grpc_requests_total
    - database_connections_active
    - cache_hit_ratio
    - message_queue_lag
    
  SLIs (Service Level Indicators):
    - Availability (uptime percentage)
    - Latency (p50, p95, p99)
    - Error rate
    - Throughput (requests per second)
```

### Logging (ELK Stack)

```go
// Structured logging with context
type Logger interface {
    WithFields(fields map[string]interface{}) Logger
    Info(msg string)
    Warn(msg string)
    Error(msg string, err error)
    Debug(msg string)
}

// Usage example
logger.WithFields(map[string]interface{}{
    "service": "workflow",
    "user_id": userID,
    "workflow_id": workflowID,
    "correlation_id": correlationID,
    "trace_id": traceID,
}).Info("Workflow execution started")
```

### Distributed Tracing (Jaeger)

```go
// OpenTelemetry integration
import "go.opentelemetry.io/otel"

func ExecuteWorkflow(ctx context.Context, workflowID string) error {
    tracer := otel.Tracer("workflow-service")
    ctx, span := tracer.Start(ctx, "ExecuteWorkflow")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("workflow.id", workflowID),
        attribute.String("user.id", getUserID(ctx)),
    )
    
    // Business logic with nested spans
    if err := validateWorkflow(ctx, workflowID); err != nil {
        span.RecordError(err)
        return err
    }
    
    return nil
}
```

### Health Checks

```go
// Kubernetes probes
type HealthChecker interface {
    // Liveness: Is the service running?
    CheckLiveness(ctx context.Context) error
    
    // Readiness: Is the service ready to accept traffic?
    CheckReadiness(ctx context.Context) error
    
    // Startup: Has the service finished initializing?
    CheckStartup(ctx context.Context) error
}

// Implementation
func (s *Server) CheckReadiness(ctx context.Context) error {
    checks := []func() error{
        s.checkDatabase,
        s.checkRedis,
        s.checkKafka,
        s.checkDependentServices,
    }
    
    for _, check := range checks {
        if err := check(); err != nil {
            return fmt.Errorf("readiness check failed: %w", err)
        }
    }
    
    return nil
}
```

## Deployment Strategy

### Container Strategy

```dockerfile
# Multi-stage build for optimization
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service cmd/services/${SERVICE_NAME}/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs
EXPOSE 8080
CMD ["./service"]
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workflow-service
  labels:
    app: workflow-service
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: workflow-service
  template:
    metadata:
      labels:
        app: workflow-service
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
    spec:
      containers:
      - name: workflow-service
        image: linkflow/workflow-service:v1.0.0
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /app/configs
      volumes:
      - name: config
        configMap:
          name: workflow-service-config
```

### CI/CD Pipeline (GitLab CI)

```yaml
stages:
  - build
  - test
  - security
  - deploy

variables:
  DOCKER_DRIVER: overlay2
  KUBERNETES_VERSION: 1.28

build:
  stage: build
  script:
    - docker build -t $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA .
    - docker push $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA

unit-tests:
  stage: test
  script:
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out

integration-tests:
  stage: test
  services:
    - postgres:15
    - redis:7
  script:
    - go test -v -tags=integration ./tests/integration/...

security-scan:
  stage: security
  script:
    - trivy image $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - gosec ./...
    - nancy sleuth

deploy-staging:
  stage: deploy
  environment: staging
  script:
    - kubectl set image deployment/workflow-service workflow-service=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - kubectl rollout status deployment/workflow-service

deploy-production:
  stage: deploy
  environment: production
  when: manual
  script:
    - kubectl set image deployment/workflow-service workflow-service=$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA
    - kubectl rollout status deployment/workflow-service
```

## Developer Experience

### Local Development Setup

```makefile
# Makefile for developer productivity
.PHONY: help dev test build clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

dev: ## Start development environment
	docker-compose up -d
	air -c .air.toml

test: ## Run all tests
	go test -v -race ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run linters
	golangci-lint run --fix
	staticcheck ./...

build: ## Build all services
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		go build -o bin/$$service cmd/services/$$service/main.go; \
	done

generate: ## Generate code
	go generate ./...
	protoc --go_out=. --go-grpc_out=. api/grpc/*.proto
	oapi-codegen -generate types,server,spec api/openapi/*.yaml

migrate: ## Run database migrations
	migrate -path migrations -database "postgresql://..." up

seed: ## Seed development data
	go run cmd/tools/seed/main.go

clean: ## Clean build artifacts
	rm -rf bin/ dist/ coverage.* vendor/
```

### Development Tools

```yaml
Code Generation:
  - protoc for gRPC
  - oapi-codegen for OpenAPI
  - sqlboiler for ORM
  - mockgen for mocks
  - wire for dependency injection

Code Quality:
  - golangci-lint (comprehensive linting)
  - staticcheck (static analysis)
  - gosec (security scanning)
  - go-critic (code review)
  - gofumpt (stricter formatting)

Testing:
  - testify (assertions)
  - gomock (mocking)
  - ginkgo (BDD testing)
  - vegeta (load testing)
  - testcontainers (integration testing)

Development:
  - air (hot reload)
  - delve (debugging)
  - cobra (CLI)
  - viper (configuration)
  - zerolog (structured logging)
```

## Performance Optimization

### Database Optimization

```sql
-- Indexes for common queries
CREATE INDEX idx_workflows_user_id ON workflows(user_id);
CREATE INDEX idx_workflows_status ON workflows(status);
CREATE INDEX idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX idx_executions_started_at ON executions(started_at);

-- Composite indexes for complex queries
CREATE INDEX idx_workflows_user_status ON workflows(user_id, status);

-- Partial indexes for filtered queries
CREATE INDEX idx_active_workflows ON workflows(id) WHERE status = 'active';

-- JSON indexes for JSONB columns
CREATE INDEX idx_workflow_metadata ON workflows USING GIN (metadata);
```

### Caching Strategies

```go
// Multi-level caching
type CacheService struct {
    l1Cache *ristretto.Cache  // In-memory L1 cache
    l2Cache *redis.Client     // Redis L2 cache
}

func (c *CacheService) Get(key string) (interface{}, bool) {
    // Check L1 cache first
    if val, found := c.l1Cache.Get(key); found {
        return val, true
    }
    
    // Check L2 cache
    val, err := c.l2Cache.Get(key).Result()
    if err == nil {
        // Populate L1 cache
        c.l1Cache.Set(key, val, 1)
        return val, true
    }
    
    return nil, false
}
```

### Connection Pooling

```go
// Database connection pool
func NewDBPool() *sql.DB {
    db, _ := sql.Open("postgres", dsn)
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    db.SetConnMaxIdleTime(10 * time.Minute)
    return db
}

// Redis connection pool
func NewRedisPool() *redis.Client {
    return redis.NewClient(&redis.Options{
        Addr:         "localhost:6379",
        PoolSize:     10,
        MinIdleConns: 5,
        MaxRetries:   3,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolTimeout:  4 * time.Second,
    })
}
```

### Load Balancing

```yaml
Service Mesh (Istio):
  Load Balancing Algorithms:
    - Round Robin (default)
    - Least Request
    - Random
    - Consistent Hash
  
  Circuit Breaking:
    - Max connections: 100
    - Max pending requests: 10
    - Max requests per connection: 1
    - Consecutive errors: 5
    - Interval: 30s
    - Base ejection time: 30s
  
  Retry Policy:
    - Attempts: 3
    - Per try timeout: 30s
    - Retry on: 5xx, reset, connect-failure
    - Backoff: exponential
```

## Testing Strategy

### Testing Pyramid

```
         /\           E2E Tests (5%)
        /  \          - Full user flows
       /    \         - Cross-service scenarios
      /      \
     /________\       Integration Tests (20%)
    /          \      - API tests
   /            \     - Database tests
  /              \    - Message queue tests
 /                \
/__________________\  Unit Tests (75%)
                      - Business logic
                      - Domain models
                      - Utilities
```

### Test Coverage Requirements

```yaml
Coverage Targets:
  - Unit Tests: 90% coverage
  - Integration Tests: 70% coverage
  - Critical Path: 100% coverage
  - Overall: 85% coverage

Test Execution:
  - Pre-commit: Unit tests
  - Pre-merge: Unit + Integration
  - Nightly: Full suite including E2E
  - Release: Full suite + load tests
```

## Monitoring & Alerting

### SLOs and Error Budgets

```yaml
Service Level Objectives:
  Availability:
    - Target: 99.9% (43.2 minutes downtime/month)
    - Measurement: Successful requests / Total requests
  
  Latency:
    - p50: < 100ms
    - p95: < 500ms
    - p99: < 1000ms
  
  Error Rate:
    - Target: < 0.1%
    - Measurement: 5xx responses / Total responses

Error Budget Policy:
  - Budget exhausted: Freeze feature work
  - Budget < 25%: Focus on reliability
  - Budget < 50%: Review incidents
  - Budget > 50%: Normal operations
```

### Alert Rules

```yaml
Critical Alerts (Page immediately):
  - Service down > 2 minutes
  - Error rate > 5% for 5 minutes
  - Database connection pool > 90%
  - Disk space < 10%
  - Memory usage > 95%

Warning Alerts (Notify team):
  - Error rate > 1% for 10 minutes
  - p95 latency > SLO for 15 minutes
  - CPU usage > 80% for 20 minutes
  - Kafka lag > 1000 messages
  - Certificate expiry < 30 days

Info Alerts (Log only):
  - Deployment completed
  - Backup completed
  - Scaling event
  - Configuration change
```

## Disaster Recovery

### Backup Strategy

```yaml
Database Backups:
  - Full backup: Daily at 2 AM
  - Incremental: Every 4 hours
  - Point-in-time recovery: Enabled
  - Retention: 30 days
  - Cross-region replication: Enabled
  - Test restore: Weekly

Application Data:
  - Code: Git repository (GitHub/GitLab)
  - Configurations: Encrypted in git
  - Secrets: HashiCorp Vault with backup
  - Files: S3 with versioning
  - Logs: 90-day retention
```

### High Availability

```yaml
Multi-Region Deployment:
  Primary Region: us-east-1
  Secondary Region: eu-west-1
  
  Database:
    - Multi-master replication
    - Automatic failover
    - Read replicas per region
  
  Services:
    - Active-active deployment
    - Geographic load balancing
    - Regional data residency
  
  Message Queue:
    - Kafka mirror maker
    - Cross-region replication
    - Regional topics
```

## Conclusion

This architecture provides:

1. **Scalability**: Horizontal scaling of services, database sharding, caching layers
2. **Reliability**: Circuit breakers, retries, health checks, graceful degradation
3. **Security**: Zero-trust, encryption, secrets management, audit logging
4. **Observability**: Metrics, logs, traces, alerts, dashboards
5. **Developer Experience**: Clear structure, automation, tooling, documentation
6. **Performance**: Optimized queries, caching, connection pooling, CDN
7. **Maintainability**: Clean architecture, DDD, SOLID principles, testing

The architecture is designed to handle millions of workflows and billions of executions while maintaining sub-second response times and 99.9% availability.

## Roadmap to Production

To realize this architecture from the current state, the following phases are recommended:

1. **Foundation (Current)**:
   - Core services (Auth, User, Workflow, Execution, Node) implemented.
   - Basic REST communication and Database persistence established.

2. **Interconnectivity & Resilience**:
   - Implement gRPC for synchronous inter-service communication.
   - Fully integrate Kafka for asynchronous event-driven flows.
   - Implement the "Executor Service" with robust sandboxing.

3. **Scalability & Security**:
   - Deploy API Gateway (Kong/Envoy).
   - Implement Circuit Breakers and Rate Limiting.
   - Introduce Object Storage for large payload handling (Claim Check pattern).

4. **Production Readiness**:
   - Achieve >85% Test Coverage (Unit, Integration, E2E).
   - Deploy Monitoring & Observability stack (Prometheus, Grafana, Jaeger).
   - Finalize Kubernetes manifests and CI/CD pipelines.
