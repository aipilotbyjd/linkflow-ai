# Local Deployment Guide

Complete guide to run LinkFlow AI on your local machine (macOS/Linux).

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Deployment Options](#deployment-options)
- [Database Setup](#database-setup)
- [Service Access Points](#service-access-points)
- [Common Commands](#common-commands)
- [Environment Configuration](#environment-configuration)
- [Troubleshooting](#troubleshooting)
- [Testing the API](#testing-the-api)

---

## Prerequisites

### Required Software

| Software | Version | Installation (macOS) |
|----------|---------|----------------------|
| Go | 1.25+ | `brew install go` |
| Docker | 20.0+ | [Docker Desktop](https://www.docker.com/products/docker-desktop) |
| Docker Compose | 2.0+ | Included with Docker Desktop |
| golang-migrate | latest | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

### Verify Installation

```bash
go version                    # Should show go1.21+
docker --version              # Should show 20.0+
docker compose version        # Should show 2.0+
migrate -version              # Should show migrate version
```

### Install Development Tools

```bash
make install-tools
```

This installs: `air` (hot reload), `migrate`, `golangci-lint`, `staticcheck`, `mockgen`

---

## Quick Start

**5-minute setup:**

```bash
# 1. Copy environment file
cp .env.example .env

# 2. Start infrastructure (PostgreSQL, Redis, Kafka, Elasticsearch)
docker compose up -d postgres redis kafka zookeeper elasticsearch

# 3. Wait for services to be ready (about 30 seconds)
sleep 30

# 4. Initialize database
docker exec -i linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow;" 2>/dev/null || true
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql

# 5. Run migrations
make migrate

# 6. Start all services
make docker-up

# 7. Verify health
make health
```

---

## Deployment Options

### Option 1: Full Docker Deployment (Recommended for Testing)

Runs all 21 microservices + infrastructure in Docker containers.

```bash
# Start everything
make docker-up

# View logs
make logs

# Check status
docker compose ps

# Stop everything
make docker-down
```

**What's Started:**
- **Microservices (21):** auth, user, workflow, execution, node, tenant, executor, webhook, schedule, credential, notification, integration, analytics, search, storage, config, admin, monitoring, migration, backup, gateway
- **API Gateway:** Kong (ports 8000, 8001)
- **Database:** PostgreSQL (port 5432)
- **Cache:** Redis (port 6379)
- **Message Queue:** Kafka (port 9092) + Zookeeper (port 2181)
- **Search:** Elasticsearch (port 9200)
- **Monitoring:** Prometheus (9090), Grafana (3000), Jaeger (16686)
- **Dev Tools:** Adminer (8080), Kafka UI (8090)

### Option 2: Infrastructure + Local Services (Development)

Best for active development with hot reload.

```bash
# 1. Start infrastructure only
docker compose up -d postgres redis kafka zookeeper elasticsearch

# 2. Initialize database
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql

# 3. Run migrations
make migrate

# 4. Start with hot reload (requires air)
make dev

# Or run individual services:
make run-auth       # Auth service on :8001
make run-workflow   # Workflow service on :8004
make run-execution  # Execution service on :8003
```

### Option 3: Minimal Setup (API Server Only)

For quick API testing with single server.

```bash
# 1. Start PostgreSQL and Redis only
docker compose up -d postgres redis

# 2. Initialize and migrate
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql
make migrate

# 3. Run main API server (monolithic mode)
go run ./cmd/services/api
```

---

## Database Setup

### Migrations Structure

```
migrations/
├── 000001_init_extensions.up.sql      # PostgreSQL extensions
├── 000002_users_auth.up.sql           # Users, sessions, tokens
├── 000003_organizations.up.sql        # Organizations, members
├── 000004_workspaces.up.sql           # Workspaces, API keys
├── 000005_workflows.up.sql            # Workflows, versions
├── 000006_executions.up.sql           # Executions, logs
├── 000007_nodes.up.sql                # Node definitions
├── 000008_webhooks.up.sql             # Webhooks, logs
├── 000009_schedules.up.sql            # Schedules
├── 000010_credentials.up.sql          # Credentials, variables
├── 000011_notifications.up.sql        # Notifications
├── 000012_tenants.up.sql              # Tenants, limits
├── 000013_billing.up.sql              # Plans, subscriptions
├── 000014_executor.up.sql             # Workers, tasks
├── 000015_storage.up.sql              # Files
├── 000016_analytics.up.sql            # Analytics events
├── 000017_integrations.up.sql         # Integrations
├── 000018_config.up.sql               # Configurations
└── 000019_audit.up.sql                # Audit logs
```

### Migration Commands

```bash
make migrate              # Run all pending migrations
make migrate-down         # Rollback 1 migration
make migrate-down-all     # Rollback all migrations
make migrate-version      # Show current version
make migrate-force V=5    # Force to version 5 (use with caution)
```

### Manual Database Access

```bash
# Connect to PostgreSQL
docker exec -it linkflow-postgres psql -U postgres -d linkflow

# Common queries
\dt                       # List tables
\d+ users                 # Describe users table
SELECT * FROM users;      # Query users
```

---

## Service Access Points

### API Gateway (Kong)

| Endpoint | URL | Description |
|----------|-----|-------------|
| Proxy | http://localhost:8000 | Main API entry point |
| Admin | http://localhost:8001 | Kong admin API |
| Proxy SSL | https://localhost:8443 | HTTPS proxy |
| Admin SSL | https://localhost:8444 | HTTPS admin |

### API Routes (via Kong Gateway)

| Service | Endpoint | Auth |
|---------|----------|------|
| Auth | `http://localhost:8000/api/v1/auth` | No |
| Users | `http://localhost:8000/api/v1/users` | JWT |
| Workflows | `http://localhost:8000/api/v1/workflows` | JWT |
| Executions | `http://localhost:8000/api/v1/executions` | JWT |
| Nodes | `http://localhost:8000/api/v1/nodes` | JWT |
| Schedules | `http://localhost:8000/api/v1/schedules` | JWT |
| Webhooks | `http://localhost:8000/api/v1/webhooks` | JWT |
| Credentials | `http://localhost:8000/api/v1/credentials` | JWT |
| Notifications | `http://localhost:8000/api/v1/notifications` | JWT |
| Analytics | `http://localhost:8000/api/v1/analytics` | JWT |
| Search | `http://localhost:8000/api/v1/search` | JWT |
| Storage | `http://localhost:8000/api/v1/storage` | JWT |
| Integrations | `http://localhost:8000/api/v1/integrations` | JWT |
| Executor | `http://localhost:8000/api/v1/executor` | JWT |
| Config | `http://localhost:8000/api/v1/config` | JWT + IP |
| Monitoring | `http://localhost:8000/api/v1/monitoring` | JWT + IP |
| Admin | `http://localhost:8000/api/v1/admin` | JWT + IP |
| Tenants | `http://localhost:8000/api/v1/tenants` | JWT |
| Health | `http://localhost:8000/health` | No |

### Infrastructure UIs

| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Jaeger (Tracing) | http://localhost:16686 | - |
| Kafka UI | http://localhost:8090 | - |
| Adminer (DB) | http://localhost:8080 | postgres / postgres |
| Prometheus | http://localhost:9090 | - |
| Elasticsearch | http://localhost:9200 | - |

### Direct Service Ports

| Service | Internal | External | Description |
|---------|----------|----------|-------------|
| Gateway | 8000 | 8100 | Internal gateway |
| Auth | 8001 | 8101 | Authentication |
| User | 8002 | 8102 | User management |
| Execution | 8003 | 8103 | Execution engine |
| Workflow | 8004 | 8104 | Workflow CRUD |
| Node | 8005 | 8105 | Node definitions |
| Tenant | 8006 | 8106 | Multi-tenancy |
| Executor | 8007 | 8107 | Task execution |
| Webhook | 8008 | 8108 | Webhook handling |
| Schedule | 8009 | 8109 | Cron scheduling |
| Credential | 8010 | 8110 | Credential vault |
| Notification | 8011 | 8111 | Notifications |
| Integration | 8012 | 8112 | 3rd party integrations |
| Analytics | 8013 | 8113 | Analytics |
| Search | 8014 | 8114 | Search service |
| Storage | 8015 | 8115 | File storage |
| Config | 8016 | 8116 | Configuration |
| Admin | 8017 | 8117 | Admin panel |
| Monitoring | 8019 | 8119 | Health monitoring |

---

## Common Commands

### Service Management

```bash
make docker-up           # Start all Docker services
make docker-down         # Stop all Docker services
make logs                # View all service logs
make status              # Show service status
make health              # Check health endpoints
```

### Development

```bash
make dev                 # Start with hot reload
make build               # Build all services
make build-all           # Build all 21 services
make build-docker        # Build Docker images
make clean               # Clean build artifacts
```

### Database

```bash
make migrate             # Run migrations
make migrate-down        # Rollback 1 step
make migrate-version     # Show version
make seed                # Seed test data
```

### Testing

```bash
make test                # Run all tests
make test-unit           # Unit tests only
make test-integration    # Integration tests
make test-coverage       # With coverage report
make lint                # Run linters
make validate            # Lint + test
```

### Building

```bash
make build               # Build for current OS
make build-all           # Build all services
make build-docker        # Build Docker images
make docker-push         # Push to registry
```

---

## Environment Configuration

Copy `.env.example` to `.env` and configure:

### Core Settings

```bash
ENVIRONMENT=development
LOG_LEVEL=debug
```

### Database

```bash
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=linkflow
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_SSL_MODE=disable
```

### Redis

```bash
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

### Authentication

```bash
JWT_SECRET=linkflow-dev-secret-key-change-in-production-min-32-chars
JWT_EXPIRATION=24h
```

### Rate Limiting

```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
RATE_LIMIT_BURST_SIZE=200
```

### Feature Flags

```bash
FEATURE_WORKFLOW_VERSIONING=true
FEATURE_MULTI_TENANCY=true
FEATURE_WEBHOOKS=true
FEATURE_AUDIT_LOGGING=true
```

### External Services (Optional)

```bash
# Stripe (Billing)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# OAuth Providers
OAUTH_GOOGLE_CLIENT_ID=
OAUTH_GOOGLE_CLIENT_SECRET=
OAUTH_GITHUB_CLIENT_ID=
OAUTH_GITHUB_CLIENT_SECRET=

# Email (SMTP)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=
SMTP_PASSWORD=
```

---

## Troubleshooting

### Port Already in Use

```bash
# Find process on port
lsof -i :8000

# Kill process
kill -9 <PID>

# Or use different port in .env
HTTP_PORT=8081
```

### Docker Services Won't Start

```bash
# Check Docker status
docker info

# View container logs
docker compose logs postgres
docker compose logs kong

# Reset everything
make docker-down
docker system prune -f
docker volume prune -f
make docker-up
```

### Database Connection Failed

```bash
# Check PostgreSQL is running
docker exec linkflow-postgres pg_isready -U postgres

# Check connection
docker exec -it linkflow-postgres psql -U postgres -c "SELECT 1"

# Reinitialize database
docker exec -i linkflow-postgres psql -U postgres -c "DROP DATABASE IF EXISTS linkflow"
docker exec -i linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow"
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql
make migrate
```

### Migration Errors

```bash
# Check current version
make migrate-version

# Force to specific version (if stuck)
make migrate-force V=000005

# Start fresh
make migrate-down-all
make migrate
```

### Kong Configuration Issues

```bash
# Validate Kong config
docker exec linkflow-kong kong config parse /kong/kong.yml

# Check Kong status
docker exec linkflow-kong kong health

# Reload Kong config
docker exec linkflow-kong kong reload

# View Kong logs
docker compose logs kong
```

### Service Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f workflow

# Last 100 lines
docker compose logs --tail=100 auth

# Multiple services
docker compose logs -f auth workflow execution
```

---

## Testing the API

### Health Check

```bash
curl http://localhost:8000/health
```

### Register User

```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "Password123!"
  }'
```

### Login

```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Password123!"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "...",
  "expires_at": "2024-12-21T12:00:00Z"
}
```

### Create Workflow

```bash
export TOKEN="your-jwt-token-here"

curl -X POST http://localhost:8000/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "My First Workflow",
    "description": "A simple test workflow",
    "nodes": [
      {
        "id": "trigger-1",
        "type": "manual_trigger",
        "name": "Start",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "http-1",
        "type": "http_request",
        "name": "Fetch Data",
        "config": {
          "url": "https://api.github.com/zen",
          "method": "GET"
        },
        "position": {"x": 300, "y": 100}
      }
    ],
    "connections": [
      {
        "source": "trigger-1",
        "target": "http-1"
      }
    ]
  }'
```

### Execute Workflow

```bash
curl -X POST http://localhost:8000/api/v1/workflows/{workflow_id}/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
```

### List Executions

```bash
curl http://localhost:8000/api/v1/executions \
  -H "Authorization: Bearer $TOKEN"
```

---

## Next Steps

- [API Reference](../api/overview.md)
- [Creating Workflows](../guides/creating-workflows.md)
- [Node Types Reference](../reference/node-types.md)
- [Testing Guide](../development/testing.md)
- [Architecture Overview](../architecture/overview.md)
