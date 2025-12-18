#!/bin/bash
# LinkFlow AI - Start All Services
# Usage: ./scripts/start-all.sh [dev|prod]

set -e

MODE=${1:-dev}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$PROJECT_DIR/bin"
LOG_DIR="$PROJECT_DIR/logs"
PID_DIR="$PROJECT_DIR/.pids"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Services and ports
declare -A SERVICES=(
    ["gateway"]="8000"
    ["auth"]="8001"
    ["user"]="8002"
    ["execution"]="8003"
    ["workflow"]="8004"
    ["node"]="8005"
    ["tenant"]="8006"
    ["executor"]="8007"
    ["webhook"]="8008"
    ["schedule"]="8009"
    ["credential"]="8010"
    ["notification"]="8011"
    ["integration"]="8012"
    ["analytics"]="8013"
    ["search"]="8014"
    ["storage"]="8015"
    ["backup"]="8016"
    ["admin"]="8017"
    ["monitoring"]="8018"
    ["config"]="8019"
    ["migration"]="8020"
)

# Create directories
mkdir -p "$LOG_DIR" "$PID_DIR"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    # Check if binaries exist
    for service in "${!SERVICES[@]}"; do
        if [[ ! -f "$BIN_DIR/$service" ]]; then
            log_error "Binary not found: $BIN_DIR/$service"
            log_info "Run 'make build-all' or './scripts/build-all.sh' first"
            exit 1
        fi
    done
    
    log_info "All binaries found"
}

check_infrastructure() {
    log_info "Checking infrastructure services..."
    
    # Check PostgreSQL
    if ! nc -z localhost 5432 2>/dev/null; then
        log_warn "PostgreSQL not running on port 5432"
        log_info "Start with: docker-compose up -d postgres"
    fi
    
    # Check Redis
    if ! nc -z localhost 6379 2>/dev/null; then
        log_warn "Redis not running on port 6379"
        log_info "Start with: docker-compose up -d redis"
    fi
    
    # Check Kafka
    if ! nc -z localhost 9092 2>/dev/null; then
        log_warn "Kafka not running on port 9092"
        log_info "Start with: docker-compose up -d kafka"
    fi
}

start_service() {
    local service=$1
    local port=${SERVICES[$service]}
    local pid_file="$PID_DIR/$service.pid"
    local log_file="$LOG_DIR/$service.log"
    
    # Check if already running
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_warn "$service already running (PID: $pid)"
            return 0
        fi
        rm -f "$pid_file"
    fi
    
    # Start service
    log_info "Starting $service on port $port..."
    
    HTTP_PORT=$port \
    SERVICE_NAME=$service \
    "$BIN_DIR/$service" >> "$log_file" 2>&1 &
    
    local pid=$!
    echo $pid > "$pid_file"
    
    # Wait for service to be ready
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        log_info "$service started (PID: $pid)"
    else
        log_error "Failed to start $service"
        return 1
    fi
}

stop_service() {
    local service=$1
    local pid_file="$PID_DIR/$service.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping $service (PID: $pid)..."
            kill "$pid"
            rm -f "$pid_file"
        fi
    fi
}

start_all() {
    log_info "Starting all LinkFlow services..."
    
    check_dependencies
    check_infrastructure
    
    local failed=0
    for service in "${!SERVICES[@]}"; do
        if ! start_service "$service"; then
            ((failed++))
        fi
    done
    
    if [[ $failed -eq 0 ]]; then
        log_info "All services started successfully!"
        echo ""
        log_info "Service URLs:"
        for service in "${!SERVICES[@]}"; do
            echo "  - $service: http://localhost:${SERVICES[$service]}"
        done
    else
        log_error "$failed service(s) failed to start"
        exit 1
    fi
}

stop_all() {
    log_info "Stopping all LinkFlow services..."
    
    for service in "${!SERVICES[@]}"; do
        stop_service "$service"
    done
    
    log_info "All services stopped"
}

status() {
    echo "Service Status:"
    echo "==============="
    
    for service in "${!SERVICES[@]}"; do
        local port=${SERVICES[$service]}
        local pid_file="$PID_DIR/$service.pid"
        local status="${RED}STOPPED${NC}"
        
        if [[ -f "$pid_file" ]]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                status="${GREEN}RUNNING${NC} (PID: $pid)"
            fi
        fi
        
        printf "  %-15s :%s  %b\n" "$service" "$port" "$status"
    done
}

case "$1" in
    start|dev|prod)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        stop_all
        sleep 2
        start_all
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status}"
        echo ""
        echo "Commands:"
        echo "  start   - Start all services"
        echo "  stop    - Stop all services"
        echo "  restart - Restart all services"
        echo "  status  - Show service status"
        exit 1
        ;;
esac
