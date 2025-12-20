# Environment Guide

How to run LinkFlow AI in different environments.

## Quick Reference

| Environment | Command | Ports Exposed | Use Case |
|-------------|---------|---------------|----------|
| Development | `make dev` | All (8000-9200) | Local coding & debugging |
| Production | `make prod` | Kong only (80/443) | Live deployment |

---

## Development Environment

For local development with all services accessible for debugging.

### Start Development

```bash
# Option 1: Using make (recommended)
make dev

# Option 2: Using script directly
./scripts/linkflow.sh dev

# Option 3: Run in background
make dev
# or
./scripts/linkflow.sh -d dev
```

### What Happens

1. Uses `.env` file from project root
2. Uses `configs/kong/kong.yml` (base config)
3. Exposes ALL service ports:
   - Kong: 8000, 8001
   - All microservices: 8001-8022
   - PostgreSQL: 5432
   - Redis: 6379
   - Kafka: 9092
   - Elasticsearch: 9200
   - Grafana: 3000
   - Prometheus: 9090
   - Jaeger: 16686
   - Adminer: 8089
   - Kafka UI: 8088

### Development Workflow

```bash
# Start services
make dev

# Check status
make status

# View logs (all)
make logs

# View specific service logs
make logs-workflow
make logs-auth

# Health check
make health

# Access database
make db-psql

# Stop when done
make stop
```

### Environment File

Uses `.env` in project root. Copy from example if needed:

```bash
cp .env.example .env
# or use dev defaults:
cp configs/envs/.env.dev .env
```

---

## Production Environment

For live deployment with security hardening.

### Start Production

```bash
# Option 1: Using make
make prod

# Option 2: Using script
./scripts/linkflow.sh -d prod
```

### What Happens

1. Uses `.env` file (must have production secrets)
2. Uses `configs/kong/kong.prod.yml` (strict security)
3. Exposes ONLY:
   - Kong: 80 (HTTP), 443 (HTTPS)
   - Grafana: 3000 (optional, for monitoring)
4. Disables admin tools (Adminer, Kafka UI, Jaeger)
5. Enables strict rate limiting
6. Restricts CORS to your domains

### Production Setup

```bash
# 1. Create production environment file
cp configs/envs/.env.prod.example .env

# 2. Edit with real values
vim .env
# Set: DB credentials, JWT_SECRET, ENCRYPTION_KEY, etc.

# 3. Use production Kong config
cp configs/kong/kong.prod.yml configs/kong/kong.yml

# 4. Start production
make prod
```

### Production Checklist

Before going live:

- [ ] Set strong `JWT_SECRET` (64+ characters)
- [ ] Set strong `ENCRYPTION_KEY` (32 bytes)
- [ ] Configure real database credentials
- [ ] Configure real Redis credentials
- [ ] Set `ALLOWED_ORIGINS` to your domains only
- [ ] Configure SSL/TLS certificates
- [ ] Set up monitoring alerts
- [ ] Configure backup schedule

---

## File Structure

```
linkflow-ai/
├── .env                          # Active environment (gitignored)
├── .env.example                  # Template for any environment
│
├── configs/
│   ├── envs/
│   │   ├── .env.example          # Full template with all variables
│   │   ├── .env.dev              # Development defaults
│   │   └── .env.prod.example     # Production template
│   │
│   └── kong/
│       ├── kong.yml              # Active Kong config
│       ├── kong.dev.yml          # Development (relaxed)
│       └── kong.prod.yml         # Production (strict)
│
└── deployments/docker/compose/
    ├── base.yml                  # All service definitions
    ├── dev.yml                   # Dev overrides (all ports)
    └── prod.yml                  # Prod overrides (Kong only)
```

---

## Environment Variables

### Required (All Environments)

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `postgres` or `db.example.com` |
| `DB_PASSWORD` | Database password | `postgres` / `<strong-password>` |
| `JWT_SECRET` | JWT signing key (32+ chars) | `your-secret-key...` |
| `REDIS_HOST` | Redis host | `redis` or `redis.example.com` |

### Development Defaults

```bash
ENVIRONMENT=development
LOG_LEVEL=debug
DB_HOST=postgres
DB_PASSWORD=postgres
JWT_SECRET=linkflow-dev-secret-key-change-in-production-min-32-chars
ALLOWED_ORIGINS=*
RATE_LIMIT_PER_MINUTE=10000
```

### Production Requirements

```bash
ENVIRONMENT=production
LOG_LEVEL=info
DB_HOST=your-rds-endpoint.amazonaws.com
DB_PASSWORD=<strong-random-password>
DB_SSL_MODE=require
JWT_SECRET=<64-char-random-string>
ENCRYPTION_KEY=<32-byte-key>
ALLOWED_ORIGINS=https://app.yourdomain.com
RATE_LIMIT_PER_MINUTE=100
```

---

## Kong Configuration

### Development (`kong.dev.yml`)

- Relaxed rate limits (10,000/min)
- No IP restrictions
- Permissive CORS (`*`)
- All routes open

### Production (`kong.prod.yml`)

- Strict rate limits (100/min per service)
- Rate limiting backed by Redis
- IP restrictions on admin endpoints
- Strict CORS (your domains only)
- Security headers enabled

### Switching Kong Config

```bash
# For development
cp configs/kong/kong.dev.yml configs/kong/kong.yml

# For production
cp configs/kong/kong.prod.yml configs/kong/kong.yml

# Restart to apply
make restart
```

---

## Common Commands

```bash
# Development
make dev              # Start development
make stop             # Stop all
make restart          # Restart all
make status           # Check status
make health           # Health check
make logs             # All logs
make logs-<service>   # Service logs

# Database
make db-psql          # PostgreSQL shell
make db-migrate       # Run migrations
make db-reset         # Reset database

# Debugging
make shell-postgres   # Shell into postgres
make shell-redis      # Redis CLI
make shell-auth       # Shell into auth service
```

---

## Troubleshooting

### Services not starting

```bash
# Check Docker
docker info

# Check logs
make logs

# Check specific service
make logs-auth
```

### Wrong environment

```bash
# Verify which .env is being used
cat .env | head -20

# Reset to development
cp configs/envs/.env.dev .env
make restart
```

### Kong routing issues

```bash
# Check Kong config
cat configs/kong/kong.yml | head -50

# Reset to development config
cp configs/kong/kong.dev.yml configs/kong/kong.yml
make restart
```

### Port conflicts

```bash
# Check what's using a port
lsof -i :8000

# Stop conflicting process or change port in .env
```
