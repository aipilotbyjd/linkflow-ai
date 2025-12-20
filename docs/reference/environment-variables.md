# Environment Variables Reference

Complete reference of all environment variables used by LinkFlow AI.

## Quick Reference

```bash
# Required
DATABASE_URL=postgres://user:pass@localhost:5432/linkflow
JWT_SECRET=your-secure-secret-key

# Server
PORT=8080
ENVIRONMENT=development

# Optional
REDIS_URL=redis://localhost:6379
ENCRYPTION_KEY=base64-encoded-32-byte-key
```

## Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | `8080` | No |
| `HOST` | Server bind address | `0.0.0.0` | No |
| `ENVIRONMENT` | Environment name | `development` | No |
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |
| `LOG_FORMAT` | Log format (json/text) | `json` | No |

### Example
```bash
PORT=3000
HOST=127.0.0.1
ENVIRONMENT=production
LOG_LEVEL=warn
LOG_FORMAT=json
```

## Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | **Yes** |
| `DB_MAX_CONNECTIONS` | Maximum open connections | `25` | No |
| `DB_MAX_IDLE_CONNECTIONS` | Maximum idle connections | `5` | No |
| `DB_CONNECTION_MAX_LIFETIME` | Connection max lifetime | `5m` | No |
| `DB_SSL_MODE` | SSL mode (disable/require/verify-full) | `disable` | No |

### Connection String Format
```
postgres://username:password@host:port/database?sslmode=disable
```

### Example
```bash
DATABASE_URL=postgres://linkflow:secret@localhost:5432/linkflow?sslmode=disable
DB_MAX_CONNECTIONS=50
DB_MAX_IDLE_CONNECTIONS=10
DB_CONNECTION_MAX_LIFETIME=10m
```

## Redis Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `REDIS_URL` | Redis connection string | - | No |
| `REDIS_HOST` | Redis host (if not using URL) | `localhost` | No |
| `REDIS_PORT` | Redis port | `6379` | No |
| `REDIS_PASSWORD` | Redis password | - | No |
| `REDIS_DB` | Redis database number | `0` | No |
| `REDIS_TLS_ENABLED` | Enable TLS | `false` | No |

### Example
```bash
REDIS_URL=redis://:password@localhost:6379/0
# Or individual settings:
REDIS_HOST=redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=secret
REDIS_DB=0
```

## Security Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | JWT signing secret (min 32 chars) | - | **Yes** |
| `JWT_EXPIRY` | JWT token expiry | `24h` | No |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry | `168h` | No |
| `ENCRYPTION_KEY` | Data encryption key (base64) | - | Production |
| `BCRYPT_COST` | Password hashing cost | `10` | No |

### Example
```bash
JWT_SECRET=your-super-secure-jwt-secret-key-min-32-chars
JWT_EXPIRY=12h
JWT_REFRESH_EXPIRY=720h
ENCRYPTION_KEY=YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY=
BCRYPT_COST=12
```

## Rate Limiting

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `RATE_LIMIT_PER_MIN` | Requests per minute per IP | `100` | No |
| `RATE_LIMIT_BURST` | Burst size | `200` | No |
| `RATE_LIMIT_ENABLED` | Enable rate limiting | `true` | No |

### Example
```bash
RATE_LIMIT_PER_MIN=60
RATE_LIMIT_BURST=120
RATE_LIMIT_ENABLED=true
```

## CORS Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ALLOWED_ORIGINS` | Comma-separated allowed origins | `*` | No |
| `ALLOWED_METHODS` | Comma-separated allowed methods | `GET,POST,PUT,DELETE,OPTIONS` | No |
| `ALLOWED_HEADERS` | Comma-separated allowed headers | `*` | No |
| `CORS_MAX_AGE` | Preflight cache duration (seconds) | `86400` | No |

### Example
```bash
ALLOWED_ORIGINS=https://app.linkflow.ai,https://admin.linkflow.ai
ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
ALLOWED_HEADERS=Authorization,Content-Type,X-Request-ID
CORS_MAX_AGE=86400
```

## Stripe Billing

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STRIPE_SECRET_KEY` | Stripe API secret key | - | For billing |
| `STRIPE_PUBLISHABLE_KEY` | Stripe publishable key | - | For billing |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret | - | For billing |
| `STRIPE_PRICE_ID_PRO` | Pro plan price ID | - | For billing |
| `STRIPE_PRICE_ID_BUSINESS` | Business plan price ID | - | For billing |
| `STRIPE_PRICE_ID_ENTERPRISE` | Enterprise plan price ID | - | For billing |

### Example
```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_PUBLISHABLE_KEY=pk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_ID_PRO=price_...
STRIPE_PRICE_ID_BUSINESS=price_...
```

## Email (SMTP)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SMTP_HOST` | SMTP server hostname | - | For email |
| `SMTP_PORT` | SMTP server port | `587` | No |
| `SMTP_USERNAME` | SMTP username | - | For email |
| `SMTP_PASSWORD` | SMTP password | - | For email |
| `SMTP_FROM` | Default from address | - | For email |
| `SMTP_FROM_NAME` | Default from name | `LinkFlow AI` | No |
| `SMTP_TLS_ENABLED` | Enable TLS | `true` | No |

### Example
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=notifications@example.com
SMTP_PASSWORD=app-specific-password
SMTP_FROM=notifications@example.com
SMTP_FROM_NAME=LinkFlow AI
```

## Email (SendGrid)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SENDGRID_API_KEY` | SendGrid API key | - | For SendGrid |
| `SENDGRID_FROM` | Default from address | - | For SendGrid |
| `SENDGRID_FROM_NAME` | Default from name | `LinkFlow AI` | No |

### Example
```bash
SENDGRID_API_KEY=SG.xxx...
SENDGRID_FROM=notifications@linkflow.ai
SENDGRID_FROM_NAME=LinkFlow AI
```

## OAuth Providers

### Google
| Variable | Description |
|----------|-------------|
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `GOOGLE_REDIRECT_URI` | OAuth callback URL |

### GitHub
| Variable | Description |
|----------|-------------|
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth client secret |
| `GITHUB_REDIRECT_URI` | OAuth callback URL |

### Microsoft
| Variable | Description |
|----------|-------------|
| `MICROSOFT_CLIENT_ID` | Microsoft OAuth client ID |
| `MICROSOFT_CLIENT_SECRET` | Microsoft OAuth client secret |
| `MICROSOFT_REDIRECT_URI` | OAuth callback URL |

## Observability

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` | No |
| `METRICS_PORT` | Metrics server port | `9090` | No |
| `TRACING_ENABLED` | Enable distributed tracing | `false` | No |
| `JAEGER_ENDPOINT` | Jaeger collector endpoint | - | For tracing |
| `JAEGER_SERVICE_NAME` | Service name for tracing | `linkflow-ai` | No |

### Example
```bash
METRICS_ENABLED=true
METRICS_PORT=9090
TRACING_ENABLED=true
JAEGER_ENDPOINT=http://jaeger:14268/api/traces
JAEGER_SERVICE_NAME=linkflow-api
```

## Execution Engine

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `WORKER_POOL_SIZE` | Number of worker goroutines | `10` | No |
| `EXECUTION_TIMEOUT` | Default execution timeout | `30m` | No |
| `MAX_CONCURRENT_EXECUTIONS` | Max concurrent executions | `100` | No |
| `QUEUE_SIZE` | Task queue size | `1000` | No |
| `MAX_RETRIES` | Default retry attempts | `3` | No |
| `RETRY_DELAY` | Initial retry delay | `1s` | No |
| `RETRY_MAX_DELAY` | Maximum retry delay | `5m` | No |

### Example
```bash
WORKER_POOL_SIZE=20
EXECUTION_TIMEOUT=1h
MAX_CONCURRENT_EXECUTIONS=200
QUEUE_SIZE=5000
MAX_RETRIES=5
RETRY_DELAY=2s
```

## Storage (S3/GCS)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STORAGE_PROVIDER` | Storage provider (s3/gcs/local) | `local` | No |
| `AWS_ACCESS_KEY_ID` | AWS access key | - | For S3 |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | - | For S3 |
| `AWS_REGION` | AWS region | `us-east-1` | For S3 |
| `S3_BUCKET` | S3 bucket name | - | For S3 |
| `GCS_PROJECT_ID` | GCS project ID | - | For GCS |
| `GCS_BUCKET` | GCS bucket name | - | For GCS |

## Message Queue (Kafka)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `KAFKA_BROKERS` | Comma-separated broker addresses | - | For Kafka |
| `KAFKA_TOPIC_PREFIX` | Topic name prefix | `linkflow` | No |
| `KAFKA_CONSUMER_GROUP` | Consumer group ID | `linkflow-consumers` | No |

### Example
```bash
KAFKA_BROKERS=kafka1:9092,kafka2:9092
KAFKA_TOPIC_PREFIX=prod-linkflow
KAFKA_CONSUMER_GROUP=linkflow-workers
```

## Feature Flags

| Variable | Description | Default |
|----------|-------------|---------|
| `FEATURE_BILLING_ENABLED` | Enable billing features | `true` |
| `FEATURE_OAUTH_ENABLED` | Enable OAuth login | `true` |
| `FEATURE_WEBHOOKS_ENABLED` | Enable webhooks | `true` |
| `FEATURE_TEMPLATES_ENABLED` | Enable templates | `true` |
| `FEATURE_SHARING_ENABLED` | Enable sharing | `true` |

## Development Only

| Variable | Description | Default |
|----------|-------------|---------|
| `DEBUG` | Enable debug mode | `false` |
| `PRETTY_LOGS` | Pretty print logs | `false` |
| `SKIP_AUTH` | Skip authentication (dangerous!) | `false` |
| `MOCK_INTEGRATIONS` | Mock external integrations | `false` |

## Complete Example

```bash
# .env.production

# Server
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
LOG_FORMAT=json

# Database
DATABASE_URL=postgres://linkflow:${DB_PASSWORD}@db.example.com:5432/linkflow?sslmode=require
DB_MAX_CONNECTIONS=50

# Redis
REDIS_URL=redis://:${REDIS_PASSWORD}@redis.example.com:6379/0

# Security
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRY=12h
ENCRYPTION_KEY=${ENCRYPTION_KEY}

# Rate Limiting
RATE_LIMIT_PER_MIN=100
RATE_LIMIT_BURST=200

# CORS
ALLOWED_ORIGINS=https://app.linkflow.ai

# Stripe
STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}

# Email
SENDGRID_API_KEY=${SENDGRID_API_KEY}
SENDGRID_FROM=noreply@linkflow.ai

# Observability
METRICS_ENABLED=true
TRACING_ENABLED=true
JAEGER_ENDPOINT=http://jaeger:14268/api/traces

# Engine
WORKER_POOL_SIZE=20
EXECUTION_TIMEOUT=1h
```

## Next Steps

- [Getting Started](../getting-started.md)
- [Deployment Guide](../deployment.md)
- [Architecture Overview](../architecture/overview.md)
