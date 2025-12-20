# LinkFlow AI

Production-ready workflow automation platform built with Go microservices.

## Features

- **21 Microservices** - Modular, scalable architecture
- **Kong API Gateway** - Rate limiting, authentication, routing
- **Event-Driven** - Kafka-based event streaming
- **Multi-Channel Notifications** - Email, SMS, Slack, Discord
- **Third-Party Integrations** - GitHub, Google Drive, Jira, Zapier
- **Real-Time Monitoring** - Prometheus, Grafana, Jaeger

## Quick Start

```bash
# Clone
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai

# Configure
cp .env.example .env

# Start
make dev

# Check status
make status
make health
```

**Access:**
- API: http://localhost:8000
- Grafana: http://localhost:3000 (admin/admin)
- Jaeger: http://localhost:16686

## Commands

```bash
make dev          # Start development (all ports exposed)
make prod         # Start production (Kong only on 80/443)
make stop         # Stop all services
make status       # Service status
make health       # Health checks
make logs         # View logs
make logs-<svc>   # Service-specific logs (e.g., make logs-workflow)
```

## Architecture

```
┌─────────────────────────────────────────┐
│            Kong API Gateway              │
│              (port 8000)                 │
└─────────────────────────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│  Auth   │    │Workflow │    │Execution│
│ Service │    │ Service │    │ Service │
└─────────┘    └─────────┘    └─────────┘
         ...18 more services...
                     │
         ┌──────────────────────┐
         │   Event Bus (Kafka)   │
         └──────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│PostgreSQL│   │  Redis  │    │  Elastic │
└─────────┘    └─────────┘    └─────────┘
```

## Project Structure

```
linkflow-ai/
├── cmd/services/           # 21 microservices
├── internal/               # Business logic
├── configs/
│   ├── kong/               # API Gateway config
│   └── envs/               # Environment templates
├── deployments/
│   └── docker/compose/     # Docker Compose files
├── migrations/             # Database migrations
├── scripts/
│   └── linkflow.sh         # CLI tool
└── docs/                   # Documentation
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| Kong | 8000 | API Gateway |
| Auth | 8001 | Authentication & JWT |
| User | 8002 | User management |
| Workflow | 8004 | Workflow CRUD |
| Execution | 8003 | Workflow orchestration |
| Node | 8005 | Node definitions |
| Schedule | 8009 | Cron scheduling |
| Webhook | 8008 | Webhook handling |
| Notification | 8011 | Multi-channel alerts |
| Analytics | 8013 | Metrics & reporting |
| Search | 8014 | Full-text search |
| Storage | 8015 | File storage (S3) |
| Integration | 8012 | Third-party connectors |

[Full service list →](docs/README.md)

## Tech Stack

- **Language:** Go 1.25
- **Database:** PostgreSQL 15, Redis 7
- **Queue:** Apache Kafka
- **Search:** Elasticsearch 8
- **Gateway:** Kong 3.4
- **Container:** Docker, Kubernetes
- **Monitoring:** Prometheus, Grafana, Jaeger

## Documentation

- [Getting Started](docs/getting-started.md)
- [Deployment Guide](docs/deployment.md)
- [API Reference](docs/api/overview.md)
- [Architecture](docs/architecture/overview.md)

## Development

```bash
# Build all services
make build

# Run tests
make test

# Run linters
make lint

# Database shell
make db-psql
```

## Production

```bash
# Configure production
cp configs/envs/.env.prod.example .env
cp configs/kong/kong.prod.yml configs/kong/kong.yml

# Start (Kong only exposed on 80/443)
make prod
```

See [Deployment Guide](docs/deployment.md) for Kubernetes and Terraform options.

## License

MIT License - see [LICENSE](LICENSE)
