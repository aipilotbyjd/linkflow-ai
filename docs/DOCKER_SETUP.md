# LinkFlow AI - Docker Setup

Production-ready Docker infrastructure with Kong API Gateway.

## Quick Start

```bash
# Development (all ports exposed)
make dev

# Production (Kong only on 80/443)
make prod

# Stop
make stop

# Status & Health
make status
make health
```

## Architecture

```
┌─────────────────────────────────────┐
│           INTERNET                  │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│        EDGE NETWORK                 │
│  Kong (80/443), Grafana (3000)      │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│        APP NETWORK                  │
│  21 Microservices (8001-8022)       │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│        DATA NETWORK                 │
│  Postgres, Redis, Kafka, ES         │
└─────────────────────────────────────┘
```

## Files

```
configs/
├── envs/
│   ├── .env.example        # Template
│   ├── .env.dev            # Development defaults
│   └── .env.prod.example   # Production template
├── kong/
│   ├── kong.yml            # Base config
│   ├── kong.dev.yml        # Dev (relaxed)
│   └── kong.prod.yml       # Prod (strict)
├── grafana/
└── prometheus/

deployments/docker/compose/
├── base.yml                # 32 services, 3 networks
├── dev.yml                 # All ports exposed
└── prod.yml                # Kong only

scripts/
└── linkflow.sh             # Unified CLI
```

## Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start development (all ports) |
| `make prod` | Start production (Kong only) |
| `make stop` | Stop all services |
| `make status` | Show service status |
| `make health` | Health check all services |
| `make logs` | View all logs |
| `make logs-workflow` | View workflow service logs |
| `make shell-postgres` | Open PostgreSQL shell |
| `make db-migrate` | Run migrations |
| `make db-reset` | Reset database |

## Services (32 total)

### Data Layer
- postgres (5432)
- redis (6379)
- kafka (9092)
- zookeeper (2181)
- elasticsearch (9200)

### Edge Layer
- kong (8000, 8001, 8443)

### Application Layer
- auth (8001)
- user (8002)
- execution (8003)
- workflow (8004)
- node (8005)
- executor (8007)
- webhook (8008)
- schedule (8009)
- credential (8010)
- notification (8011)
- integration (8012)
- analytics (8013)
- search (8014)
- storage (8015)
- config (8016)
- admin (8017)
- tenant (8019)
- monitoring (8020)
- backup (8021)
- migration (8022)
- gateway (8080)

### Monitoring
- prometheus (9090)
- grafana (3000)
- jaeger (16686)

### Admin Tools (dev only)
- adminer (8089)
- kafka-ui (8088)

## Environment Setup

```bash
# Copy template
cp configs/envs/.env.example .env

# Edit with your values
vim .env

# Start
make dev
```

## Production Deployment

1. Copy production template:
   ```bash
   cp configs/envs/.env.prod.example .env
   ```

2. Fill in production values (secrets, database URLs, etc.)

3. Use production Kong config:
   ```bash
   cp configs/kong/kong.prod.yml configs/kong/kong.yml
   ```

4. Start production:
   ```bash
   make prod
   ```

## Network Isolation

- **edge-network**: Public-facing (Kong, Grafana)
- **app-network**: Internal services
- **data-network**: Databases and caches

In production, only Kong is exposed on ports 80/443. All other services are internal only.
