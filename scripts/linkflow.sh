#!/bin/bash
# ============================================================================
# LinkFlow AI - Unified Command Interface
# ============================================================================
# Usage: ./scripts/linkflow.sh <command> [options]
# ============================================================================

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPOSE_DIR="$PROJECT_ROOT/deployments/docker/compose"

# Environment
ENV_FILE="${ENV_FILE:-$PROJECT_ROOT/.env}"
export ENV_FILE

# Docker Compose command - use project directory for correct .env resolution
DC="docker compose --project-directory $PROJECT_ROOT -f $COMPOSE_DIR/base.yml"

# ============================================================================
# Helper Functions
# ============================================================================

log() { echo -e "${GREEN}[LinkFlow]${NC} $1"; }
warn() { echo -e "${YELLOW}[Warning]${NC} $1"; }
error() { echo -e "${RED}[Error]${NC} $1" >&2; exit 1; }

usage() {
    cat << EOF
${BLUE}LinkFlow AI - Unified Command Interface${NC}

${YELLOW}Usage:${NC}
  ./scripts/linkflow.sh <command> [options]

${YELLOW}Commands:${NC}
  ${GREEN}dev${NC}              Start development environment (all ports exposed)
  ${GREEN}prod${NC}             Start production environment (Kong only)
  ${GREEN}stop${NC}             Stop all services
  ${GREEN}restart${NC}          Restart all services
  ${GREEN}build${NC}            Build all service images
  ${GREEN}rebuild${NC}          Force rebuild all images
  ${GREEN}status${NC}           Show service status
  ${GREEN}logs${NC} [service]   Show logs (all or specific service)
  ${GREEN}health${NC}           Check health of all services
  ${GREEN}shell${NC} <service>  Open shell in a service container
  ${GREEN}db${NC} <command>     Database commands (migrate, psql, reset)

${YELLOW}Options:${NC}
  -d, --detached     Run in background
  -v, --verbose      Verbose output
  -h, --help         Show this help

${YELLOW}Examples:${NC}
  ./scripts/linkflow.sh dev                  # Start dev environment
  ./scripts/linkflow.sh logs workflow        # View workflow service logs
  ./scripts/linkflow.sh shell postgres       # Open psql shell
  ./scripts/linkflow.sh db migrate           # Run migrations

EOF
    exit 0
}

# ============================================================================
# Commands
# ============================================================================

cmd_dev() {
    log "Starting development environment..."
    
    if [ ! -f "$ENV_FILE" ]; then
        warn "No .env file found. Creating from template..."
        cp "$PROJECT_ROOT/.env.example" "$ENV_FILE" 2>/dev/null || \
            error "No .env.example file found. Please create .env manually."
    fi
    
    DC="$DC -f $COMPOSE_DIR/dev.yml"
    
    if [ "$DETACHED" = true ]; then
        $DC up -d --build
        log "Services started in background"
        cmd_status
    else
        $DC up --build
    fi
}

cmd_prod() {
    log "Starting production environment..."
    
    if [ ! -f "$ENV_FILE" ]; then
        error "Production requires .env file with all secrets configured"
    fi
    
    DC="$DC -f $COMPOSE_DIR/prod.yml"
    
    if [ "$DETACHED" = true ]; then
        $DC up -d --build
        log "Production services started"
        cmd_status
    else
        $DC up --build
    fi
}

cmd_stop() {
    log "Stopping all services..."
    docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/dev.yml" down 2>/dev/null || true
    docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/prod.yml" down 2>/dev/null || true
    log "All services stopped"
}

cmd_restart() {
    log "Restarting services..."
    cmd_stop
    sleep 2
    cmd_dev
}

cmd_build() {
    log "Building all service images..."
    $DC -f "$COMPOSE_DIR/dev.yml" build
    log "Build complete"
}

cmd_rebuild() {
    log "Force rebuilding all images..."
    $DC -f "$COMPOSE_DIR/dev.yml" build --no-cache
    log "Rebuild complete"
}

cmd_status() {
    log "Service Status:"
    echo ""
    docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/dev.yml" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
    docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/prod.yml" ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
    warn "No services running"
}

cmd_logs() {
    local service="$1"
    if [ -n "$service" ]; then
        log "Showing logs for: $service"
        docker logs -f "linkflow-$service" 2>/dev/null || \
            docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/dev.yml" logs -f "$service" 2>/dev/null || \
            error "Service '$service' not found"
    else
        log "Showing all logs..."
        docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/dev.yml" logs -f 2>/dev/null || \
            docker compose -f "$COMPOSE_DIR/base.yml" -f "$COMPOSE_DIR/prod.yml" logs -f 2>/dev/null
    fi
}

cmd_health() {
    log "Health Check:"
    echo ""
    
    local services=(
        "kong:8000:/status"
        "auth:8001:/health/live"
        "user:8002:/health/live"
        "execution:8003:/health/live"
        "workflow:8004:/health/live"
        "node:8005:/health/live"
        "executor:8007:/health"
        "webhook:8008:/health/live"
        "schedule:8009:/health/live"
        "credential:8010:/health"
        "notification:8011:/health/live"
        "integration:8012:/health/live"
        "analytics:8013:/health/live"
        "search:8014:/health/live"
        "storage:8015:/health/live"
        "config:8016:/health/live"
        "admin:8017:/health/live"
        "tenant:8019:/health"
        "monitoring:8020:/health/live"
        "backup:8021:/health/live"
        "migration:8022:/health/live"
    )
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r name port path <<< "$service_info"
        if curl -sf "http://localhost:$port$path" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} $name (port $port)"
        else
            echo -e "${RED}✗${NC} $name (port $port)"
        fi
    done
    
    echo ""
    log "Infrastructure:"
    
    # Postgres
    if docker exec linkflow-postgres pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} postgres (port 5432)"
    else
        echo -e "${RED}✗${NC} postgres (port 5432)"
    fi
    
    # Redis
    if docker exec linkflow-redis redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} redis (port 6379)"
    else
        echo -e "${RED}✗${NC} redis (port 6379)"
    fi
    
    # Elasticsearch
    if curl -sf "http://localhost:9200/_cluster/health" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} elasticsearch (port 9200)"
    else
        echo -e "${RED}✗${NC} elasticsearch (port 9200)"
    fi
}

cmd_shell() {
    local service="$1"
    [ -z "$service" ] && error "Usage: $0 shell <service>"
    
    case "$service" in
        postgres|pg)
            log "Opening PostgreSQL shell..."
            docker exec -it linkflow-postgres psql -U postgres -d linkflow
            ;;
        redis)
            log "Opening Redis CLI..."
            docker exec -it linkflow-redis redis-cli
            ;;
        *)
            log "Opening shell in $service..."
            docker exec -it "linkflow-$service" sh
            ;;
    esac
}

cmd_db() {
    local subcmd="$1"
    shift || true
    
    case "$subcmd" in
        migrate)
            log "Running database migrations..."
            docker exec linkflow-postgres psql -U postgres -d linkflow -f /docker-entrypoint-initdb.d/init.sql 2>/dev/null || \
                log "Migrations already applied or init.sql not found"
            ;;
        psql)
            cmd_shell postgres
            ;;
        reset)
            warn "This will DELETE all data!"
            read -p "Are you sure? (yes/no): " confirm
            if [ "$confirm" = "yes" ]; then
                log "Resetting database..."
                docker exec linkflow-postgres psql -U postgres -c "DROP DATABASE IF EXISTS linkflow;"
                docker exec linkflow-postgres psql -U postgres -c "CREATE DATABASE linkflow;"
                log "Database reset complete"
            fi
            ;;
        *)
            echo "Database commands:"
            echo "  migrate  - Run migrations"
            echo "  psql     - Open PostgreSQL shell"
            echo "  reset    - Reset database (WARNING: deletes all data)"
            ;;
    esac
}

# ============================================================================
# Main
# ============================================================================

DETACHED=false
VERBOSE=false

# Parse options
while [[ $# -gt 0 ]]; do
    case "$1" in
        -d|--detached) DETACHED=true; shift ;;
        -v|--verbose) VERBOSE=true; set -x; shift ;;
        -h|--help) usage ;;
        *) break ;;
    esac
done

# Get command
COMMAND="${1:-help}"
shift || true

case "$COMMAND" in
    dev)        cmd_dev ;;
    prod)       cmd_prod ;;
    stop)       cmd_stop ;;
    restart)    cmd_restart ;;
    build)      cmd_build ;;
    rebuild)    cmd_rebuild ;;
    status)     cmd_status ;;
    logs)       cmd_logs "$@" ;;
    health)     cmd_health ;;
    shell)      cmd_shell "$@" ;;
    db)         cmd_db "$@" ;;
    help|--help|-h) usage ;;
    *) error "Unknown command: $COMMAND. Run '$0 --help' for usage." ;;
esac
