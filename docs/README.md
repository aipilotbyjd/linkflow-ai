# LinkFlow AI Documentation

## Quick Links

| Document | Description |
|----------|-------------|
| [Getting Started](getting-started.md) | Install and run locally |
| [Environments](environments.md) | Dev vs Production setup |
| [Deployment](deployment.md) | Deploy to production |
| [API Reference](api/overview.md) | REST API documentation |
| [Architecture](architecture/overview.md) | System design |

## Documentation Structure

```
docs/
├── getting-started.md      # Installation & quick start
├── deployment.md           # Production deployment
├── api/
│   ├── overview.md         # API introduction
│   ├── authentication.md   # Auth endpoints
│   ├── workflows.md        # Workflow endpoints
│   ├── executions.md       # Execution endpoints
│   └── webhooks.md         # Webhook endpoints
├── architecture/
│   ├── overview.md         # High-level architecture
│   ├── services.md         # Service descriptions
│   └── domain-driven-design.md
├── development/
│   ├── coding-standards.md # Code style guide
│   ├── contributing.md     # Contribution guidelines
│   └── testing.md          # Testing guide
└── reference/
    ├── environment-variables.md
    ├── node-types.md       # Workflow node types
    └── expression-functions.md
```

## Commands Reference

### Development
```bash
make dev          # Start development environment
make stop         # Stop all services
make status       # Show service status
make health       # Health check all services
make logs         # View all logs
```

### Database
```bash
make db-psql      # PostgreSQL shell
make db-migrate   # Run migrations
make db-reset     # Reset database
```

### Building
```bash
make build        # Build Go binaries
make test         # Run tests
make lint         # Run linters
```

## Services

| Service | Port | Description |
|---------|------|-------------|
| Kong | 8000 | API Gateway |
| Auth | 8001 | Authentication |
| User | 8002 | User management |
| Execution | 8003 | Workflow execution |
| Workflow | 8004 | Workflow CRUD |
| Node | 8005 | Node definitions |
| Executor | 8007 | Task execution |
| Webhook | 8008 | Webhook handling |
| Schedule | 8009 | Cron scheduling |
| Credential | 8010 | Secrets management |
| Notification | 8011 | Notifications |
| Integration | 8012 | Third-party integrations |
| Analytics | 8013 | Analytics & metrics |
| Search | 8014 | Full-text search |
| Storage | 8015 | File storage |
| Config | 8016 | Configuration |
| Admin | 8017 | Admin dashboard |
| Tenant | 8019 | Multi-tenancy |
| Monitoring | 8020 | System monitoring |
| Backup | 8021 | Backup service |
| Migration | 8022 | DB migrations |

## External Resources

- [Go Documentation](https://go.dev/doc/)
- [Docker Documentation](https://docs.docker.com/)
- [Kong Gateway](https://docs.konghq.com/)
- [PostgreSQL](https://www.postgresql.org/docs/)
