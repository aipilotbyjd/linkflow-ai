# Configuration Guide

This guide covers all configuration options for LinkFlow AI.

## Configuration Methods

Configuration can be provided via:
1. Environment variables (recommended for production)
2. `.env` file (recommended for development)
3. YAML configuration files in `configs/`

## Environment Variables

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | `8080` | No |
| `ENVIRONMENT` | Environment name (development/staging/production) | `development` | No |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | `info` | No |

### Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `DB_MAX_CONNECTIONS` | Maximum database connections | `25` | No |
| `DB_MAX_IDLE_CONNECTIONS` | Maximum idle connections | `5` | No |
| `DB_CONNECTION_MAX_LIFETIME` | Connection max lifetime | `5m` | No |

### Redis Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_URL` | Redis connection string | - | No |
| `REDIS_PASSWORD` | Redis password | - | No |
| `REDIS_DB` | Redis database number | `0` | No |

### Security Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | Secret key for JWT signing | - | Yes |
| `JWT_EXPIRY` | JWT token expiry duration | `24h` | No |
| `ENCRYPTION_KEY` | Key for credential encryption | - | Yes (prod) |

### Rate Limiting

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RATE_LIMIT_PER_MIN` | Requests per minute per IP | `100` | No |
| `RATE_LIMIT_BURST` | Burst size for rate limiting | `200` | No |

### CORS Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ALLOWED_ORIGINS` | Comma-separated allowed origins | `*` | No |
| `ALLOWED_METHODS` | Allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS` | No |

### Billing (Stripe)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STRIPE_SECRET_KEY` | Stripe API secret key | - | No |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret | - | No |
| `STRIPE_PRICE_ID_PRO` | Price ID for Pro plan | - | No |
| `STRIPE_PRICE_ID_BUSINESS` | Price ID for Business plan | - | No |

### Email (SMTP)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SMTP_HOST` | SMTP server hostname | - | No |
| `SMTP_PORT` | SMTP server port | `587` | No |
| `SMTP_USERNAME` | SMTP username | - | No |
| `SMTP_PASSWORD` | SMTP password | - | No |
| `SMTP_FROM` | Default from email address | - | No |

### Email (SendGrid)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SENDGRID_API_KEY` | SendGrid API key | - | No |
| `SENDGRID_FROM` | Default from email address | - | No |

### Observability

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JAEGER_ENDPOINT` | Jaeger collector endpoint | - | No |
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` | No |
| `TRACING_ENABLED` | Enable distributed tracing | `false` | No |

### Execution Engine

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `WORKER_POOL_SIZE` | Number of worker goroutines | `10` | No |
| `EXECUTION_TIMEOUT` | Default execution timeout | `30m` | No |
| `MAX_RETRIES` | Maximum retry attempts | `3` | No |
| `RETRY_DELAY` | Initial retry delay | `1s` | No |

## YAML Configuration

For complex configurations, use YAML files in `configs/`:

### configs/app.yaml

```yaml
server:
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

database:
  url: postgres://localhost:5432/linkflow
  max_connections: 25
  max_idle: 5

redis:
  url: redis://localhost:6379
  db: 0

security:
  jwt_secret: ${JWT_SECRET}
  jwt_expiry: 24h
  encryption_key: ${ENCRYPTION_KEY}

rate_limit:
  requests_per_minute: 100
  burst_size: 200

cors:
  allowed_origins:
    - http://localhost:3000
    - https://app.linkflow.ai
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  allow_credentials: true
  max_age: 86400
```

### configs/integrations.yaml

```yaml
integrations:
  slack:
    client_id: ${SLACK_CLIENT_ID}
    client_secret: ${SLACK_CLIENT_SECRET}
    
  github:
    client_id: ${GITHUB_CLIENT_ID}
    client_secret: ${GITHUB_CLIENT_SECRET}
    
  google:
    client_id: ${GOOGLE_CLIENT_ID}
    client_secret: ${GOOGLE_CLIENT_SECRET}
```

## Environment-Specific Configuration

### Development

```bash
# .env.development
ENVIRONMENT=development
LOG_LEVEL=debug
DATABASE_URL=postgres://localhost:5432/linkflow_dev
JWT_SECRET=dev-secret-not-for-production
```

### Staging

```bash
# .env.staging
ENVIRONMENT=staging
LOG_LEVEL=info
DATABASE_URL=postgres://staging-db:5432/linkflow
JWT_SECRET=${JWT_SECRET}
TRACING_ENABLED=true
```

### Production

```bash
# .env.production
ENVIRONMENT=production
LOG_LEVEL=warn
DATABASE_URL=${DATABASE_URL}
JWT_SECRET=${JWT_SECRET}
ENCRYPTION_KEY=${ENCRYPTION_KEY}
TRACING_ENABLED=true
METRICS_ENABLED=true
```

## Secrets Management

For production, use a secrets manager:

### AWS Secrets Manager

```yaml
# configs/secrets.yaml
secrets:
  provider: aws
  region: us-east-1
  secrets:
    - name: linkflow/production
      keys:
        - DATABASE_URL
        - JWT_SECRET
        - ENCRYPTION_KEY
```

### HashiCorp Vault

```yaml
# configs/secrets.yaml
secrets:
  provider: vault
  address: https://vault.example.com
  path: secret/data/linkflow
```

## Validation

The application validates configuration on startup. Missing required values will cause startup to fail with a descriptive error message.

To validate configuration without starting:

```bash
go run ./cmd/tools/validate-config
```

## Next Steps

- [Installation Guide](installation.md)
- [Deployment Guide](../guides/deployment.md)
- [Environment Variables Reference](../reference/environment-variables.md)
