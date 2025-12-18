#!/bin/bash

# LinkFlow AI - Start Services Script

set -e

echo "🚀 LinkFlow AI - Starting Services"
echo "=================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to check if port is in use
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null ; then
        return 0
    else
        return 1
    fi
}

# Function to start a service
start_service() {
    local service_name=$1
    local port=$2
    local binary=$3
    
    echo -e "${BLUE}Starting $service_name on port $port...${NC}"
    
    # Check if port is already in use
    if check_port $port; then
        echo -e "${YELLOW}⚠ Port $port is already in use. Skipping $service_name.${NC}"
        return
    fi
    
    # Check if binary exists
    if [ ! -f "$binary" ]; then
        echo -e "${YELLOW}Building $service_name...${NC}"
        go build -o "$binary" "cmd/services/${service_name}/main.go"
    fi
    
    # Start the service in background
    nohup "$binary" > "logs/${service_name}.log" 2>&1 &
    echo $! > "logs/${service_name}.pid"
    
    echo -e "${GREEN}✓ $service_name started (PID: $(cat logs/${service_name}.pid))${NC}"
}

# Create logs directory if it doesn't exist
mkdir -p logs

# Check if infrastructure is running
echo "📋 Checking infrastructure services..."

if ! docker ps | grep -q linkflow-postgres; then
    echo -e "${RED}❌ PostgreSQL is not running${NC}"
    echo "Please run: docker-compose up -d postgres"
    exit 1
else
    echo -e "${GREEN}✓ PostgreSQL is running${NC}"
fi

if ! docker ps | grep -q linkflow-redis; then
    echo -e "${YELLOW}⚠ Redis is not running (optional)${NC}"
else
    echo -e "${GREEN}✓ Redis is running${NC}"
fi

echo ""
echo "🔧 Starting microservices..."
echo ""

# Start services
start_service "auth" 8001 "bin/auth"
sleep 2
start_service "user" 8002 "bin/user"
sleep 2
start_service "workflow" 8004 "bin/workflow"
sleep 2

echo ""
echo "✨ Services started successfully!"
echo ""
echo "📚 Service URLs:"
echo "   Auth Service:     http://localhost:8001"
echo "   User Service:     http://localhost:8002"
echo "   Workflow Service: http://localhost:8004"
echo ""
echo "📊 Health Checks:"
echo "   Auth:     http://localhost:8001/health/ready"
echo "   User:     http://localhost:8002/health/ready"
echo "   Workflow: http://localhost:8004/health/ready"
echo ""
echo "📖 View logs:"
echo "   tail -f logs/auth.log"
echo "   tail -f logs/user.log"
echo "   tail -f logs/workflow.log"
echo ""
echo "🛑 To stop services:"
echo "   ./scripts/stop-services.sh"
echo ""
