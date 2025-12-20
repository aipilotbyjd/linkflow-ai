# Deployment Guide

Deploy LinkFlow AI to development, staging, or production environments.

## Environments

| Environment | Command | Exposed Ports | Use Case |
|-------------|---------|---------------|----------|
| Development | `make dev` | All services | Local development |
| Production | `make prod` | Kong only (80/443) | Production deployment |

## Development Deployment

```bash
# Start with all ports exposed for debugging
make dev

# Check status
make status
make health

# View logs
make logs
```

All 32 services accessible on their individual ports (8000-8022, 9090, 3000, etc.)

## Production Deployment

### Docker Compose

```bash
# 1. Configure production environment
cp configs/envs/.env.prod.example .env
# Edit .env with production values

# 2. Use production Kong config
cp configs/kong/kong.prod.yml configs/kong/kong.yml

# 3. Start production
make prod
```

**Production features:**
- Only Kong exposed on ports 80 (HTTP) and 443 (HTTPS)
- Admin tools (Adminer, Kafka UI, Jaeger) disabled
- Strict CORS (only your domains)
- Rate limiting backed by Redis
- IP restrictions on admin endpoints

### Kubernetes

```bash
# Apply production overlay
kubectl apply -k deployments/kubernetes/overlays/production/

# Or use Helm
helm install linkflow deployments/helm/linkflow \
  --namespace linkflow \
  --create-namespace \
  -f values-production.yaml
```

### Terraform (AWS/GCP/Azure)

```bash
cd deployments/terraform
terraform init
terraform plan -var-file=production.tfvars
terraform apply -var-file=production.tfvars
```

## Architecture

### Network Tiers

```
Internet
    │
    ▼
┌─────────────────────────────┐
│   EDGE NETWORK              │
│   Kong (80/443)             │
│   Grafana (3000) - optional │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│   APP NETWORK               │
│   21 Microservices          │
│   (internal only)           │
└─────────────────────────────┘
    │
    ▼
┌─────────────────────────────┐
│   DATA NETWORK              │
│   PostgreSQL, Redis         │
│   Kafka, Elasticsearch      │
│   (internal only)           │
└─────────────────────────────┘
```

### Services

| Layer | Services | Ports |
|-------|----------|-------|
| Edge | Kong | 80, 443 |
| App | auth, user, workflow, execution, node, executor, webhook, schedule, credential, notification, integration, analytics, search, storage, config, admin, tenant, monitoring, backup, migration, gateway | 8001-8022 |
| Data | PostgreSQL, Redis, Kafka, Zookeeper, Elasticsearch | 5432, 6379, 9092, 2181, 9200 |
| Monitor | Prometheus, Grafana, Jaeger | 9090, 3000, 16686 |

## Configuration

### Kong API Gateway

Three configurations available:

| Config | File | Description |
|--------|------|-------------|
| Base | `configs/kong/kong.yml` | Default configuration |
| Development | `configs/kong/kong.dev.yml` | Relaxed rate limits |
| Production | `configs/kong/kong.prod.yml` | Strict security |

Production Kong features:
- Rate limiting with Redis backend (distributed)
- Strict CORS (whitelist your domains)
- IP restrictions on admin/config/monitoring endpoints
- Security headers (HSTS, X-Frame-Options, etc.)
- Request correlation IDs

### Environment Variables

Development: `configs/envs/.env.dev`
Production template: `configs/envs/.env.prod.example`

Critical production variables:
```bash
# Strong secrets (generate random strings)
JWT_SECRET=<64-char-random-string>
ENCRYPTION_KEY=<32-byte-key>

# Managed databases
DB_HOST=your-rds-endpoint.amazonaws.com
DB_SSL_MODE=require
REDIS_HOST=your-elasticache-endpoint

# Restricted CORS
ALLOWED_ORIGINS=https://app.yourdomain.com

# Monitoring
SENTRY_DSN=https://xxx@sentry.io/xxx
```

## Health Checks

All services expose health endpoints:

```bash
# Liveness (is the service running?)
GET /health/live

# Readiness (can the service handle requests?)
GET /health/ready

# Check all services
make health
```

## Monitoring

### Prometheus Metrics

All services expose metrics at `/metrics`:
- HTTP request duration
- Request count by status
- Active connections
- Custom business metrics

### Grafana Dashboards

Pre-configured dashboards in `configs/grafana/dashboards/`:
- Service overview
- Request latency
- Error rates
- Resource usage

### Distributed Tracing

Jaeger integration for request tracing across services.

Access: http://localhost:16686 (dev only)

## Security Checklist

Before going to production:

- [ ] Change all default passwords
- [ ] Generate strong JWT_SECRET (64+ chars)
- [ ] Generate strong ENCRYPTION_KEY (32 bytes)
- [ ] Configure SSL/TLS certificates
- [ ] Set strict CORS origins
- [ ] Enable rate limiting
- [ ] Configure IP restrictions for admin endpoints
- [ ] Set up secrets management (Vault, AWS Secrets Manager)
- [ ] Enable audit logging
- [ ] Configure backup schedule
- [ ] Set up monitoring alerts

## Scaling

### Horizontal Scaling

```yaml
# Kubernetes HPA example
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: workflow-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: workflow
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### Resource Limits

All services have default limits in `base.yml`:
- Memory: 256MB - 1GB depending on service
- CPU: 0.25 - 1 core depending on service

Adjust in production based on load testing.
