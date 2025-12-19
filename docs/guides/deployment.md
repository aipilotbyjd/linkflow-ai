# Deployment Guide

Deploy LinkFlow AI to production environments.

## Deployment Options

1. **Docker Compose** - Simple single-server deployment
2. **Kubernetes** - Scalable cloud deployment
3. **Managed Cloud** - AWS, GCP, Azure

## Prerequisites

- Docker and Docker Compose
- PostgreSQL 14+
- Redis 6+ (optional)
- Domain name and SSL certificate

## Docker Compose Deployment

### 1. Prepare Environment

```bash
# Clone repository
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai

# Create production environment file
cp .env.example .env.production
```

### 2. Configure Environment

Edit `.env.production`:

```bash
# Server
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info

# Database
DATABASE_URL=postgres://linkflow:${DB_PASSWORD}@postgres:5432/linkflow?sslmode=disable

# Redis
REDIS_URL=redis://redis:6379/0

# Security (generate strong secrets)
JWT_SECRET=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)

# Rate Limiting
RATE_LIMIT_PER_MIN=100

# CORS
ALLOWED_ORIGINS=https://app.yourdomain.com
```

### 3. Create docker-compose.production.yml

```yaml
version: '3.8'

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  postgres:
    image: postgres:14-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: linkflow
      POSTGRES_USER: linkflow
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    restart: unless-stopped

  redis:
    image: redis:6-alpine
    volumes:
      - redis_data:/data
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./certs:/etc/nginx/certs
    depends_on:
      - api
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### 4. Configure Nginx

Create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream api {
        server api:8080;
    }

    server {
        listen 80;
        server_name api.yourdomain.com;
        return 301 https://$server_name$request_uri;
    }

    server {
        listen 443 ssl http2;
        server_name api.yourdomain.com;

        ssl_certificate /etc/nginx/certs/fullchain.pem;
        ssl_certificate_key /etc/nginx/certs/privkey.pem;

        location / {
            proxy_pass http://api;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

### 5. Deploy

```bash
# Build and start
docker-compose -f docker-compose.production.yml up -d

# Run migrations
docker-compose exec api go run ./cmd/tools/migrate up

# Check logs
docker-compose logs -f api
```

## Kubernetes Deployment

### 1. Create Namespace

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: linkflow
```

### 2. Create Secrets

```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: linkflow-secrets
  namespace: linkflow
type: Opaque
stringData:
  database-url: "postgres://user:pass@postgres:5432/linkflow"
  jwt-secret: "your-jwt-secret"
  encryption-key: "your-encryption-key"
```

### 3. Create ConfigMap

```yaml
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: linkflow-config
  namespace: linkflow
data:
  PORT: "8080"
  ENVIRONMENT: "production"
  LOG_LEVEL: "info"
  RATE_LIMIT_PER_MIN: "100"
```

### 4. Create Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linkflow-api
  namespace: linkflow
spec:
  replicas: 3
  selector:
    matchLabels:
      app: linkflow-api
  template:
    metadata:
      labels:
        app: linkflow-api
    spec:
      containers:
        - name: api
          image: linkflow/api:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: linkflow-config
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: linkflow-secrets
                  key: database-url
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: linkflow-secrets
                  key: jwt-secret
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
```

### 5. Create Service

```yaml
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: linkflow-api
  namespace: linkflow
spec:
  selector:
    app: linkflow-api
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

### 6. Create Ingress

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: linkflow-ingress
  namespace: linkflow
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - api.yourdomain.com
      secretName: linkflow-tls
  rules:
    - host: api.yourdomain.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: linkflow-api
                port:
                  number: 80
```

### 7. Deploy to Kubernetes

```bash
kubectl apply -f k8s/
```

## Database Migrations

### Run Migrations

```bash
# Docker Compose
docker-compose exec api go run ./cmd/tools/migrate up

# Kubernetes
kubectl exec -it deployment/linkflow-api -n linkflow -- go run ./cmd/tools/migrate up
```

### Rollback Migrations

```bash
go run ./cmd/tools/migrate down 1
```

## Health Checks

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Basic health check |
| `/health/ready` | Readiness check (DB, Redis) |
| `/health/live` | Liveness check |

### Monitoring

```bash
# Check health
curl https://api.yourdomain.com/health

# Expected response
{"status":"healthy","timestamp":"2024-12-19T12:00:00Z"}
```

## Scaling

### Horizontal Scaling

```bash
# Docker Compose
docker-compose up -d --scale api=3

# Kubernetes
kubectl scale deployment linkflow-api -n linkflow --replicas=5
```

### Vertical Scaling

Update resource limits in deployment:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Backup & Recovery

### Database Backup

```bash
# Backup
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d).sql

# Restore
psql $DATABASE_URL < backup_20241219.sql
```

### Automated Backups

```yaml
# k8s/cronjob-backup.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: db-backup
  namespace: linkflow
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: backup
              image: postgres:14
              command:
                - /bin/sh
                - -c
                - pg_dump $DATABASE_URL | gzip > /backups/backup_$(date +%Y%m%d).sql.gz
          restartPolicy: OnFailure
```

## Monitoring

### Prometheus Metrics

```yaml
# k8s/servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: linkflow-api
  namespace: linkflow
spec:
  selector:
    matchLabels:
      app: linkflow-api
  endpoints:
    - port: metrics
      interval: 30s
```

### Key Metrics

| Metric | Description |
|--------|-------------|
| `http_requests_total` | Total HTTP requests |
| `http_request_duration_seconds` | Request latency |
| `workflow_executions_total` | Total executions |
| `workflow_execution_duration_seconds` | Execution time |

## Security Checklist

- [ ] Use HTTPS with valid SSL certificate
- [ ] Set strong JWT secret (32+ characters)
- [ ] Enable rate limiting
- [ ] Configure CORS properly
- [ ] Use database SSL mode in production
- [ ] Rotate secrets regularly
- [ ] Enable audit logging
- [ ] Set up monitoring and alerting

## Troubleshooting

### Common Issues

**Connection refused to database**
```bash
# Check PostgreSQL is running
docker-compose ps postgres
# Check connection string
docker-compose exec api env | grep DATABASE_URL
```

**High memory usage**
```bash
# Check container stats
docker stats
# Adjust worker pool size
WORKER_POOL_SIZE=5
```

**Slow responses**
```bash
# Check database performance
docker-compose exec postgres psql -U linkflow -c "SELECT * FROM pg_stat_activity;"
```

## Next Steps

- [Configuration Guide](../getting-started/configuration.md)
- [Monitoring Setup](monitoring.md)
- [Security Best Practices](security.md)
