# Architecture Overview

LinkFlow AI is built using Domain-Driven Design (DDD) principles with a Clean Architecture approach. This document provides a high-level overview of the system architecture.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │  Web UI  │  │  CLI     │  │  SDK     │  │ Webhooks │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
└───────┼─────────────┼─────────────┼─────────────┼───────────────┘
        │             │             │             │
        └─────────────┴──────┬──────┴─────────────┘
                             │
┌────────────────────────────┼────────────────────────────────────┐
│                      API Gateway                                 │
│  ┌─────────────────────────┴─────────────────────────┐          │
│  │  Rate Limiting │ CORS │ Auth │ Logging │ Metrics  │          │
│  └───────────────────────────────────────────────────┘          │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────────┐
│                      Application Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Workflow │  │Execution │  │   Node   │  │   Auth   │        │
│  │ Service  │  │ Service  │  │ Registry │  │ Service  │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
│       │             │             │             │               │
│  ┌────┴─────────────┴─────────────┴─────────────┴────┐          │
│  │              Workflow Execution Engine             │          │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐           │          │
│  │  │ Queue   │  │ Worker  │  │Scheduler│           │          │
│  │  │ Manager │  │  Pool   │  │         │           │          │
│  │  └─────────┘  └─────────┘  └─────────┘           │          │
│  └───────────────────────────────────────────────────┘          │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────────┐
│                       Domain Layer                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ Workflow │  │Execution │  │Credential│  │Workspace │        │
│  │  Domain  │  │  Domain  │  │  Domain  │  │  Domain  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────┬───────────────────────────────────┘
                              │
┌─────────────────────────────┼───────────────────────────────────┐
│                   Infrastructure Layer                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │PostgreSQL│  │  Redis   │  │  Kafka   │  │  S3/GCS  │        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. API Gateway (`internal/gateway/`)

The entry point for all HTTP requests:
- **Rate Limiting**: Token bucket algorithm
- **CORS**: Configurable cross-origin policies
- **Authentication**: JWT and API key validation
- **Request ID**: Tracing across services

### 2. Workflow Engine (`internal/engine/`)

The heart of the execution system:
- **DAG Execution**: Processes workflow nodes in topological order
- **Parallel Processing**: Executes independent nodes concurrently
- **State Management**: Tracks execution progress
- **Error Handling**: Retry logic with exponential backoff

### 3. Node Runtime (`internal/node/runtime/`)

Executes individual workflow nodes:
- **Registry**: Manages available node types
- **Executors**: Implements node-specific logic
- **Expression Engine**: Evaluates dynamic expressions

### 4. Domain Services

Each domain handles specific business capabilities:

| Domain | Responsibility |
|--------|---------------|
| `workflow` | Workflow CRUD, versioning, templates |
| `execution` | Execution tracking, history |
| `credential` | Secure credential storage |
| `integration` | Third-party service connections |
| `schedule` | Cron-based workflow scheduling |
| `webhook` | Incoming webhook handling |
| `auth` | Authentication, authorization |
| `billing` | Subscription, usage tracking |
| `workspace` | Multi-tenancy |
| `notification` | Email, in-app notifications |

## Data Flow

### Workflow Execution Flow

```
1. Request → API Gateway → Workflow Service
                              │
2. Load Workflow Definition ──┘
                              │
3. Create Execution Record ───┤
                              │
4. Queue Execution ───────────┤
                              │
5. Worker Pool Picks Up ──────┤
                              │
6. Engine Processes DAG ──────┤
   │                          │
   ├── Execute Node 1 ────────┤
   │   └── Update State       │
   ├── Execute Node 2 ────────┤
   │   └── Update State       │
   └── Execute Node N ────────┤
       └── Update State       │
                              │
7. Mark Execution Complete ───┘
```

### WebSocket Real-time Updates

```
Client ←──── WebSocket ←──── Event Broadcaster ←──── Engine Events
```

## Scalability

### Horizontal Scaling

- **Stateless API servers**: Scale behind load balancer
- **Worker pool**: Configurable worker count
- **Queue-based execution**: Distribute load across workers

### Database Scaling

- **Connection pooling**: Configurable pool size
- **Read replicas**: For read-heavy workloads
- **Partitioning**: Execution history by date

## Security Architecture

```
┌─────────────────────────────────────────┐
│              Security Layers            │
├─────────────────────────────────────────┤
│  1. TLS/HTTPS (Transport)               │
├─────────────────────────────────────────┤
│  2. Rate Limiting (DDoS Protection)     │
├─────────────────────────────────────────┤
│  3. JWT Authentication (Identity)       │
├─────────────────────────────────────────┤
│  4. RBAC Authorization (Access Control) │
├─────────────────────────────────────────┤
│  5. AES-256 Encryption (Data at Rest)   │
└─────────────────────────────────────────┘
```

## Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go | Performance, concurrency, type safety |
| Database | PostgreSQL | ACID compliance, JSON support |
| Cache | Redis | Speed, pub/sub for real-time |
| Queue | In-memory/Redis | Simplicity, can scale to Kafka |
| Auth | JWT | Stateless, industry standard |
| Encryption | AES-256-GCM | Strong, authenticated encryption |

## Next Steps

- [Domain-Driven Design](domain-driven-design.md)
- [Service Architecture](services.md)
- [API Documentation](../api/overview.md)
