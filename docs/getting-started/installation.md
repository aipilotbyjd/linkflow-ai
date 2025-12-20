# Installation Guide

This guide covers how to install and set up LinkFlow AI for development and production environments.

## Prerequisites

- **Go**: Version 1.25 or higher
- **PostgreSQL**: Version 14 or higher
- **Redis**: Version 6 or higher (optional, for caching)
- **Docker**: For containerized deployment (optional)

## Installation Methods

### Method 1: From Source

```bash
# Clone the repository
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai

# Install dependencies
go mod download

# Build the project
go build ./...

# Run database migrations
go run ./cmd/tools/migrate up
```

### Method 2: Using Docker

```bash
# Clone the repository
git clone https://github.com/linkflow-ai/linkflow-ai.git
cd linkflow-ai

# Start with Docker Compose
docker-compose up -d
```

### Method 3: Using Docker Compose (Full Stack)

```bash
# Start all services including PostgreSQL and Redis
docker-compose -f docker-compose.yml -f docker-compose.override.yml up -d
```

## Database Setup

### PostgreSQL Setup

1. Create a database:
```sql
CREATE DATABASE linkflow;
CREATE USER linkflow WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE linkflow TO linkflow;
```

2. Run migrations:
```bash
export DATABASE_URL="postgres://linkflow:your-password@localhost:5432/linkflow?sslmode=disable"
go run ./cmd/tools/migrate up
```

### Redis Setup (Optional)

Redis is used for caching and session storage. Install and start Redis:

```bash
# macOS
brew install redis
brew services start redis

# Ubuntu/Debian
sudo apt install redis-server
sudo systemctl start redis
```

## Configuration

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` with your settings:

```bash
# Server
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://linkflow:password@localhost:5432/linkflow?sslmode=disable

# Redis (optional)
REDIS_URL=redis://localhost:6379

# Security
JWT_SECRET=your-secure-jwt-secret-key

# Stripe (for billing)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

## Verifying Installation

1. Start the API server:
```bash
go run ./cmd/services/api
```

2. Check health endpoint:
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","timestamp":"2024-12-19T12:00:00Z"}
```

3. List available nodes:
```bash
curl http://localhost:8080/api/v1/nodes
```

## Troubleshooting

### Common Issues

**Database connection failed**
- Verify PostgreSQL is running: `pg_isready`
- Check DATABASE_URL format
- Ensure database exists and user has permissions

**Port already in use**
- Change PORT in .env file
- Or kill the process using the port: `lsof -i :8080`

**Migration errors**
- Ensure database user has CREATE TABLE permissions
- Check for existing tables that may conflict

## Next Steps

- [Quick Start Guide](quickstart.md)
- [Configuration Reference](configuration.md)
- [API Overview](../api/overview.md)
