#!/bin/bash

# LinkFlow AI Quick Start Script

set -e

echo "🚀 LinkFlow AI - Quick Start"
echo "============================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check prerequisites
echo "📋 Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Go is installed: $(go version)${NC}"
fi

# Check Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Docker is installed: $(docker --version)${NC}"
fi

# Check Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}⚠ Docker Compose not found, trying docker compose...${NC}"
    if ! docker compose version &> /dev/null; then
        echo -e "${RED}❌ Docker Compose is not installed${NC}"
        exit 1
    else
        echo -e "${GREEN}✓ Docker Compose is available${NC}"
        DOCKER_COMPOSE="docker compose"
    fi
else
    echo -e "${GREEN}✓ Docker Compose is installed: $(docker-compose --version)${NC}"
    DOCKER_COMPOSE="docker-compose"
fi

echo ""
echo "🔧 Setting up development environment..."

# Install development tools
echo "📦 Installing development tools..."
if command -v air &> /dev/null; then
    echo -e "${GREEN}✓ Air (hot reload) is already installed${NC}"
else
    echo "Installing Air for hot reload..."
    go install github.com/air-verse/air@latest
fi

if command -v migrate &> /dev/null; then
    echo -e "${GREEN}✓ Migrate tool is already installed${NC}"
else
    echo "Installing migrate tool..."
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
fi

echo ""
echo "🐳 Starting infrastructure services..."
echo "This may take a few minutes on first run..."

# Start infrastructure
$DOCKER_COMPOSE up -d postgres redis kafka zookeeper elasticsearch

echo ""
echo "⏳ Waiting for services to be ready..."

# Wait for PostgreSQL
echo -n "Waiting for PostgreSQL..."
until docker exec linkflow-postgres pg_isready -U postgres &> /dev/null; do
    echo -n "."
    sleep 2
done
echo -e " ${GREEN}Ready!${NC}"

# Wait for Redis
echo -n "Waiting for Redis..."
until docker exec linkflow-redis redis-cli ping &> /dev/null; do
    echo -n "."
    sleep 2
done
echo -e " ${GREEN}Ready!${NC}"

# Wait for Elasticsearch
echo -n "Waiting for Elasticsearch..."
until curl -s http://localhost:9200/_cluster/health &> /dev/null; do
    echo -n "."
    sleep 2
done
echo -e " ${GREEN}Ready!${NC}"

echo ""
echo "🗄️  Initializing database..."

# Run database initialization
docker exec -i linkflow-postgres psql -U postgres < scripts/init-db.sql || true

echo -e "${GREEN}✓ Database initialized${NC}"

echo ""
echo "🎉 Setup complete!"
echo ""
echo "📚 Available services:"
echo "   - PostgreSQL: localhost:5432"
echo "   - Redis: localhost:6379"
echo "   - Kafka: localhost:9092"
echo "   - Elasticsearch: localhost:9200"
echo ""
echo "🚀 To start the services:"
echo "   make dev          # Start all services with hot reload"
echo "   make run-auth     # Run auth service only"
echo "   make run-workflow # Run workflow service only"
echo ""
echo "📖 To view logs:"
echo "   make logs         # View all service logs"
echo ""
echo "🧪 To run tests:"
echo "   make test         # Run all tests"
echo ""
echo "🛑 To stop services:"
echo "   make docker-down  # Stop all Docker services"
echo ""
echo "Happy coding! 🎊"
