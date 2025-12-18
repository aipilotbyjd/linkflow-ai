# LinkFlow Go - Implementation Guide

## Table of Contents
1. [Service Implementation Pattern](#service-implementation-pattern)
2. [Domain-Driven Design Implementation](#domain-driven-design-implementation)
3. [Event-Driven Architecture](#event-driven-architecture)
4. [API Implementation](#api-implementation)
5. [Database Layer](#database-layer)
6. [Testing Implementation](#testing-implementation)
7. [Security Implementation](#security-implementation)
8. [Performance Patterns](#performance-patterns)

## Service Implementation Pattern

### Base Service Structure

```go
// cmd/services/workflow/main.go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/linkflow/internal/workflow/server"
    "github.com/linkflow/internal/platform/config"
    "github.com/linkflow/internal/platform/logger"
    "github.com/linkflow/internal/platform/telemetry"
)

func main() {
    // Initialize configuration
    cfg, err := config.Load()
    if err != nil {
        panic(fmt.Sprintf("failed to load config: %v", err))
    }
    
    // Initialize logger
    log := logger.New(cfg.Logger)
    
    // Initialize telemetry
    tel, err := telemetry.New(cfg.Telemetry)
    if err != nil {
        log.Fatal("failed to initialize telemetry", "error", err)
    }
    defer tel.Close()
    
    // Create server
    srv, err := server.New(
        server.WithConfig(cfg),
        server.WithLogger(log),
        server.WithTelemetry(tel),
    )
    if err != nil {
        log.Fatal("failed to create server", "error", err)
    }
    
    // Start server
    errCh := make(chan error, 1)
    go func() {
        if err := srv.Start(); err != nil {
            errCh <- err
        }
    }()
    
    // Wait for shutdown signal
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    select {
    case err := <-errCh:
        log.Error("server error", "error", err)
    case sig := <-sigCh:
        log.Info("received shutdown signal", "signal", sig)
    }
    
    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Error("shutdown error", "error", err)
    }
}
```

### Server Implementation

```go
// internal/workflow/server/server.go
package server

import (
    "context"
    "fmt"
    "net"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/linkflow/internal/workflow/adapters/http/handlers"
    "github.com/linkflow/internal/workflow/adapters/repository/postgres"
    "github.com/linkflow/internal/workflow/adapters/messaging/kafka"
    "github.com/linkflow/internal/workflow/app/service"
    "github.com/linkflow/internal/workflow/domain"
    "github.com/linkflow/internal/platform/database"
    "github.com/linkflow/internal/platform/middleware"
    grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
    "google.golang.org/grpc"
)

type Server struct {
    config     *Config
    logger     Logger
    telemetry  Telemetry
    httpServer *http.Server
    grpcServer *grpc.Server
    
    // Dependencies
    db          *database.DB
    cache       CacheService
    eventBus    EventBus
    
    // Services
    workflowService *service.WorkflowService
}

type Option func(*Server)

func New(opts ...Option) (*Server, error) {
    s := &Server{}
    
    for _, opt := range opts {
        opt(s)
    }
    
    if err := s.initialize(); err != nil {
        return nil, fmt.Errorf("failed to initialize server: %w", err)
    }
    
    return s, nil
}

func (s *Server) initialize() error {
    // Initialize database
    db, err := database.New(s.config.Database)
    if err != nil {
        return fmt.Errorf("failed to initialize database: %w", err)
    }
    s.db = db
    
    // Initialize cache
    cache, err := NewRedisCache(s.config.Redis)
    if err != nil {
        return fmt.Errorf("failed to initialize cache: %w", err)
    }
    s.cache = cache
    
    // Initialize event bus
    eventBus, err := kafka.NewEventBus(s.config.Kafka)
    if err != nil {
        return fmt.Errorf("failed to initialize event bus: %w", err)
    }
    s.eventBus = eventBus
    
    // Initialize repositories
    workflowRepo := postgres.NewWorkflowRepository(db)
    
    // Initialize domain services
    domainService := domain.NewWorkflowDomainService(workflowRepo)
    
    // Initialize application service
    s.workflowService = service.NewWorkflowService(
        domainService,
        workflowRepo,
        s.eventBus,
        s.cache,
    )
    
    // Setup HTTP server
    s.setupHTTPServer()
    
    // Setup gRPC server
    s.setupGRPCServer()
    
    return nil
}

func (s *Server) setupHTTPServer() {
    router := mux.NewRouter()
    
    // Add middleware
    router.Use(
        middleware.RequestID(),
        middleware.Logger(s.logger),
        middleware.Metrics(s.telemetry),
        middleware.Tracing(s.telemetry),
        middleware.Recovery(),
        middleware.CORS(),
        middleware.RateLimiter(),
    )
    
    // Health checks
    router.HandleFunc("/health/live", s.handleLiveness).Methods("GET")
    router.HandleFunc("/health/ready", s.handleReadiness).Methods("GET")
    
    // Metrics
    router.Handle("/metrics", s.telemetry.MetricsHandler()).Methods("GET")
    
    // API routes
    apiRouter := router.PathPrefix("/api/v1").Subrouter()
    
    // Initialize handlers
    workflowHandler := handlers.NewWorkflowHandler(s.workflowService, s.logger)
    
    // Register routes
    workflowHandler.RegisterRoutes(apiRouter)
    
    s.httpServer = &http.Server{
        Addr:         fmt.Sprintf(":%d", s.config.HTTP.Port),
        Handler:      router,
        ReadTimeout:  s.config.HTTP.ReadTimeout,
        WriteTimeout: s.config.HTTP.WriteTimeout,
        IdleTimeout:  s.config.HTTP.IdleTimeout,
    }
}

func (s *Server) setupGRPCServer() {
    opts := []grpc.ServerOption{
        grpc.UnaryInterceptor(grpcMiddleware.ChainUnaryServer(
            s.telemetry.UnaryServerInterceptor(),
            s.logger.UnaryServerInterceptor(),
            s.recoveryInterceptor(),
        )),
        grpc.StreamInterceptor(grpcMiddleware.ChainStreamServer(
            s.telemetry.StreamServerInterceptor(),
            s.logger.StreamServerInterceptor(),
        )),
    }
    
    s.grpcServer = grpc.NewServer(opts...)
    
    // Register gRPC services
    // pb.RegisterWorkflowServiceServer(s.grpcServer, grpcHandler)
}

func (s *Server) Start() error {
    // Start event bus consumers
    go s.startEventConsumers()
    
    // Start HTTP server
    go func() {
        s.logger.Info("starting HTTP server", "port", s.config.HTTP.Port)
        if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            s.logger.Error("HTTP server error", "error", err)
        }
    }()
    
    // Start gRPC server
    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
    if err != nil {
        return fmt.Errorf("failed to listen on gRPC port: %w", err)
    }
    
    s.logger.Info("starting gRPC server", "port", s.config.GRPC.Port)
    return s.grpcServer.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("shutting down server")
    
    // Shutdown HTTP server
    if err := s.httpServer.Shutdown(ctx); err != nil {
        s.logger.Error("HTTP server shutdown error", "error", err)
    }
    
    // Shutdown gRPC server
    s.grpcServer.GracefulStop()
    
    // Close connections
    s.db.Close()
    s.cache.Close()
    s.eventBus.Close()
    
    return nil
}
```

## Domain-Driven Design Implementation

### Domain Models

```go
// internal/workflow/domain/model/workflow.go
package model

import (
    "errors"
    "time"
    
    "github.com/google/uuid"
)

// Value Objects
type WorkflowID string

func NewWorkflowID() WorkflowID {
    return WorkflowID(uuid.New().String())
}

func (id WorkflowID) String() string {
    return string(id)
}

func (id WorkflowID) Validate() error {
    if id == "" {
        return errors.New("workflow ID cannot be empty")
    }
    _, err := uuid.Parse(string(id))
    return err
}

type WorkflowStatus string

const (
    WorkflowStatusDraft    WorkflowStatus = "draft"
    WorkflowStatusActive   WorkflowStatus = "active"
    WorkflowStatusInactive WorkflowStatus = "inactive"
    WorkflowStatusArchived WorkflowStatus = "archived"
)

// Entity
type Workflow struct {
    // Aggregate root
    id          WorkflowID
    version     int
    events      []DomainEvent
    
    // Properties
    userID      string
    name        string
    description string
    status      WorkflowStatus
    nodes       []Node
    connections []Connection
    settings    Settings
    createdAt   time.Time
    updatedAt   time.Time
    
    // Business invariants
    maxNodes int
}

// Factory method
func NewWorkflow(userID, name, description string) (*Workflow, error) {
    if userID == "" {
        return nil, errors.New("user ID is required")
    }
    if name == "" {
        return nil, errors.New("workflow name is required")
    }
    
    w := &Workflow{
        id:          NewWorkflowID(),
        version:     0,
        userID:      userID,
        name:        name,
        description: description,
        status:      WorkflowStatusDraft,
        nodes:       make([]Node, 0),
        connections: make([]Connection, 0),
        settings:    DefaultSettings(),
        createdAt:   time.Now(),
        updatedAt:   time.Now(),
        maxNodes:    100, // Business rule
    }
    
    // Raise domain event
    w.addEvent(WorkflowCreatedEvent{
        WorkflowID:  w.id,
        UserID:      userID,
        Name:        name,
        Description: description,
        CreatedAt:   w.createdAt,
    })
    
    return w, nil
}

// Domain logic
func (w *Workflow) Activate() error {
    if w.status != WorkflowStatusDraft && w.status != WorkflowStatusInactive {
        return errors.New("workflow can only be activated from draft or inactive status")
    }
    
    if len(w.nodes) == 0 {
        return errors.New("workflow must have at least one node")
    }
    
    if err := w.validateConnections(); err != nil {
        return fmt.Errorf("invalid connections: %w", err)
    }
    
    w.status = WorkflowStatusActive
    w.updatedAt = time.Now()
    
    w.addEvent(WorkflowActivatedEvent{
        WorkflowID:  w.id,
        ActivatedAt: w.updatedAt,
    })
    
    return nil
}

func (w *Workflow) AddNode(node Node) error {
    if len(w.nodes) >= w.maxNodes {
        return fmt.Errorf("workflow cannot have more than %d nodes", w.maxNodes)
    }
    
    if w.status == WorkflowStatusArchived {
        return errors.New("cannot modify archived workflow")
    }
    
    // Check for duplicate node ID
    for _, existing := range w.nodes {
        if existing.ID == node.ID {
            return errors.New("node with this ID already exists")
        }
    }
    
    w.nodes = append(w.nodes, node)
    w.updatedAt = time.Now()
    
    w.addEvent(NodeAddedEvent{
        WorkflowID: w.id,
        Node:       node,
        AddedAt:    w.updatedAt,
    })
    
    return nil
}

func (w *Workflow) validateConnections() error {
    nodeMap := make(map[string]bool)
    for _, node := range w.nodes {
        nodeMap[node.ID] = true
    }
    
    for _, conn := range w.connections {
        if !nodeMap[conn.SourceNodeID] {
            return fmt.Errorf("source node %s not found", conn.SourceNodeID)
        }
        if !nodeMap[conn.TargetNodeID] {
            return fmt.Errorf("target node %s not found", conn.TargetNodeID)
        }
    }
    
    // Check for cycles
    if w.hasCycle() {
        return errors.New("workflow contains a cycle")
    }
    
    return nil
}

// Event handling
func (w *Workflow) addEvent(event DomainEvent) {
    w.events = append(w.events, event)
    w.version++
}

func (w *Workflow) GetUncommittedEvents() []DomainEvent {
    return w.events
}

func (w *Workflow) MarkEventsAsCommitted() {
    w.events = []DomainEvent{}
}

// Repository interface
type WorkflowRepository interface {
    Save(ctx context.Context, workflow *Workflow) error
    FindByID(ctx context.Context, id WorkflowID) (*Workflow, error)
    FindByUserID(ctx context.Context, userID string, offset, limit int) ([]*Workflow, error)
    Update(ctx context.Context, workflow *Workflow) error
    Delete(ctx context.Context, id WorkflowID) error
}
```

### Domain Services

```go
// internal/workflow/domain/service/workflow_service.go
package service

import (
    "context"
    "fmt"
    
    "github.com/linkflow/internal/workflow/domain/model"
)

type WorkflowDomainService struct {
    repo model.WorkflowRepository
}

func NewWorkflowDomainService(repo model.WorkflowRepository) *WorkflowDomainService {
    return &WorkflowDomainService{
        repo: repo,
    }
}

// Complex business logic that involves multiple aggregates
func (s *WorkflowDomainService) DuplicateWorkflow(ctx context.Context, sourceID model.WorkflowID, newName string) (*model.Workflow, error) {
    // Load source workflow
    source, err := s.repo.FindByID(ctx, sourceID)
    if err != nil {
        return nil, fmt.Errorf("failed to find source workflow: %w", err)
    }
    
    // Create new workflow with copied properties
    duplicate, err := model.NewWorkflow(source.UserID(), newName, source.Description())
    if err != nil {
        return nil, fmt.Errorf("failed to create duplicate workflow: %w", err)
    }
    
    // Copy nodes
    for _, node := range source.Nodes() {
        nodeCopy := node.Clone()
        if err := duplicate.AddNode(nodeCopy); err != nil {
            return nil, fmt.Errorf("failed to add node: %w", err)
        }
    }
    
    // Copy connections
    for _, conn := range source.Connections() {
        if err := duplicate.AddConnection(conn.Clone()); err != nil {
            return nil, fmt.Errorf("failed to add connection: %w", err)
        }
    }
    
    // Save duplicate
    if err := s.repo.Save(ctx, duplicate); err != nil {
        return nil, fmt.Errorf("failed to save duplicate workflow: %w", err)
    }
    
    return duplicate, nil
}

func (s *WorkflowDomainService) ValidateWorkflowExecutability(ctx context.Context, id model.WorkflowID) error {
    workflow, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return fmt.Errorf("failed to find workflow: %w", err)
    }
    
    // Check if workflow is active
    if workflow.Status() != model.WorkflowStatusActive {
        return errors.New("workflow must be active to execute")
    }
    
    // Validate has trigger node
    hasTrigger := false
    for _, node := range workflow.Nodes() {
        if node.Type == model.NodeTypeTrigger {
            hasTrigger = true
            break
        }
    }
    
    if !hasTrigger {
        return errors.New("workflow must have at least one trigger node")
    }
    
    // Validate all required node parameters are set
    for _, node := range workflow.Nodes() {
        if err := node.ValidateParameters(); err != nil {
            return fmt.Errorf("node %s has invalid parameters: %w", node.ID, err)
        }
    }
    
    return nil
}
```

## Event-Driven Architecture

### Event Definitions

```go
// internal/shared/events/workflow_events.go
package events

import (
    "encoding/json"
    "time"
)

// Base event
type Event struct {
    ID            string                 `json:"id"`
    AggregateID   string                 `json:"aggregateId"`
    AggregateType string                 `json:"aggregateType"`
    EventType     string                 `json:"eventType"`
    EventVersion  int                    `json:"eventVersion"`
    Timestamp     time.Time              `json:"timestamp"`
    UserID        string                 `json:"userId"`
    CorrelationID string                 `json:"correlationId"`
    Metadata      map[string]interface{} `json:"metadata"`
    Payload       json.RawMessage        `json:"payload"`
}

// Workflow events
type WorkflowCreated struct {
    WorkflowID  string    `json:"workflowId"`
    UserID      string    `json:"userId"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"createdAt"`
}

type WorkflowActivated struct {
    WorkflowID  string    `json:"workflowId"`
    ActivatedBy string    `json:"activatedBy"`
    ActivatedAt time.Time `json:"activatedAt"`
}

type WorkflowExecutionStarted struct {
    ExecutionID string                 `json:"executionId"`
    WorkflowID  string                 `json:"workflowId"`
    TriggerType string                 `json:"triggerType"`
    InputData   map[string]interface{} `json:"inputData"`
    StartedAt   time.Time              `json:"startedAt"`
}

type WorkflowExecutionCompleted struct {
    ExecutionID string                 `json:"executionId"`
    WorkflowID  string                 `json:"workflowId"`
    Status      string                 `json:"status"`
    OutputData  map[string]interface{} `json:"outputData"`
    Duration    time.Duration          `json:"duration"`
    CompletedAt time.Time              `json:"completedAt"`
}
```

### Event Publisher

```go
// internal/platform/messaging/publisher.go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/Shopify/sarama"
    "github.com/google/uuid"
    "github.com/linkflow/internal/shared/events"
)

type EventPublisher interface {
    Publish(ctx context.Context, event events.Event) error
    PublishBatch(ctx context.Context, events []events.Event) error
}

type KafkaEventPublisher struct {
    producer sarama.AsyncProducer
    config   *Config
}

func NewKafkaEventPublisher(config *Config) (*KafkaEventPublisher, error) {
    saramaConfig := sarama.NewConfig()
    saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
    saramaConfig.Producer.Retry.Max = 5
    saramaConfig.Producer.Return.Successes = true
    saramaConfig.Producer.Return.Errors = true
    saramaConfig.Producer.Compression = sarama.CompressionSnappy
    
    producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create producer: %w", err)
    }
    
    publisher := &KafkaEventPublisher{
        producer: producer,
        config:   config,
    }
    
    // Handle producer errors
    go publisher.handleErrors()
    
    return publisher, nil
}

func (p *KafkaEventPublisher) Publish(ctx context.Context, event events.Event) error {
    // Set event metadata
    if event.ID == "" {
        event.ID = uuid.New().String()
    }
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }
    
    // Extract correlation ID from context
    if correlationID := ctx.Value("correlationID"); correlationID != nil {
        event.CorrelationID = correlationID.(string)
    }
    
    // Serialize event
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to serialize event: %w", err)
    }
    
    // Determine topic based on event type
    topic := p.getTopicForEvent(event.EventType)
    
    // Create Kafka message
    message := &sarama.ProducerMessage{
        Topic: topic,
        Key:   sarama.StringEncoder(event.AggregateID),
        Value: sarama.ByteEncoder(data),
        Headers: []sarama.RecordHeader{
            {
                Key:   []byte("eventType"),
                Value: []byte(event.EventType),
            },
            {
                Key:   []byte("correlationId"),
                Value: []byte(event.CorrelationID),
            },
        },
    }
    
    // Send message
    select {
    case p.producer.Input() <- message:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (p *KafkaEventPublisher) handleErrors() {
    for err := range p.producer.Errors() {
        // Log error and potentially send to DLQ
        fmt.Printf("Producer error: %v\n", err.Err)
    }
}

func (p *KafkaEventPublisher) getTopicForEvent(eventType string) string {
    // Map event types to topics
    switch eventType {
    case "WorkflowCreated", "WorkflowUpdated", "WorkflowDeleted":
        return "workflow-events"
    case "WorkflowExecutionStarted", "WorkflowExecutionCompleted":
        return "execution-events"
    default:
        return "default-events"
    }
}
```

### Event Consumer

```go
// internal/platform/messaging/consumer.go
package messaging

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/Shopify/sarama"
    "github.com/linkflow/internal/shared/events"
)

type EventHandler func(ctx context.Context, event events.Event) error

type EventConsumer interface {
    Subscribe(eventType string, handler EventHandler) error
    Start(ctx context.Context) error
    Stop() error
}

type KafkaEventConsumer struct {
    consumerGroup sarama.ConsumerGroup
    handlers      map[string][]EventHandler
    topics        []string
    config        *Config
}

func NewKafkaEventConsumer(config *Config) (*KafkaEventConsumer, error) {
    saramaConfig := sarama.NewConfig()
    saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
    saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
    saramaConfig.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
    
    consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroup, saramaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create consumer group: %w", err)
    }
    
    return &KafkaEventConsumer{
        consumerGroup: consumerGroup,
        handlers:      make(map[string][]EventHandler),
        topics:        config.Topics,
        config:        config,
    }, nil
}

func (c *KafkaEventConsumer) Subscribe(eventType string, handler EventHandler) error {
    c.handlers[eventType] = append(c.handlers[eventType], handler)
    return nil
}

func (c *KafkaEventConsumer) Start(ctx context.Context) error {
    handler := &consumerGroupHandler{
        handlers: c.handlers,
    }
    
    for {
        // Check if context is cancelled
        if ctx.Err() != nil {
            return ctx.Err()
        }
        
        // Consume messages
        err := c.consumerGroup.Consume(ctx, c.topics, handler)
        if err != nil {
            return fmt.Errorf("consumer error: %w", err)
        }
    }
}

type consumerGroupHandler struct {
    handlers map[string][]EventHandler
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for message := range claim.Messages() {
        // Parse event
        var event events.Event
        if err := json.Unmarshal(message.Value, &event); err != nil {
            fmt.Printf("Failed to parse event: %v\n", err)
            session.MarkMessage(message, "")
            continue
        }
        
        // Create context with metadata
        ctx := context.Background()
        ctx = context.WithValue(ctx, "correlationID", event.CorrelationID)
        
        // Execute handlers for this event type
        if handlers, ok := h.handlers[event.EventType]; ok {
            for _, handler := range handlers {
                if err := handler(ctx, event); err != nil {
                    fmt.Printf("Handler error for event %s: %v\n", event.ID, err)
                    // Implement retry logic or send to DLQ
                }
            }
        }
        
        // Mark message as processed
        session.MarkMessage(message, "")
    }
    
    return nil
}
```

## API Implementation

### REST API Handler

```go
// internal/workflow/adapters/http/handlers/workflow_handler.go
package handlers

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "github.com/linkflow/internal/workflow/app/service"
    "github.com/linkflow/internal/workflow/adapters/http/dto"
    "github.com/linkflow/internal/platform/logger"
    "github.com/linkflow/internal/platform/response"
)

type WorkflowHandler struct {
    service *service.WorkflowService
    logger  logger.Logger
}

func NewWorkflowHandler(service *service.WorkflowService, logger logger.Logger) *WorkflowHandler {
    return &WorkflowHandler{
        service: service,
        logger:  logger,
    }
}

func (h *WorkflowHandler) RegisterRoutes(router *mux.Router) {
    router.HandleFunc("/workflows", h.CreateWorkflow).Methods("POST")
    router.HandleFunc("/workflows", h.ListWorkflows).Methods("GET")
    router.HandleFunc("/workflows/{id}", h.GetWorkflow).Methods("GET")
    router.HandleFunc("/workflows/{id}", h.UpdateWorkflow).Methods("PUT")
    router.HandleFunc("/workflows/{id}", h.DeleteWorkflow).Methods("DELETE")
    router.HandleFunc("/workflows/{id}/activate", h.ActivateWorkflow).Methods("POST")
    router.HandleFunc("/workflows/{id}/execute", h.ExecuteWorkflow).Methods("POST")
    router.HandleFunc("/workflows/{id}/duplicate", h.DuplicateWorkflow).Methods("POST")
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Parse request
    var req dto.CreateWorkflowRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response.Error(w, response.ErrBadRequest.WithDetails(err.Error()))
        return
    }
    
    // Validate request
    if err := req.Validate(); err != nil {
        response.Error(w, response.ErrValidation.WithDetails(err.Error()))
        return
    }
    
    // Get user from context
    userID := ctx.Value("userID").(string)
    
    // Create workflow
    workflow, err := h.service.CreateWorkflow(ctx, service.CreateWorkflowCommand{
        UserID:      userID,
        Name:        req.Name,
        Description: req.Description,
        Nodes:       req.Nodes,
        Connections: req.Connections,
    })
    if err != nil {
        h.logger.Error("Failed to create workflow", "error", err, "user_id", userID)
        response.Error(w, response.ErrInternal.WithDetails("Failed to create workflow"))
        return
    }
    
    // Convert to DTO
    resp := dto.WorkflowResponse{
        ID:          workflow.ID,
        Name:        workflow.Name,
        Description: workflow.Description,
        Status:      workflow.Status,
        Nodes:       workflow.Nodes,
        Connections: workflow.Connections,
        CreatedAt:   workflow.CreatedAt,
        UpdatedAt:   workflow.UpdatedAt,
    }
    
    response.JSON(w, http.StatusCreated, resp)
}

func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Parse query parameters
    params := dto.ListWorkflowsParams{
        Offset: r.URL.Query().Get("offset"),
        Limit:  r.URL.Query().Get("limit"),
        Status: r.URL.Query().Get("status"),
        Sort:   r.URL.Query().Get("sort"),
    }
    
    // Get user from context
    userID := ctx.Value("userID").(string)
    
    // List workflows
    workflows, total, err := h.service.ListWorkflows(ctx, service.ListWorkflowsQuery{
        UserID: userID,
        Offset: params.GetOffset(),
        Limit:  params.GetLimit(),
        Status: params.Status,
        Sort:   params.Sort,
    })
    if err != nil {
        h.logger.Error("Failed to list workflows", "error", err, "user_id", userID)
        response.Error(w, response.ErrInternal.WithDetails("Failed to list workflows"))
        return
    }
    
    // Convert to DTOs
    items := make([]dto.WorkflowResponse, len(workflows))
    for i, wf := range workflows {
        items[i] = dto.WorkflowResponse{
            ID:          wf.ID,
            Name:        wf.Name,
            Description: wf.Description,
            Status:      wf.Status,
            CreatedAt:   wf.CreatedAt,
            UpdatedAt:   wf.UpdatedAt,
        }
    }
    
    resp := dto.ListWorkflowsResponse{
        Items: items,
        Total: total,
        Pagination: dto.Pagination{
            Offset: params.GetOffset(),
            Limit:  params.GetLimit(),
            Total:  total,
        },
    }
    
    response.JSON(w, http.StatusOK, resp)
}
```

### GraphQL Implementation

```go
// internal/platform/graphql/resolver/workflow_resolver.go
package resolver

import (
    "context"
    
    "github.com/linkflow/internal/workflow/app/service"
    "github.com/linkflow/internal/platform/graphql/model"
)

type WorkflowResolver struct {
    workflowService *service.WorkflowService
}

func NewWorkflowResolver(workflowService *service.WorkflowService) *WorkflowResolver {
    return &WorkflowResolver{
        workflowService: workflowService,
    }
}

// Queries
func (r *WorkflowResolver) Workflow(ctx context.Context, id string) (*model.Workflow, error) {
    workflow, err := r.workflowService.GetWorkflow(ctx, id)
    if err != nil {
        return nil, err
    }
    
    return &model.Workflow{
        ID:          workflow.ID,
        Name:        workflow.Name,
        Description: workflow.Description,
        Status:      model.WorkflowStatus(workflow.Status),
        CreatedAt:   workflow.CreatedAt,
        UpdatedAt:   workflow.UpdatedAt,
    }, nil
}

func (r *WorkflowResolver) Workflows(ctx context.Context, filter *model.WorkflowFilter) ([]*model.Workflow, error) {
    query := service.ListWorkflowsQuery{}
    
    if filter != nil {
        if filter.Status != nil {
            query.Status = string(*filter.Status)
        }
        if filter.Name != nil {
            query.Name = *filter.Name
        }
    }
    
    workflows, _, err := r.workflowService.ListWorkflows(ctx, query)
    if err != nil {
        return nil, err
    }
    
    result := make([]*model.Workflow, len(workflows))
    for i, wf := range workflows {
        result[i] = &model.Workflow{
            ID:          wf.ID,
            Name:        wf.Name,
            Description: wf.Description,
            Status:      model.WorkflowStatus(wf.Status),
            CreatedAt:   wf.CreatedAt,
            UpdatedAt:   wf.UpdatedAt,
        }
    }
    
    return result, nil
}

// Mutations
func (r *WorkflowResolver) CreateWorkflow(ctx context.Context, input model.CreateWorkflowInput) (*model.Workflow, error) {
    workflow, err := r.workflowService.CreateWorkflow(ctx, service.CreateWorkflowCommand{
        Name:        input.Name,
        Description: input.Description,
    })
    if err != nil {
        return nil, err
    }
    
    return &model.Workflow{
        ID:          workflow.ID,
        Name:        workflow.Name,
        Description: workflow.Description,
        Status:      model.WorkflowStatus(workflow.Status),
        CreatedAt:   workflow.CreatedAt,
        UpdatedAt:   workflow.UpdatedAt,
    }, nil
}

// Subscriptions
func (r *WorkflowResolver) WorkflowExecutionUpdates(ctx context.Context, workflowID string) (<-chan *model.ExecutionUpdate, error) {
    updates := make(chan *model.ExecutionUpdate)
    
    // Subscribe to execution events
    go func() {
        defer close(updates)
        
        subscription := r.workflowService.SubscribeToExecutionUpdates(ctx, workflowID)
        for update := range subscription {
            select {
            case updates <- &model.ExecutionUpdate{
                ExecutionID: update.ExecutionID,
                Status:      model.ExecutionStatus(update.Status),
                Progress:    update.Progress,
                UpdatedAt:   update.UpdatedAt,
            }:
            case <-ctx.Done():
                return
            }
        }
    }()
    
    return updates, nil
}
```

## Database Layer

### Repository Implementation

```go
// internal/workflow/adapters/repository/postgres/workflow_repository.go
package postgres

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    
    "github.com/lib/pq"
    "github.com/linkflow/internal/workflow/domain/model"
)

type WorkflowRepository struct {
    db *sql.DB
}

func NewWorkflowRepository(db *sql.DB) *WorkflowRepository {
    return &WorkflowRepository{db: db}
}

func (r *WorkflowRepository) Save(ctx context.Context, workflow *model.Workflow) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Serialize nodes and connections
    nodesJSON, err := json.Marshal(workflow.Nodes())
    if err != nil {
        return fmt.Errorf("failed to serialize nodes: %w", err)
    }
    
    connectionsJSON, err := json.Marshal(workflow.Connections())
    if err != nil {
        return fmt.Errorf("failed to serialize connections: %w", err)
    }
    
    settingsJSON, err := json.Marshal(workflow.Settings())
    if err != nil {
        return fmt.Errorf("failed to serialize settings: %w", err)
    }
    
    // Insert workflow
    query := `
        INSERT INTO workflows (
            id, user_id, name, description, status, 
            nodes, connections, settings, version,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
    `
    
    _, err = tx.ExecContext(ctx, query,
        workflow.ID().String(),
        workflow.UserID(),
        workflow.Name(),
        workflow.Description(),
        workflow.Status(),
        nodesJSON,
        connectionsJSON,
        settingsJSON,
        workflow.Version(),
        workflow.CreatedAt(),
        workflow.UpdatedAt(),
    )
    if err != nil {
        return fmt.Errorf("failed to insert workflow: %w", err)
    }
    
    // Save events
    if err := r.saveEvents(ctx, tx, workflow); err != nil {
        return fmt.Errorf("failed to save events: %w", err)
    }
    
    return tx.Commit()
}

func (r *WorkflowRepository) FindByID(ctx context.Context, id model.WorkflowID) (*model.Workflow, error) {
    query := `
        SELECT 
            id, user_id, name, description, status,
            nodes, connections, settings, version,
            created_at, updated_at
        FROM workflows
        WHERE id = $1
    `
    
    var (
        workflowID      string
        userID          string
        name            string
        description     string
        status          string
        nodesJSON       []byte
        connectionsJSON []byte
        settingsJSON    []byte
        version         int
        createdAt       time.Time
        updatedAt       time.Time
    )
    
    err := r.db.QueryRowContext(ctx, query, id.String()).Scan(
        &workflowID,
        &userID,
        &name,
        &description,
        &status,
        &nodesJSON,
        &connectionsJSON,
        &settingsJSON,
        &version,
        &createdAt,
        &updatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, model.ErrWorkflowNotFound
        }
        return nil, fmt.Errorf("failed to query workflow: %w", err)
    }
    
    // Deserialize JSON fields
    var nodes []model.Node
    if err := json.Unmarshal(nodesJSON, &nodes); err != nil {
        return nil, fmt.Errorf("failed to deserialize nodes: %w", err)
    }
    
    var connections []model.Connection
    if err := json.Unmarshal(connectionsJSON, &connections); err != nil {
        return nil, fmt.Errorf("failed to deserialize connections: %w", err)
    }
    
    var settings model.Settings
    if err := json.Unmarshal(settingsJSON, &settings); err != nil {
        return nil, fmt.Errorf("failed to deserialize settings: %w", err)
    }
    
    // Reconstruct workflow
    workflow := model.ReconstructWorkflow(
        model.WorkflowID(workflowID),
        userID,
        name,
        description,
        model.WorkflowStatus(status),
        nodes,
        connections,
        settings,
        version,
        createdAt,
        updatedAt,
    )
    
    return workflow, nil
}

func (r *WorkflowRepository) Update(ctx context.Context, workflow *model.Workflow) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Serialize JSON fields
    nodesJSON, _ := json.Marshal(workflow.Nodes())
    connectionsJSON, _ := json.Marshal(workflow.Connections())
    settingsJSON, _ := json.Marshal(workflow.Settings())
    
    // Update with optimistic locking
    query := `
        UPDATE workflows
        SET 
            name = $2,
            description = $3,
            status = $4,
            nodes = $5,
            connections = $6,
            settings = $7,
            version = $8,
            updated_at = $9
        WHERE id = $1 AND version = $10
    `
    
    result, err := tx.ExecContext(ctx, query,
        workflow.ID().String(),
        workflow.Name(),
        workflow.Description(),
        workflow.Status(),
        nodesJSON,
        connectionsJSON,
        settingsJSON,
        workflow.Version() + 1,
        workflow.UpdatedAt(),
        workflow.Version(), // Current version for optimistic lock
    )
    if err != nil {
        return fmt.Errorf("failed to update workflow: %w", err)
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rowsAffected == 0 {
        return model.ErrOptimisticLock
    }
    
    // Save events
    if err := r.saveEvents(ctx, tx, workflow); err != nil {
        return fmt.Errorf("failed to save events: %w", err)
    }
    
    return tx.Commit()
}

func (r *WorkflowRepository) saveEvents(ctx context.Context, tx *sql.Tx, workflow *model.Workflow) error {
    events := workflow.GetUncommittedEvents()
    if len(events) == 0 {
        return nil
    }
    
    query := `
        INSERT INTO domain_events (
            id, aggregate_id, aggregate_type, event_type, 
            event_version, event_data, user_id, created_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8
        )
    `
    
    stmt, err := tx.PrepareContext(ctx, query)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()
    
    for _, event := range events {
        eventData, err := json.Marshal(event)
        if err != nil {
            return fmt.Errorf("failed to serialize event: %w", err)
        }
        
        _, err = stmt.ExecContext(ctx,
            event.ID(),
            workflow.ID().String(),
            "Workflow",
            event.Type(),
            event.Version(),
            eventData,
            workflow.UserID(),
            event.Timestamp(),
        )
        if err != nil {
            return fmt.Errorf("failed to insert event: %w", err)
        }
    }
    
    workflow.MarkEventsAsCommitted()
    
    return nil
}
```

## Testing Implementation

### Unit Tests

```go
// internal/workflow/domain/model/workflow_test.go
package model_test

import (
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/linkflow/internal/workflow/domain/model"
)

func TestNewWorkflow(t *testing.T) {
    tests := []struct {
        name        string
        userID      string
        wfName      string
        description string
        wantErr     bool
        errMsg      string
    }{
        {
            name:        "valid workflow",
            userID:      "user123",
            wfName:      "My Workflow",
            description: "Test workflow",
            wantErr:     false,
        },
        {
            name:        "missing user ID",
            userID:      "",
            wfName:      "My Workflow",
            description: "Test workflow",
            wantErr:     true,
            errMsg:      "user ID is required",
        },
        {
            name:        "missing workflow name",
            userID:      "user123",
            wfName:      "",
            description: "Test workflow",
            wantErr:     true,
            errMsg:      "workflow name is required",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            workflow, err := model.NewWorkflow(tt.userID, tt.wfName, tt.description)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
                assert.Nil(t, workflow)
            } else {
                require.NoError(t, err)
                assert.NotNil(t, workflow)
                assert.NotEmpty(t, workflow.ID())
                assert.Equal(t, tt.userID, workflow.UserID())
                assert.Equal(t, tt.wfName, workflow.Name())
                assert.Equal(t, tt.description, workflow.Description())
                assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
                assert.Empty(t, workflow.Nodes())
                assert.Empty(t, workflow.Connections())
                
                // Check that domain event was created
                events := workflow.GetUncommittedEvents()
                assert.Len(t, events, 1)
                assert.IsType(t, model.WorkflowCreatedEvent{}, events[0])
            }
        })
    }
}

func TestWorkflow_Activate(t *testing.T) {
    t.Run("activate draft workflow with nodes", func(t *testing.T) {
        // Arrange
        workflow, _ := model.NewWorkflow("user123", "Test", "Description")
        node := model.Node{
            ID:   "node1",
            Type: model.NodeTypeTrigger,
            Name: "HTTP Trigger",
        }
        err := workflow.AddNode(node)
        require.NoError(t, err)
        
        // Act
        err = workflow.Activate()
        
        // Assert
        assert.NoError(t, err)
        assert.Equal(t, model.WorkflowStatusActive, workflow.Status())
        
        events := workflow.GetUncommittedEvents()
        assert.Len(t, events, 3) // Created, NodeAdded, Activated
    })
    
    t.Run("cannot activate workflow without nodes", func(t *testing.T) {
        // Arrange
        workflow, _ := model.NewWorkflow("user123", "Test", "Description")
        
        // Act
        err := workflow.Activate()
        
        // Assert
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "must have at least one node")
        assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
    })
    
    t.Run("cannot activate archived workflow", func(t *testing.T) {
        // Arrange
        workflow := model.ReconstructWorkflow(
            model.NewWorkflowID(),
            "user123",
            "Test",
            "Description",
            model.WorkflowStatusArchived,
            []model.Node{{ID: "node1", Type: model.NodeTypeTrigger}},
            []model.Connection{},
            model.DefaultSettings(),
            1,
            time.Now(),
            time.Now(),
        )
        
        // Act
        err := workflow.Activate()
        
        // Assert
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "can only be activated from draft or inactive")
        assert.Equal(t, model.WorkflowStatusArchived, workflow.Status())
    })
}

func TestWorkflow_AddNode(t *testing.T) {
    t.Run("add valid node", func(t *testing.T) {
        // Arrange
        workflow, _ := model.NewWorkflow("user123", "Test", "Description")
        node := model.Node{
            ID:   "node1",
            Type: model.NodeTypeAction,
            Name: "Send Email",
        }
        
        // Act
        err := workflow.AddNode(node)
        
        // Assert
        assert.NoError(t, err)
        assert.Len(t, workflow.Nodes(), 1)
        assert.Equal(t, node, workflow.Nodes()[0])
    })
    
    t.Run("cannot add duplicate node", func(t *testing.T) {
        // Arrange
        workflow, _ := model.NewWorkflow("user123", "Test", "Description")
        node := model.Node{
            ID:   "node1",
            Type: model.NodeTypeAction,
            Name: "Send Email",
        }
        _ = workflow.AddNode(node)
        
        // Act
        err := workflow.AddNode(node)
        
        // Assert
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "node with this ID already exists")
        assert.Len(t, workflow.Nodes(), 1)
    })
    
    t.Run("cannot exceed max nodes", func(t *testing.T) {
        // Arrange
        workflow, _ := model.NewWorkflow("user123", "Test", "Description")
        workflow.SetMaxNodes(2) // Set low limit for testing
        
        _ = workflow.AddNode(model.Node{ID: "node1"})
        _ = workflow.AddNode(model.Node{ID: "node2"})
        
        // Act
        err := workflow.AddNode(model.Node{ID: "node3"})
        
        // Assert
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "cannot have more than 2 nodes")
        assert.Len(t, workflow.Nodes(), 2)
    })
}
```

### Integration Tests

```go
// tests/integration/workflow_test.go
// +build integration

package integration_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/linkflow/internal/workflow/adapters/repository/postgres"
    "github.com/linkflow/internal/workflow/app/service"
)

func TestWorkflowService_Integration(t *testing.T) {
    // Setup test containers
    ctx := context.Background()
    
    postgresContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:15-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
    )
    require.NoError(t, err)
    defer postgresContainer.Terminate(ctx)
    
    // Get connection string
    connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)
    
    // Setup database
    db, err := sql.Open("postgres", connStr)
    require.NoError(t, err)
    defer db.Close()
    
    // Run migrations
    err = runMigrations(db)
    require.NoError(t, err)
    
    // Create repository
    repo := postgres.NewWorkflowRepository(db)
    
    // Create service
    svc := service.NewWorkflowService(repo, nil, nil, nil)
    
    t.Run("create and retrieve workflow", func(t *testing.T) {
        // Create workflow
        cmd := service.CreateWorkflowCommand{
            UserID:      "user123",
            Name:        "Test Workflow",
            Description: "Integration test",
        }
        
        workflow, err := svc.CreateWorkflow(ctx, cmd)
        require.NoError(t, err)
        assert.NotEmpty(t, workflow.ID)
        assert.Equal(t, cmd.Name, workflow.Name)
        
        // Retrieve workflow
        retrieved, err := svc.GetWorkflow(ctx, workflow.ID)
        require.NoError(t, err)
        assert.Equal(t, workflow.ID, retrieved.ID)
        assert.Equal(t, workflow.Name, retrieved.Name)
    })
    
    t.Run("list user workflows", func(t *testing.T) {
        // Create multiple workflows
        for i := 0; i < 5; i++ {
            cmd := service.CreateWorkflowCommand{
                UserID:      "user456",
                Name:        fmt.Sprintf("Workflow %d", i),
                Description: "Test",
            }
            _, err := svc.CreateWorkflow(ctx, cmd)
            require.NoError(t, err)
        }
        
        // List workflows
        query := service.ListWorkflowsQuery{
            UserID: "user456",
            Offset: 0,
            Limit:  10,
        }
        
        workflows, total, err := svc.ListWorkflows(ctx, query)
        require.NoError(t, err)
        assert.Len(t, workflows, 5)
        assert.Equal(t, int64(5), total)
    })
}
```

## Security Implementation

### Authentication Middleware

```go
// internal/platform/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/linkflow/internal/platform/response"
)

type AuthMiddleware struct {
    jwtSecret   []byte
    jwtVerifier JWTVerifier
}

func NewAuthMiddleware(secret []byte) *AuthMiddleware {
    return &AuthMiddleware{
        jwtSecret:   secret,
        jwtVerifier: NewJWTVerifier(secret),
    }
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            response.Error(w, response.ErrUnauthorized.WithDetails("Missing authorization header"))
            return
        }
        
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            response.Error(w, response.ErrUnauthorized.WithDetails("Invalid authorization header"))
            return
        }
        
        tokenString := parts[1]
        
        // Verify token
        claims, err := m.jwtVerifier.Verify(tokenString)
        if err != nil {
            response.Error(w, response.ErrUnauthorized.WithDetails("Invalid token"))
            return
        }
        
        // Add claims to context
        ctx := context.WithValue(r.Context(), "userID", claims.UserID)
        ctx = context.WithValue(ctx, "email", claims.Email)
        ctx = context.WithValue(ctx, "roles", claims.Roles)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            roles, ok := r.Context().Value("roles").([]string)
            if !ok {
                response.Error(w, response.ErrForbidden.WithDetails("No roles found"))
                return
            }
            
            hasRole := false
            for _, r := range roles {
                if r == role {
                    hasRole = true
                    break
                }
            }
            
            if !hasRole {
                response.Error(w, response.ErrForbidden.WithDetails("Insufficient permissions"))
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### Input Validation

```go
// internal/platform/validation/validator.go
package validation

import (
    "fmt"
    "regexp"
    
    "github.com/go-playground/validator/v10"
)

type Validator struct {
    validate *validator.Validate
}

func NewValidator() *Validator {
    v := validator.New()
    
    // Register custom validations
    v.RegisterValidation("workflow_name", validateWorkflowName)
    v.RegisterValidation("node_id", validateNodeID)
    
    return &Validator{validate: v}
}

func (v *Validator) Validate(i interface{}) error {
    if err := v.validate.Struct(i); err != nil {
        return formatValidationError(err)
    }
    return nil
}

func validateWorkflowName(fl validator.FieldLevel) bool {
    name := fl.Field().String()
    if len(name) < 3 || len(name) > 100 {
        return false
    }
    // Allow alphanumeric, spaces, hyphens, underscores
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9\s\-_]+$`, name)
    return matched
}

func validateNodeID(fl validator.FieldLevel) bool {
    id := fl.Field().String()
    // Must be valid UUID or custom format
    _, err := uuid.Parse(id)
    return err == nil
}

func formatValidationError(err error) error {
    validationErrors := err.(validator.ValidationErrors)
    
    var messages []string
    for _, e := range validationErrors {
        switch e.Tag() {
        case "required":
            messages = append(messages, fmt.Sprintf("%s is required", e.Field()))
        case "email":
            messages = append(messages, fmt.Sprintf("%s must be a valid email", e.Field()))
        case "min":
            messages = append(messages, fmt.Sprintf("%s must be at least %s", e.Field(), e.Param()))
        case "max":
            messages = append(messages, fmt.Sprintf("%s must be at most %s", e.Field(), e.Param()))
        default:
            messages = append(messages, fmt.Sprintf("%s is invalid", e.Field()))
        }
    }
    
    return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
}
```

## Performance Patterns

### Circuit Breaker

```go
// internal/platform/resilience/circuit_breaker.go
package resilience

import (
    "context"
    "errors"
    "sync"
    "time"
)

type State int

const (
    StateClosed State = iota
    StateOpen
    StateHalfOpen
)

type CircuitBreaker struct {
    mu              sync.RWMutex
    state           State
    failures        int
    successes       int
    lastFailureTime time.Time
    
    maxFailures     int
    timeout         time.Duration
    halfOpenSuccess int
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
    return &CircuitBreaker{
        state:           StateClosed,
        maxFailures:     maxFailures,
        timeout:         timeout,
        halfOpenSuccess: 3, // Require 3 successful requests to close
    }
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
    if !cb.canExecute() {
        return errors.New("circuit breaker is open")
    }
    
    err := fn()
    cb.recordResult(err)
    
    return err
}

func (cb *CircuitBreaker) canExecute() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()
    
    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        // Check if timeout has passed
        if time.Since(cb.lastFailureTime) > cb.timeout {
            cb.mu.RUnlock()
            cb.mu.Lock()
            cb.state = StateHalfOpen
            cb.successes = 0
            cb.mu.Unlock()
            cb.mu.RLock()
            return true
        }
        return false
    case StateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) recordResult(err error) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if err != nil {
        cb.failures++
        cb.lastFailureTime = time.Now()
        
        if cb.state == StateHalfOpen {
            cb.state = StateOpen
        } else if cb.failures >= cb.maxFailures {
            cb.state = StateOpen
        }
    } else {
        if cb.state == StateHalfOpen {
            cb.successes++
            if cb.successes >= cb.halfOpenSuccess {
                cb.state = StateClosed
                cb.failures = 0
            }
        } else if cb.state == StateClosed {
            cb.failures = 0
        }
    }
}
```

### Caching with Patterns

```go
// internal/platform/cache/cache.go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/vmihailenco/msgpack/v5"
)

type Cache interface {
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    InvalidatePattern(ctx context.Context, pattern string) error
}

type RedisCache struct {
    client       *redis.Client
    defaultTTL   time.Duration
    keyPrefix    string
    serializer   Serializer
}

type Serializer interface {
    Marshal(v interface{}) ([]byte, error)
    Unmarshal(data []byte, v interface{}) error
}

type MsgpackSerializer struct{}

func (s MsgpackSerializer) Marshal(v interface{}) ([]byte, error) {
    return msgpack.Marshal(v)
}

func (s MsgpackSerializer) Unmarshal(data []byte, v interface{}) error {
    return msgpack.Unmarshal(data, v)
}

func NewRedisCache(client *redis.Client, keyPrefix string) *RedisCache {
    return &RedisCache{
        client:     client,
        defaultTTL: 5 * time.Minute,
        keyPrefix:  keyPrefix,
        serializer: MsgpackSerializer{},
    }
}

func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
    fullKey := c.buildKey(key)
    
    data, err := c.client.Get(ctx, fullKey).Bytes()
    if err != nil {
        if err == redis.Nil {
            return ErrCacheMiss
        }
        return fmt.Errorf("failed to get from cache: %w", err)
    }
    
    if err := c.serializer.Unmarshal(data, dest); err != nil {
        return fmt.Errorf("failed to unmarshal cached data: %w", err)
    }
    
    return nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    fullKey := c.buildKey(key)
    
    data, err := c.serializer.Marshal(value)
    if err != nil {
        return fmt.Errorf("failed to marshal data: %w", err)
    }
    
    if ttl == 0 {
        ttl = c.defaultTTL
    }
    
    if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
        return fmt.Errorf("failed to set cache: %w", err)
    }
    
    return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
    fullKey := c.buildKey(key)
    
    if err := c.client.Del(ctx, fullKey).Err(); err != nil {
        return fmt.Errorf("failed to delete from cache: %w", err)
    }
    
    return nil
}

func (c *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
    fullPattern := c.buildKey(pattern)
    
    // Use SCAN instead of KEYS for production
    iter := c.client.Scan(ctx, 0, fullPattern, 0).Iterator()
    
    var keys []string
    for iter.Next(ctx) {
        keys = append(keys, iter.Val())
    }
    
    if err := iter.Err(); err != nil {
        return fmt.Errorf("failed to scan keys: %w", err)
    }
    
    if len(keys) > 0 {
        if err := c.client.Del(ctx, keys...).Err(); err != nil {
            return fmt.Errorf("failed to delete keys: %w", err)
        }
    }
    
    return nil
}

func (c *RedisCache) buildKey(key string) string {
    if c.keyPrefix != "" {
        return fmt.Sprintf("%s:%s", c.keyPrefix, key)
    }
    return key
}

// Cache-aside pattern implementation
func CacheAsideGet[T any](
    ctx context.Context,
    cache Cache,
    key string,
    ttl time.Duration,
    loader func() (T, error),
) (T, error) {
    var result T
    
    // Try cache first
    err := cache.Get(ctx, key, &result)
    if err == nil {
        return result, nil
    }
    
    // Load from source
    result, err = loader()
    if err != nil {
        return result, err
    }
    
    // Update cache (async to not block)
    go func() {
        _ = cache.Set(context.Background(), key, result, ttl)
    }()
    
    return result, nil
}
```

This comprehensive implementation guide provides production-ready patterns for building a scalable microservices architecture. Each component follows best practices with proper error handling, testing, security, and performance optimization.
