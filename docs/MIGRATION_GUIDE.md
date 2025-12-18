# LinkFlow Go - Migration Guide

## From Monolith to Microservices - Step-by-Step Migration

### Phase 1: Foundation (Weeks 1-2)

#### 1.1 Infrastructure Setup
```bash
# Set up local development environment
make setup-infra

# This will provision:
# - PostgreSQL with proper schemas
# - Redis cluster
# - Kafka with topics
# - Elasticsearch
# - Monitoring stack (Prometheus, Grafana, Jaeger)
```

#### 1.2 Database Migration
```sql
-- Create schemas for each service
CREATE SCHEMA IF NOT EXISTS auth_service;
CREATE SCHEMA IF NOT EXISTS workflow_service;
CREATE SCHEMA IF NOT EXISTS execution_service;
CREATE SCHEMA IF NOT EXISTS event_store;
CREATE SCHEMA IF NOT EXISTS read_models;

-- Migrate existing tables to appropriate schemas
ALTER TABLE users SET SCHEMA auth_service;
ALTER TABLE workflows SET SCHEMA workflow_service;
ALTER TABLE executions SET SCHEMA execution_service;
```

#### 1.3 Event Store Setup
```sql
-- Create event store table
CREATE TABLE event_store.domain_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id VARCHAR(255) NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_version INTEGER NOT NULL,
    event_data JSONB NOT NULL,
    metadata JSONB,
    user_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_aggregate (aggregate_id, event_version),
    INDEX idx_event_type (event_type),
    INDEX idx_created_at (created_at)
);

-- Create snapshots table for event sourcing
CREATE TABLE event_store.snapshots (
    aggregate_id VARCHAR(255) PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,
    version INTEGER NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Phase 2: Extract Auth Service (Week 3)

#### 2.1 Create Auth Service Structure
```bash
# Generate service scaffold
go run cmd/tools/generate/service.go --name=auth --port=8001

# This creates:
# - cmd/services/auth/main.go
# - internal/auth/... (complete structure)
# - migrations/auth/...
# - api/openapi/auth.yaml
```

#### 2.2 Migrate Authentication Logic
```go
// Move from monolith
// OLD: internal/services/auth.go
// NEW: internal/auth/domain/service/auth_service.go

// Extract interfaces
type AuthService interface {
    Login(ctx context.Context, email, password string) (*Token, error)
    Refresh(ctx context.Context, refreshToken string) (*Token, error)
    Logout(ctx context.Context, token string) error
    Validate(ctx context.Context, token string) (*Claims, error)
}
```

#### 2.3 Setup Event Publishing
```go
// Publish auth events
func (s *AuthService) Login(ctx context.Context, email, password string) (*Token, error) {
    // ... authentication logic ...
    
    // Publish event
    event := events.UserLoggedIn{
        UserID:    user.ID,
        Email:     email,
        IP:        getIPFromContext(ctx),
        UserAgent: getUserAgentFromContext(ctx),
        Timestamp: time.Now(),
    }
    
    s.eventBus.Publish(ctx, events.Wrap("auth.user.logged_in.v1", event))
    
    return token, nil
}
```

### Phase 3: Extract Workflow Service (Week 4)

#### 3.1 Implement Domain Model
```go
// internal/workflow/domain/model/workflow.go
type Workflow struct {
    id          WorkflowID
    version     int
    userID      string
    name        string
    description string
    nodes       []Node
    connections []Connection
    status      WorkflowStatus
    events      []DomainEvent
}

// Apply domain-driven design patterns
func (w *Workflow) Activate() error {
    if !w.canActivate() {
        return ErrInvalidStateTransition
    }
    
    w.status = WorkflowStatusActive
    w.addEvent(WorkflowActivated{
        WorkflowID: w.id,
        Timestamp:  time.Now(),
    })
    
    return nil
}
```

#### 3.2 Implement CQRS
```go
// Command side
type CreateWorkflowCommand struct {
    UserID      string
    Name        string
    Description string
    Nodes       []NodeDTO
}

// Query side
type GetWorkflowQuery struct {
    WorkflowID string
    UserID     string
}

// Separate handlers
type CommandHandler struct {
    writeDB  WriteDatabase
    eventBus EventPublisher
}

type QueryHandler struct {
    readDB ReadDatabase
    cache  CacheService
}
```

### Phase 4: Setup Service Mesh (Week 5)

#### 4.1 Install Istio
```bash
# Install Istio
istioctl install --set profile=demo -y

# Enable sidecar injection
kubectl label namespace default istio-injection=enabled
```

#### 4.2 Configure Traffic Management
```yaml
# deployments/istio/virtual-service.yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: workflow-service
spec:
  hosts:
  - workflow-service
  http:
  - match:
    - headers:
        x-version:
          exact: v2
    route:
    - destination:
        host: workflow-service
        subset: v2
      weight: 100
  - route:
    - destination:
        host: workflow-service
        subset: v1
      weight: 90
    - destination:
        host: workflow-service
        subset: v2
      weight: 10  # Canary deployment
```

### Phase 5: Implement Saga Pattern (Week 6)

#### 5.1 Define Saga Steps
```go
// internal/platform/saga/workflow_execution_saga.go
type WorkflowExecutionSaga struct {
    steps []SagaStep
}

func NewWorkflowExecutionSaga() *WorkflowExecutionSaga {
    return &WorkflowExecutionSaga{
        steps: []SagaStep{
            {
                Name:       "ReserveResources",
                Action:     reserveResources,
                Compensate: releaseResources,
            },
            {
                Name:       "InitializeExecution",
                Action:     initializeExecution,
                Compensate: markExecutionFailed,
            },
            {
                Name:       "ExecuteNodes",
                Action:     executeNodes,
                Compensate: rollbackNodeChanges,
            },
            {
                Name:       "SaveResults",
                Action:     saveResults,
                Compensate: deleteResults,
            },
        },
    }
}
```

### Phase 6: Performance Optimization (Week 7)

#### 6.1 Implement Caching Strategy
```go
// Cache warming on startup
func (s *WorkflowService) WarmCache(ctx context.Context) error {
    // Load frequently accessed workflows
    workflows, err := s.repo.FindMostUsed(ctx, 100)
    if err != nil {
        return err
    }
    
    for _, wf := range workflows {
        key := fmt.Sprintf("workflow:%s", wf.ID)
        s.cache.Set(ctx, key, wf, 10*time.Minute)
    }
    
    return nil
}

// Multi-level caching
func (s *WorkflowService) GetWorkflow(ctx context.Context, id string) (*Workflow, error) {
    // L1: Local memory cache
    if wf, ok := s.localCache.Get(id); ok {
        return wf.(*Workflow), nil
    }
    
    // L2: Redis cache
    var workflow Workflow
    if err := s.redisCache.Get(ctx, id, &workflow); err == nil {
        s.localCache.Set(id, &workflow, 1*time.Minute)
        return &workflow, nil
    }
    
    // L3: Database
    workflow, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Update caches
    s.redisCache.Set(ctx, id, workflow, 5*time.Minute)
    s.localCache.Set(id, workflow, 1*time.Minute)
    
    return workflow, nil
}
```

### Phase 7: Monitoring & Observability (Week 8)

#### 7.1 Setup Metrics Collection
```go
// internal/platform/metrics/metrics.go
var (
    WorkflowsCreated = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "linkflow_workflows_created_total",
            Help: "Total number of workflows created",
        },
        []string{"user_id", "status"},
    )
    
    ExecutionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "linkflow_execution_duration_seconds",
            Help:    "Workflow execution duration in seconds",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300},
        },
        []string{"workflow_id", "status"},
    )
)

// Usage
func (s *ExecutionService) Execute(ctx context.Context, workflowID string) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        ExecutionDuration.WithLabelValues(workflowID, "success").Observe(duration)
    }()
    
    // ... execution logic ...
}
```

#### 7.2 Setup Distributed Tracing
```go
// internal/platform/tracing/tracing.go
func InitTracer(serviceName string) (*sdktrace.TracerProvider, error) {
    exporter, err := jaeger.New(
        jaeger.WithCollectorEndpoint(
            jaeger.WithEndpoint("http://jaeger:14268/api/traces"),
        ),
    )
    if err != nil {
        return nil, err
    }
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
        )),
        sdktrace.WithSampler(sdktrace.AlwaysSample()),
    )
    
    otel.SetTracerProvider(tp)
    otel.SetTextMapPropagator(propagation.TraceContext{})
    
    return tp, nil
}
```

### Phase 8: Testing Migration (Week 9)

#### 8.1 Contract Testing
```go
// tests/contract/workflow_api_test.go
func TestWorkflowAPIContract(t *testing.T) {
    // Use Pact for contract testing
    pact := &dsl.Pact{
        Consumer: "workflow-ui",
        Provider: "workflow-service",
    }
    
    pact.AddInteraction().
        Given("A workflow exists").
        UponReceiving("A request for workflow details").
        WithRequest(dsl.Request{
            Method: "GET",
            Path:   dsl.String("/api/v1/workflows/123"),
        }).
        WillRespondWith(dsl.Response{
            Status: 200,
            Body: dsl.Match(&WorkflowResponse{}),
        })
    
    err := pact.Verify(func() error {
        // Test implementation
        resp, err := client.GetWorkflow("123")
        assert.NoError(t, err)
        assert.NotNil(t, resp)
        return nil
    })
    
    assert.NoError(t, err)
}
```

#### 8.2 Load Testing
```go
// tests/load/workflow_load_test.go
func TestWorkflowServiceLoad(t *testing.T) {
    rate := vegeta.Rate{Freq: 100, Per: time.Second}
    duration := 5 * time.Minute
    
    targeter := vegeta.NewStaticTargeter(vegeta.Target{
        Method: "POST",
        URL:    "http://localhost:8003/api/v1/workflows",
        Body:   []byte(`{"name":"Load Test","description":"Test"}`),
        Header: http.Header{
            "Content-Type":  []string{"application/json"},
            "Authorization": []string{"Bearer " + token},
        },
    })
    
    attacker := vegeta.NewAttacker()
    
    var metrics vegeta.Metrics
    for res := range attacker.Attack(targeter, rate, duration, "Load Test") {
        metrics.Add(res)
    }
    metrics.Close()
    
    assert.Less(t, metrics.Latencies.P95, 200*time.Millisecond)
    assert.Less(t, metrics.Errors, float64(0.01)) // Less than 1% errors
}
```

### Phase 9: Deployment (Week 10)

#### 9.1 Kubernetes Deployment
```bash
# Deploy to staging
kubectl apply -k deployments/kubernetes/overlays/staging/

# Run smoke tests
go test -tags=smoke ./tests/smoke/...

# Deploy to production with canary
kubectl apply -f deployments/kubernetes/canary-deployment.yaml

# Monitor metrics
kubectl port-forward svc/grafana 3000:3000
# Open http://localhost:3000

# Full rollout after verification
kubectl apply -k deployments/kubernetes/overlays/production/
```

#### 9.2 Database Migration
```bash
# Run migrations for each service
migrate -path migrations/auth -database $AUTH_DB_URL up
migrate -path migrations/workflow -database $WORKFLOW_DB_URL up
migrate -path migrations/execution -database $EXECUTION_DB_URL up

# Verify migrations
psql $DATABASE_URL -c "SELECT * FROM schema_migrations;"
```

### Phase 10: Cutover Strategy

#### 10.1 Gradual Migration
```nginx
# Use nginx for gradual traffic shifting
upstream monolith {
    server old-monolith:8080 weight=90;
}

upstream microservices {
    server kong-gateway:8000 weight=10;
}

server {
    location /api/ {
        proxy_pass http://monolith;
        
        # Gradually increase microservices weight
        # Week 1: 10%
        # Week 2: 25%
        # Week 3: 50%
        # Week 4: 75%
        # Week 5: 100%
    }
}
```

#### 10.2 Rollback Plan
```bash
# Quick rollback script
#!/bin/bash
if [ "$1" = "rollback" ]; then
    echo "Rolling back to monolith..."
    kubectl scale deployment/auth-service --replicas=0
    kubectl scale deployment/workflow-service --replicas=0
    kubectl scale deployment/monolith --replicas=5
    
    # Update routing
    kubectl apply -f deployments/kubernetes/rollback-routing.yaml
    
    echo "Rollback complete"
fi
```

## Migration Checklist

- [ ] **Infrastructure**
  - [ ] PostgreSQL with schemas
  - [ ] Redis cluster
  - [ ] Kafka/NATS
  - [ ] Elasticsearch
  - [ ] Monitoring stack

- [ ] **Services Extracted**
  - [ ] Auth Service
  - [ ] User Service
  - [ ] Workflow Service
  - [ ] Execution Service
  - [ ] Node Service
  - [ ] Webhook Service
  - [ ] Schedule Service
  - [ ] Notification Service

- [ ] **Patterns Implemented**
  - [ ] Event-driven architecture
  - [ ] CQRS
  - [ ] Event sourcing
  - [ ] Saga pattern
  - [ ] Circuit breakers
  - [ ] Service mesh

- [ ] **Quality Assurance**
  - [ ] Unit tests (>90% coverage)
  - [ ] Integration tests
  - [ ] Contract tests
  - [ ] Load tests
  - [ ] Security tests

- [ ] **Operations**
  - [ ] CI/CD pipelines
  - [ ] Monitoring dashboards
  - [ ] Alerting rules
  - [ ] Runbooks
  - [ ] Disaster recovery plan

- [ ] **Documentation**
  - [ ] API documentation
  - [ ] Architecture diagrams
  - [ ] Developer guides
  - [ ] Operations manual
  - [ ] Troubleshooting guides

## Success Metrics

### Technical Metrics
- Response time p95 < 200ms
- Error rate < 0.1%
- Availability > 99.9%
- Deployment frequency > 10x per day
- Mean time to recovery < 15 minutes

### Business Metrics
- Customer satisfaction score > 4.5/5
- Developer productivity increased by 50%
- Time to market reduced by 40%
- Infrastructure costs optimized by 30%
- System scalability improved 10x

## Support & Resources

- **Documentation**: `/docs/`
- **Slack Channel**: #linkflow-migration
- **Office Hours**: Tuesdays & Thursdays 2-3 PM
- **Emergency Contact**: oncall@linkflow.com
- **Training Videos**: https://learn.linkflow.com/microservices
