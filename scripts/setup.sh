#!/bin/bash

# ============================================================================
# LinkFlow AI - Automated Setup Script
# Run this script to set up the entire application from scratch
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_step() {
    echo -e "\n${BLUE}==>${NC} ${GREEN}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}    $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Header
echo -e "${GREEN}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║           LinkFlow AI - Setup Script                      ║"
echo "║           Complete End-to-End Setup                       ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# ============================================================================
# Step 1: Check Prerequisites
# ============================================================================
print_step "Step 1/8: Checking prerequisites..."

# Check Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Install with: brew install go"
    exit 1
fi
print_success "Go $(go version | awk '{print $3}')"

# Check Docker
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed. Install Docker Desktop from https://docker.com"
    exit 1
fi
print_success "Docker $(docker --version | awk '{print $3}' | tr -d ',')"

# Check Docker is running
if ! docker info &> /dev/null; then
    print_error "Docker is not running. Please start Docker Desktop."
    exit 1
fi
print_success "Docker is running"

# Check Docker Compose
if ! docker compose version &> /dev/null; then
    print_error "Docker Compose is not available"
    exit 1
fi
print_success "Docker Compose $(docker compose version --short)"

# ============================================================================
# Step 2: Setup Environment
# ============================================================================
print_step "Step 2/8: Setting up environment..."

if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        print_success "Created .env from .env.example"
    else
        print_error ".env.example not found"
        exit 1
    fi
else
    print_info ".env already exists, skipping"
fi

# ============================================================================
# Step 3: Install Dependencies
# ============================================================================
print_step "Step 3/8: Installing dependencies..."

print_info "Downloading Go modules..."
go mod download
print_success "Go modules installed"

# Install migrate if not present
if ! command -v migrate &> /dev/null; then
    print_info "Installing migrate CLI..."
    go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
    export PATH=$PATH:$(go env GOPATH)/bin
    print_success "Migrate CLI installed"
else
    print_success "Migrate CLI already installed"
fi

# ============================================================================
# Step 4: Start Infrastructure
# ============================================================================
print_step "Step 4/8: Starting infrastructure services..."

print_info "Starting PostgreSQL, Redis, Kafka, Elasticsearch..."
docker compose up -d postgres redis kafka zookeeper elasticsearch

print_info "Waiting for services to be ready (45 seconds)..."
sleep 45

# Verify PostgreSQL
until docker exec linkflow-postgres pg_isready -U postgres &> /dev/null; do
    print_info "Waiting for PostgreSQL..."
    sleep 5
done
print_success "PostgreSQL is ready"

# Verify Redis
until docker exec linkflow-redis redis-cli ping &> /dev/null; do
    print_info "Waiting for Redis..."
    sleep 5
done
print_success "Redis is ready"

# ============================================================================
# Step 5: Setup Database
# ============================================================================
print_step "Step 5/8: Setting up database..."

# Create database if not exists
docker exec -i linkflow-postgres psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'linkflow'" | grep -q 1 || \
    docker exec -i linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow;"
print_success "Database 'linkflow' ready"

# Initialize extensions
print_info "Initializing database extensions..."
docker exec -i linkflow-postgres psql -U postgres -d linkflow < scripts/init-db.sql 2>/dev/null || true
print_success "Database initialized"

# ============================================================================
# Step 6: Run Migrations
# ============================================================================
print_step "Step 6/8: Running database migrations..."

export PATH=$PATH:$(go env GOPATH)/bin
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" up

print_success "All migrations applied"

# ============================================================================
# Step 7: Start Application Services
# ============================================================================
print_step "Step 7/8: Starting application services..."

print_info "Starting all microservices and Kong gateway..."
docker compose up -d

print_info "Waiting for services to start (60 seconds)..."
sleep 60

# ============================================================================
# Step 8: Verify Setup
# ============================================================================
print_step "Step 8/8: Verifying setup..."

# Check health endpoint
if curl -s http://localhost:8000/health > /dev/null 2>&1; then
    print_success "API Gateway (Kong) is responding"
else
    print_info "Kong gateway may still be starting up..."
fi

# Show running containers
echo ""
print_info "Running containers:"
docker compose ps --format "table {{.Name}}\t{{.Status}}" | head -20

# ============================================================================
# Complete
# ============================================================================
echo ""
echo -e "${GREEN}"
echo "╔═══════════════════════════════════════════════════════════╗"
echo "║              Setup Complete!                              ║"
echo "╚═══════════════════════════════════════════════════════════╝"
echo -e "${NC}"

echo -e "${YELLOW}Access Points:${NC}"
echo "  API Gateway:    http://localhost:8000"
echo "  Kong Admin:     http://localhost:8001"
echo "  Grafana:        http://localhost:3000  (admin/admin)"
echo "  Jaeger:         http://localhost:16686"
echo "  Adminer:        http://localhost:8080  (postgres/postgres)"
echo "  Kafka UI:       http://localhost:8090"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Test API:     curl http://localhost:8000/health"
echo "  2. View logs:    make logs"
echo "  3. Stop:         make docker-down"
echo ""
echo -e "${YELLOW}Documentation:${NC}"
echo "  Setup Guide:     docs/getting-started/SETUP.md"
echo "  API Reference:   docs/api/overview.md"
echo ""
