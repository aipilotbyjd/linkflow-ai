# LinkFlow AI - Agent Instructions

> A production-ready workflow automation platform (n8n/Zapier clone) built in Go.

## Project Overview

LinkFlow AI is a workflow automation platform that allows users to create, execute, and manage automated workflows. The platform supports various node types (triggers, actions, conditions), integrations with third-party services, and a robust execution engine.

**Tech Stack:**
- **Language:** Go 1.25+
- **Database:** PostgreSQL
- **Cache:** Redis (optional)
- **Architecture:** Domain-Driven Design (DDD) with Clean Architecture
- **API:** REST with OpenAPI specs

## Directory Structure

```
linkflow-ai/
├── cmd/                    # Application entry points
│   ├── services/           # Individual service entry points
│   │   ├── api/            # Main API server
│   │   ├── workflow/       # Workflow service
│   │   ├── execution/      # Execution service
│   │   └── ...
│   ├── tools/              # CLI tools
│   └── workers/            # Background workers
├── internal/               # Private application code
│   ├── auth/               # Authentication & authorization
│   ├── billing/            # Stripe billing integration
│   ├── credential/         # Credential management with encryption
│   ├── engine/             # Workflow execution engine
│   ├── execution/          # Execution tracking
│   ├── executor/           # Worker pool & task execution
│   ├── gateway/            # API gateway
│   ├── integration/        # Third-party integrations
│   ├── node/               # Node definitions & runtime
│   ├── notification/       # Email & notifications
│   ├── platform/           # Shared platform utilities
│   │   ├── config/         # Configuration loading
│   │   ├── database/       # Database connections
│   │   ├── di/             # Dependency injection
│   │   ├── middleware/     # HTTP middleware
│   │   └── validation/     # Input validation
│   ├── schedule/           # Cron scheduling
│   ├── webhook/            # Webhook handling
│   ├── workflow/           # Workflow domain
│   │   ├── domain/         # Domain models & repository interfaces
│   │   ├── adapters/       # Repository implementations & HTTP handlers
│   │   ├── app/            # Application services
│   │   └── features/       # Feature modules (templates, sharing, debug)
│   └── workspace/          # Multi-tenancy
├── pkg/                    # Public packages
│   ├── api/                # API utilities
│   ├── expression/         # Expression parser
│   ├── middleware/         # Reusable middleware
│   └── validators/         # Validation utilities
├── api/                    # OpenAPI specifications
├── migrations/             # Database migrations
├── deployments/            # Kubernetes & Helm charts
└── configs/                # Configuration files
```

## Architecture Patterns

### Domain-Driven Design Structure
Each domain follows this structure:
```
internal/<domain>/
├── domain/
│   ├── model/          # Domain entities and value objects
│   └── repository/     # Repository interfaces
├── adapters/
│   ├── http/           # HTTP handlers and DTOs
│   └── repository/     # Repository implementations (postgres/)
├── app/
│   └── service/        # Application services
└── server/             # Server setup (optional)
```

### Key Conventions

1. **Error Handling:** Use custom error types in `domain/repository/errors.go`
2. **Context:** Always pass `context.Context` as the first parameter
3. **Validation:** Use `internal/platform/validation/validator.go` for input validation
4. **Logging:** Use structured logging with key-value pairs
5. **IDs:** Use UUIDs for entity identifiers via `github.com/google/uuid`

## Building and Running

```bash
# Build all packages
go build ./...

# Run API server
go run ./cmd/services/api

# Run specific service
go run ./cmd/services/workflow

# Run database migrations
go run ./cmd/tools/migrate up

# Run tests (currently some test suites are failing)
go test ./...
```

## Important Files

### Configuration
- `configs/` - YAML configuration files
- Environment variables are loaded via `internal/platform/config/`

### Key Entry Points
- `cmd/services/api/main.go` - Main API server with middleware chain and DI container
- `internal/engine/engine.go` - Workflow execution engine
- `internal/node/runtime/registry.go` - Node type registry

### Domain Models
- `internal/workflow/domain/model/workflow.go` - Workflow entity
- `internal/execution/domain/model/` - Execution tracking
- `internal/node/domain/model/` - Node definitions
- `internal/credential/domain/model/` - Credential storage

### Features
- `internal/workflow/features/templates.go` - Workflow templates
- `internal/workflow/features/sharing.go` - Workflow sharing
- `internal/workflow/features/debug.go` - Debug mode with breakpoints
- `internal/workflow/features/replay.go` - Execution replay

## Node System

### Node Types
Nodes are implemented in `internal/node/runtime/nodes/` and registered automatically via init().

Available node types:
- **Triggers:** Manual, Schedule, Webhook, Interval
- **Control Flow:** IF/Switch, Loop, Merge, Split
- **Actions:** HTTP Request, Code/Function, Set, Wait/Delay
- **Integrations:** Slack, Email, PostgreSQL, MySQL, MongoDB, GitHub, Google Sheets, Notion, Airtable, S3, Discord, Telegram

### Adding a New Node
1. Create file in `internal/node/runtime/nodes/`
2. Implement the `NodeExecutor` interface
3. Register via `runtime.Register()` in init()

## Database

### Migrations
Located in `migrations/` directory. Run with:
```bash
go run ./cmd/tools/migrate up
```

### Key Tables
- `workflows` - Workflow definitions
- `executions` - Execution history
- `nodes` - Node configurations
- `credentials` - Encrypted credentials
- `users` - User accounts
- `workspaces` - Multi-tenant workspaces
- `notifications` - User notifications

## Security

### Credential Encryption
- Uses AES-256-GCM encryption
- Implementation: `internal/credential/encryption.go`
- Sensitive fields are auto-encrypted: password, secret, api_key, access_token, refresh_token, private_key, client_secret

### Authentication
- JWT-based authentication: `pkg/middleware/auth.go`
- API key support for service-to-service communication
- OAuth2 provider support (Google, GitHub, Microsoft)

### Middleware
- Rate limiting: `pkg/middleware/ratelimit.go` (token bucket algorithm)
- CORS: `pkg/middleware/cors.go`
- Recovery: `pkg/middleware/recovery.go`

## Testing Guidelines

**Note:** Some test suites (`tests/integration`, `tests/security`) currently have build issues.

When writing tests:
1. Use table-driven tests
2. Mock external dependencies
3. Use `testify/assert` for assertions
4. Place tests next to the code they test (`*_test.go`)

## Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Keep functions focused and small
- Document exported functions with comments
- Use meaningful variable names (avoid single letters except for loops)

## Common Tasks

### Adding a New API Endpoint
1. Define handler in `internal/<domain>/adapters/http/handlers/`
2. Create DTOs in `internal/<domain>/adapters/http/dto/`
3. Wire route in the service's server setup
4. Add validation using `internal/platform/validation/`

### Adding a New Integration
1. Create connector in `internal/integration/connectors/`
2. Implement the `Connector` interface
3. Register in the connector registry
4. Add OAuth configuration if needed

### Modifying Workflow Execution
1. Engine is in `internal/engine/engine.go`
2. Node executors are in `internal/node/runtime/nodes/`
3. Execution state is tracked via `internal/execution/`

## Environment Variables

```bash
PORT=8080                                    # API server port
DATABASE_URL=postgres://localhost:5432/linkflow  # PostgreSQL connection
JWT_SECRET=your-secret-key                   # JWT signing key
STRIPE_SECRET_KEY=sk_...                     # Stripe API key
STRIPE_WEBHOOK_SECRET=whsec_...              # Stripe webhook secret
ENVIRONMENT=development                       # development/staging/production
RATE_LIMIT_PER_MIN=100                       # API rate limit
ALLOWED_ORIGINS=*                            # CORS allowed origins
```

## Current Status

The project has complete backend implementation including:
- Core workflow engine with parallel execution
- 15+ node types
- 13+ integration connectors
- Full authentication & authorization
- Billing with Stripe
- WebSocket real-time updates
- Workflow templates and sharing

**Not yet implemented:**
- Frontend UI (workflow canvas, node palette, etc.)

See `N8N_CLONE_TRACKING.md` for detailed feature status.
