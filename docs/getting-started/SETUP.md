# LinkFlow AI - Complete Setup Guide

Step-by-step guide to set up and run LinkFlow AI from scratch.

---

## Step 1: Install Prerequisites

### 1.1 Install Homebrew (if not installed)
```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 1.2 Install Go
```bash
brew install go
```

### 1.3 Install Docker Desktop
Download and install from: https://www.docker.com/products/docker-desktop

After installation, start Docker Desktop from Applications.

### 1.4 Verify Installations
```bash
go version          # Expected: go1.25 or higher
docker --version    # Expected: Docker version 20+
docker compose version  # Expected: v2.0+
```

---

## Step 2: Clone and Configure

### 2.1 Clone Repository
```bash
git clone <your-repo-url>
cd linkflow-ai
```

### 2.2 Create Environment File
```bash
cp .env.example .env
```

### 2.3 Install Go Dependencies
```bash
go mod download
```

### 2.4 Install Development Tools
```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install all dev tools (optional)
make install-tools
```

### 2.5 Verify migrate is installed
```bash
migrate -version
```

If `migrate: command not found`, add Go bin to PATH:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
# Add this to ~/.zshrc or ~/.bashrc for permanent fix
```

---

## Step 3: Start Infrastructure

### 3.1 Start Docker Services
```bash
docker compose up -d postgres redis kafka zookeeper elasticsearch
```

### 3.2 Wait for Services (30 seconds)
```bash
echo "Waiting for services to start..."
sleep 30
```

### 3.3 Verify Services are Running
```bash
docker compose ps
```

Expected output - all services should show "running":
```
NAME                    STATUS
linkflow-postgres       running
linkflow-redis          running
linkflow-kafka          running
linkflow-zookeeper      running
linkflow-elasticsearch  running
```

### 3.4 Check PostgreSQL is Ready
```bash
docker exec linkflow-postgres pg_isready -U postgres
```
Expected: `accepting connections`

---

## Step 4: Setup Database

### 4.1 Create Database
```bash
docker exec -i linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow;"
```

### 4.2 Initialize Database (Extensions)
```bash
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql
```

### 4.3 Run Migrations
```bash
make migrate
```

Expected output:
```
Running migrations...
19/19 migrations applied
Migrations completed
```

### 4.4 Verify Tables Created
```bash
docker exec -i linkflow-postgres psql -U postgres -d linkflow -c "\dt"
```

You should see 40+ tables listed.

---

## Step 5: Start Application

### Option A: Full Docker Stack (Recommended)

Start all 21 microservices + Kong gateway:

```bash
make docker-up
```

Wait 1-2 minutes for all services to start, then verify:

```bash
docker compose ps
```

### Option B: Development Mode

For local development with hot reload:

```bash
# Terminal 1: Start main API
go run ./cmd/services/api

# Or use hot reload
make dev
```

---

## Step 6: Verify Setup

### 6.1 Check Health Endpoint
```bash
curl http://localhost:8000/health
```

Expected response:
```json
{"status":"healthy"}
```

### 6.2 Check Kong Gateway
```bash
curl http://localhost:8001/status
```

### 6.3 Check All Services
```bash
make health
```

---

## Step 7: Test the API

### 7.1 Register a User
```bash
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@linkflow.ai",
    "username": "admin",
    "password": "Admin123!"
  }'
```

### 7.2 Login
```bash
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@linkflow.ai",
    "password": "Admin123!"
  }'
```

Save the `token` from response.

### 7.3 Create a Workflow
```bash
TOKEN="<paste-your-token-here>"

curl -X POST http://localhost:8000/api/v1/workflows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Hello World",
    "description": "My first workflow",
    "nodes": [],
    "connections": []
  }'
```

---

## Step 8: Access UIs

| Service | URL | Login |
|---------|-----|-------|
| **Kong API** | http://localhost:8000 | - |
| **Grafana** | http://localhost:3000 | admin / admin |
| **Jaeger** | http://localhost:16686 | - |
| **Adminer** | http://localhost:8080 | System: PostgreSQL, Server: postgres, User: postgres, Password: postgres, Database: linkflow |
| **Kafka UI** | http://localhost:8090 | - |
| **Prometheus** | http://localhost:9090 | - |

---

## Step 9: Stop Services

### Stop All Services
```bash
make docker-down
```

### Stop Infrastructure Only
```bash
docker compose down
```

### Remove All Data (Clean Start)
```bash
docker compose down -v
```

---

## Quick Reference

### Start Everything
```bash
docker compose up -d postgres redis kafka zookeeper elasticsearch
sleep 30
make migrate
make docker-up
```

### Stop Everything
```bash
make docker-down
docker compose down
```

### View Logs
```bash
# All logs
make logs

# Specific service
docker compose logs -f auth
docker compose logs -f workflow
```

### Rebuild After Code Changes
```bash
make build-docker
make docker-up
```

---

## Troubleshooting

### "migrate: command not found"
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### "connection refused" on port 5432
```bash
# PostgreSQL not ready, wait more
sleep 30
docker exec linkflow-postgres pg_isready -U postgres
```

### "database linkflow does not exist"
```bash
docker exec -i linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow;"
```

### "dirty database version"
```bash
make migrate-force V=1
make migrate
```

### Port 8000 already in use
```bash
lsof -i :8000
kill -9 <PID>
```

### Reset Everything
```bash
make docker-down
docker compose down -v
docker system prune -f
# Then start from Step 3
```

---

## Next Steps

1. **Explore API**: Import `LinkFlow_AI.postman_collection.json` into Postman
2. **Create Workflows**: See [Creating Workflows Guide](../guides/creating-workflows.md)
3. **Add Integrations**: See [Integrations Guide](../guides/integrations.md)
4. **Deploy to Production**: See [Production Deployment](./production-deployment.md)
