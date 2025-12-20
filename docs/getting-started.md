# Getting Started

Complete guide to run LinkFlow AI locally.

## Prerequisites

| Software | Version | Install (macOS) |
|----------|---------|-----------------|
| Go | 1.25+ | `brew install go` |
| Docker | 20.0+ | [Docker Desktop](https://docker.com/products/docker-desktop) |
| Make | any | Pre-installed on macOS |

```bash
# Verify installations
go version              # go1.25+
docker --version        # 20.0+
docker compose version  # 2.0+
```

## Quick Start

```bash
# 1. Clone and enter directory
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai

# 2. Copy environment file
cp .env.example .env

# 3. Start development environment
make dev

# 4. Check status
make status
make health
```

That's it! All 32 services will start with:
- PostgreSQL, Redis, Kafka, Elasticsearch
- Kong API Gateway on port 8000
- 21 microservices
- Monitoring stack (Prometheus, Grafana, Jaeger)

## Access Points

| Service | URL | Credentials |
|---------|-----|-------------|
| API Gateway | http://localhost:8000 | - |
| Grafana | http://localhost:3000 | admin/admin |
| Jaeger | http://localhost:16686 | - |
| Kafka UI | http://localhost:8088 | - |
| Adminer (DB) | http://localhost:8089 | postgres/postgres |
| Prometheus | http://localhost:9090 | - |

## Commands

```bash
# Development
make dev              # Start all services (all ports exposed)
make stop             # Stop all services
make restart          # Restart all services
make status           # Show service status
make health           # Health check all services

# Logs
make logs             # All logs
make logs-workflow    # Specific service logs
make logs-auth        # Auth service logs

# Database
make db-psql          # PostgreSQL shell
make db-migrate       # Run migrations
make db-reset         # Reset database (WARNING: deletes data)

# Shell access
make shell-postgres   # PostgreSQL shell
make shell-redis      # Redis CLI
make shell-auth       # Shell into auth container

# Building
make build            # Build all Go binaries
make build-docker     # Build Docker images

# Testing
make test             # Run all tests
make test-coverage    # Tests with coverage
make lint             # Run linters
```

## Project Structure

```
linkflow-ai/
├── cmd/services/           # Service entry points (21 services)
├── internal/               # Private application code
├── pkg/                    # Public packages
├── configs/
│   ├── kong/               # Kong API Gateway configs
│   ├── envs/               # Environment templates
│   ├── prometheus/         # Prometheus config
│   └── grafana/            # Grafana dashboards
├── deployments/
│   └── docker/compose/     # Docker Compose files
│       ├── base.yml        # Base service definitions
│       ├── dev.yml         # Development (all ports)
│       └── prod.yml        # Production (Kong only)
├── migrations/             # Database migrations
├── scripts/
│   └── linkflow.sh         # Unified CLI
└── docs/                   # Documentation
```

## Environment Configuration

Copy and customize:
```bash
cp .env.example .env
# or use pre-configured dev settings:
cp configs/envs/.env.dev .env
```

Key variables:
```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=linkflow

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Auth
JWT_SECRET=your-secret-key-min-32-chars

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json
```

See `configs/envs/.env.example` for all options.

## Troubleshooting

### Services not starting
```bash
# Check Docker is running
docker info

# View service logs
make logs-<service>

# Restart specific service
docker compose restart <service>
```

### Database connection issues
```bash
# Check PostgreSQL is ready
docker exec linkflow-postgres pg_isready

# View PostgreSQL logs
make logs-postgres

# Reset database
make db-reset
```

### Port conflicts
```bash
# Check what's using a port
lsof -i :8000

# Stop conflicting service or change port in .env
```

### Build failures
```bash
# Clear Docker cache
docker system prune -af

# Rebuild from scratch
make rebuild
```

## Next Steps

- [API Reference](api/overview.md) - API documentation
- [Architecture](architecture/overview.md) - System design
- [Node Types](reference/node-types.md) - Available workflow nodes
