# Local Deployment Guide

Complete guide to run LinkFlow AI on your local machine (macOS/Linux).

## Prerequisites

### Required Software
| Software | Version | Installation |
|----------|---------|--------------|
| Go | 1.21+ | `brew install go` |
| Docker | 20.0+ | [Docker Desktop](https://www.docker.com/products/docker-desktop) |
| Docker Compose | 2.0+ | Included with Docker Desktop |

### Verify Installation
```bash
go version          # Should show go1.21+
docker --version    # Should show 20.0+
docker compose version
```

## Quick Start (5 minutes)

```bash
# 1. Clone and enter the repository
cd linkflow-ai

# 2. Run quick start script
chmod +x scripts/quick-start.sh
./scripts/quick-start.sh

# 3. Start all services
make docker-up

# 4. Verify services are running
make health
```

## Deployment Options

### Option 1: Full Docker Deployment (Recommended)

Best for testing the complete platform with all microservices.

```bash
# Start everything
make docker-up

# View logs
make logs

# Stop everything
make docker-down
```

**Services Started:**
- 21 microservices (auth, workflow, execution, etc.)
- Kong API Gateway (port 8000)
- PostgreSQL (port 5432)
- Redis (port 6379)
- Kafka (port 9092)
- Elasticsearch (port 9200)
- Prometheus (port 9090)
- Grafana (port 3000)
- Jaeger (port 16686)

### Option 2: Infrastructure Only + Local Services

Best for active development with hot reload.

```bash
# 1. Start infrastructure only
docker compose up -d postgres redis kafka elasticsearch

# 2. Install development tools
make install-tools

# 3. Initialize database
docker exec -i linkflow-postgres psql -U postgres < scripts/init-db.sql

# 4. Run specific service with hot reload
make run-auth      # Auth service
make run-workflow  # Workflow service

# Or run all with hot reload
make dev
```

### Option 3: Minimal Setup (API only)

Best for quick API testing.

```bash
# 1. Start PostgreSQL and Redis only
docker compose up -d postgres redis

# 2. Run migrations
make migrate-all

# 3. Run main API server
go run ./cmd/services/api
```

## Service Access Points

### API Gateway (Kong)
| Endpoint | URL |
|----------|-----|
| Proxy (API requests) | http://localhost:8000 |
| Admin API | http://localhost:8001 |
| Proxy SSL | https://localhost:8443 |

### API Routes via Kong
| Service | Endpoint |
|---------|----------|
| Auth | http://localhost:8000/api/v1/auth |
| Users | http://localhost:8000/api/v1/users |
| Workflows | http://localhost:8000/api/v1/workflows |
| Executions | http://localhost:8000/api/v1/executions |
| Nodes | http://localhost:8000/api/v1/nodes |
| Webhooks | http://localhost:8000/api/v1/webhooks |
| Schedules | http://localhost:8000/api/v1/schedules |
| Credentials | http://localhost:8000/api/v1/credentials |
| Notifications | http://localhost:8000/api/v1/notifications |
| Integrations | http://localhost:8000/api/v1/integrations |
| Analytics | http://localhost:8000/api/v1/analytics |
| Search | http://localhost:8000/api/v1/search |
| Storage | http://localhost:8000/api/v1/storage |

### Infrastructure UIs
| Service | URL | Credentials |
|---------|-----|-------------|
| Grafana | http://localhost:3000 | admin / admin |
| Jaeger (Tracing) | http://localhost:16686 | - |
| Kafka UI | http://localhost:8090 | - |
| Adminer (DB) | http://localhost:8080 | postgres / postgres |
| Prometheus | http://localhost:9090 | - |

### Direct Service Ports (Development)
| Service | Internal Port | External Port |
|---------|--------------|---------------|
| Auth | 8001 | 8101 |
| User | 8002 | 8102 |
| Execution | 8003 | 8103 |
| Workflow | 8004 | 8104 |
| Node | 8005 | 8105 |
| Tenant | 8006 | 8106 |
| Executor | 8007 | 8107 |
| Webhook | 8008 | 8108 |
| Schedule | 8009 | 8109 |
| Credential | 8010 | 8110 |
| Notification | 8011 | 8111 |
| Integration | 8012 | 8112 |
| Analytics | 8013 | 8113 |
| Search | 8014 | 8114 |
| Storage | 8015 | 8115 |
| Config | 8016 | 8116 |
| Admin | 8017 | 8117 |
| Monitoring | 8019 | 8119 |
| Gateway | 8000 | 8100 |

## Common Commands

### Service Management
```bash
make docker-up      # Start all services
make docker-down    # Stop all services
make logs           # View all logs
make status         # Check service status
make health         # Check health endpoints
```

### Development
```bash
make dev            # Start with hot reload
make build          # Build all services
make build-docker   # Build Docker images
make clean          # Clean build artifacts
```

### Database
```bash
make migrate-all    # Run all migrations
make migrate-down   # Rollback migrations
make seed           # Seed test data
```

### Testing
```bash
make test           # Run all tests
make test-unit      # Unit tests only
make test-coverage  # With coverage report
make lint           # Run linters
make validate       # Lint + test
```

## Environment Configuration

The `.env` file contains all configuration. Key sections:

### Database
```bash
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=linkflow
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
```

### Authentication
```bash
JWT_SECRET=your-secret-key-min-32-chars
JWT_EXPIRATION=24h
```

### Rate Limiting
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
```

### Feature Flags
```bash
FEATURE_WORKFLOW_VERSIONING=true
FEATURE_MULTI_TENANCY=true
FEATURE_WEBHOOKS=true
```

## Troubleshooting

### Port Already in Use
```bash
# Find and kill process on port
lsof -i :8000
kill -9 <PID>
```

### Docker Services Won't Start
```bash
# Check Docker is running
docker info

# Reset Docker environment
make docker-down
docker system prune -f
make docker-up
```

### Database Connection Failed
```bash
# Check PostgreSQL is running
docker exec linkflow-postgres pg_isready -U postgres

# Reinitialize database
docker exec -i linkflow-postgres psql -U postgres < scripts/init-db.sql
```

### Kong Configuration Issues
```bash
# Validate Kong config
docker exec linkflow-kong kong config parse /kong/kong.yml

# Reload Kong config
docker exec linkflow-kong kong reload
```

### View Service Logs
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f workflow

# Kong gateway
docker compose logs -f kong
```

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
    "password": "password123"
  }'
```

### Login
```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

### Create Workflow (with JWT token)
```bash
curl -X POST http://localhost:8000/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{
    "name": "My First Workflow",
    "description": "Test workflow",
    "nodes": [],
    "connections": []
  }'
```

## Next Steps

- [API Reference](../api/overview.md)
- [Creating Workflows](../guides/creating-workflows.md)
- [Node Types](../reference/node-types.md)
- [Testing Guide](../development/testing.md)
