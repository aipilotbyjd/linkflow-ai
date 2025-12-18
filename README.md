# LinkFlow AI - Microservices Workflow Automation Platform

A production-ready, cloud-native workflow automation platform built with Go microservices architecture.

## 🚀 Features

- **Microservices Architecture**: 18+ specialized services following Domain-Driven Design
- **Event-Driven**: Kafka-based event streaming with CQRS and Event Sourcing
- **Cloud-Native**: Kubernetes-ready with service mesh (Istio) support
- **Scalable**: Horizontal scaling, caching layers, and optimized database queries
- **Observable**: Distributed tracing (Jaeger), metrics (Prometheus/Grafana), structured logging
- **Secure**: JWT authentication, RBAC, mTLS, secrets management
- **Developer-Friendly**: Hot reload, comprehensive testing, CI/CD pipelines

## 🏗 Architecture

```
┌─────────────────────────────────────────┐
│            API Gateway (Kong)            │
└─────────────────────────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│  Auth   │    │Workflow │    │Execution│
│ Service │    │ Service │    │ Service │
└─────────┘    └─────────┘    └─────────┘
    │                │                │
    └────────────────┼────────────────┘
                     ▼
         ┌──────────────────────┐
         │   Event Bus (Kafka)   │
         └──────────────────────┘
                     │
    ┌────────────────┼────────────────┐
    ▼                ▼                ▼
┌─────────┐    ┌─────────┐    ┌─────────┐
│PostgreSQL│    │  Redis  │    │Elastic │
└─────────┘    └─────────┘    └─────────┘
```

## 🛠 Technology Stack

- **Language**: Go 1.21+
- **Databases**: PostgreSQL 15+, Redis 7+
- **Message Queue**: Apache Kafka
- **Search**: Elasticsearch 8+
- **API Gateway**: Kong
- **Service Mesh**: Istio
- **Monitoring**: Prometheus, Grafana, Jaeger
- **Container**: Docker, Kubernetes
- **CI/CD**: GitLab CI / GitHub Actions

## 📦 Services

| Service | Port | Description |
|---------|------|-------------|
| Auth Service | 8001 | Authentication, JWT, OAuth2 |
| User Service | 8002 | User profiles, organizations |
| Workflow Service | 8004 | Workflow CRUD, versioning |
| Execution Service | 8005 | Workflow orchestration |
| Node Service | 8006 | Node registry, marketplace |
| Webhook Service | 8008 | Webhook management |
| Schedule Service | 8009 | Cron scheduling |
| Notification Service | 8011 | Email, SMS, push notifications |
| Analytics Service | 8013 | Usage analytics, reporting |

## 🚀 Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Make

### Installation

1. Clone the repository:
```bash
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai
```

2. Install development tools:
```bash
make install-tools
```

3. Start infrastructure:
```bash
make docker-up
```

4. Run database migrations:
```bash
make migrate
```

5. Start services:
```bash
make dev
```

The platform will be available at:
- API Gateway: http://localhost:8000
- Grafana: http://localhost:3000 (admin/admin)
- Jaeger: http://localhost:16686
- Kafka UI: http://localhost:8090

## 🧪 Testing

Run all tests:
```bash
make test
```

Run with coverage:
```bash
make test-coverage
```

Run specific service tests:
```bash
go test -v ./internal/workflow/...
```

## 📚 Documentation

- [Architecture Overview](docs/ARCHITECTURE.md)
- [Implementation Guide](docs/IMPLEMENTATION_GUIDE.md)
- [Migration Guide](docs/MIGRATION_GUIDE.md)
- [API Documentation](http://localhost:8000/docs)

## 🔧 Development

### Project Structure

```
linkflow-ai/
├── cmd/                    # Service entry points
│   └── services/          # Individual services
├── internal/              # Private application code
│   ├── [service]/        # Service-specific code
│   ├── platform/         # Shared platform code
│   └── shared/           # Shared business logic
├── pkg/                   # Public packages
├── api/                   # API definitions
├── deployments/          # Deployment configurations
├── migrations/           # Database migrations
├── configs/              # Configuration files
├── scripts/              # Utility scripts
└── tests/                # Test suites
```

### Available Commands

```bash
make help              # Show all available commands
make dev              # Start development environment
make build            # Build all services
make test             # Run tests
make lint             # Run linters
make docker-build     # Build Docker images
make k8s-deploy       # Deploy to Kubernetes
```

## 🚢 Deployment

### Docker Compose (Development)

```bash
docker-compose up -d
```

### Kubernetes (Production)

```bash
# Apply Kubernetes manifests
kubectl apply -k deployments/kubernetes/overlays/production/

# Or use Helm
helm install linkflow deployments/helm/linkflow
```

## 📊 Monitoring

- **Metrics**: Prometheus metrics available at `/metrics`
- **Tracing**: Distributed tracing with Jaeger
- **Logging**: Structured JSON logging
- **Health Checks**: `/health/live` and `/health/ready`

## 🔒 Security

- JWT-based authentication
- Role-based access control (RBAC)
- TLS/mTLS communication
- Secrets management with HashiCorp Vault
- Rate limiting and DDoS protection
- Input validation and sanitization

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Team

- Architecture & Backend: LinkFlow AI Team
- DevOps & Infrastructure: LinkFlow AI Team

## 📞 Support

- Documentation: [https://docs.linkflow.ai](https://docs.linkflow.ai)
- Issues: [GitHub Issues](https://github.com/linkflow-ai/linkflow-ai/issues)
- Discord: [Join our community](https://discord.gg/linkflow)

---

Built with ❤️ by the LinkFlow AI Team
