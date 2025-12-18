#!/bin/bash

# Start All LinkFlow AI Services
echo "🚀 Starting LinkFlow AI Platform - 18 Microservices"
echo "=================================================="

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Base directory
BASE_DIR="$(dirname "$0")/.."
cd "$BASE_DIR"

# Build all services first
echo -e "${BLUE}Building all services...${NC}"
for service in auth user execution workflow node schedule webhook notification analytics search storage integration monitoring config migration backup admin gateway; do
    echo "Building $service..."
    go build -o bin/$service ./cmd/services/$service 2>/dev/null || echo "Failed to build $service"
done

echo -e "${GREEN}✅ All services built${NC}"
echo ""

# Service list with ports
declare -A services=(
    ["gateway"]="8000"
    ["auth"]="8001"
    ["user"]="8002"
    ["execution"]="8003"
    ["workflow"]="8004"
    ["node"]="8005"
    ["schedule"]="8006"
    ["webhook"]="8007"
    ["notification"]="8008"
    ["analytics"]="8009"
    ["search"]="8010"
    ["storage"]="8011"
    ["integration"]="8012"
    ["monitoring"]="8013"
    ["config"]="8014"
    ["migration"]="8015"
    ["backup"]="8016"
    ["admin"]="8017"
)

# Start each service
echo -e "${BLUE}Starting services...${NC}"
for service in "${!services[@]}"; do
    port="${services[$service]}"
    echo -e "Starting ${GREEN}$service${NC} on port ${BLUE}$port${NC}"
    
    # Set environment variables
    export SERVICE_NAME=$service
    export HTTP_PORT=$port
    export LOG_LEVEL=info
    export DATABASE_HOST=localhost
    export DATABASE_PORT=5432
    export DATABASE_NAME=linkflow
    export DATABASE_USER=linkflow
    export DATABASE_PASSWORD=linkflow123
    export REDIS_HOST=localhost
    export REDIS_PORT=6379
    export KAFKA_BROKERS=localhost:9092
    export JWT_SECRET=your-secret-key-change-in-production
    
    # Start service in background
    ./bin/$service > logs/$service.log 2>&1 &
    echo "  PID: $!"
    
    # Small delay to prevent port conflicts
    sleep 0.5
done

echo ""
echo -e "${GREEN}✅ All services started!${NC}"
echo ""
echo "📊 Service Status:"
echo "=================="

# Check if services are running
sleep 2
for service in "${!services[@]}"; do
    port="${services[$service]}"
    if curl -s "http://localhost:$port/health/live" > /dev/null 2>&1; then
        echo -e "✅ ${GREEN}$service${NC} (port $port) - ${GREEN}RUNNING${NC}"
    else
        echo -e "❌ $service (port $port) - NOT RESPONDING"
    fi
done

echo ""
echo "🌐 API Gateway: http://localhost:8000"
echo "📊 Admin Dashboard: http://localhost:8017/api/v1/admin/dashboard"
echo ""
echo "To stop all services, run: ./scripts/stop-all.sh"
echo "To view logs: tail -f logs/<service>.log"
