#!/bin/bash

# LinkFlow AI - Stop Services Script

echo "🛑 LinkFlow AI - Stopping Services"
echo "==================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to stop a service
stop_service() {
    local service_name=$1
    local pid_file="logs/${service_name}.pid"
    
    if [ -f "$pid_file" ]; then
        PID=$(cat "$pid_file")
        if kill -0 $PID 2>/dev/null; then
            echo -e "${YELLOW}Stopping $service_name (PID: $PID)...${NC}"
            kill $PID
            rm "$pid_file"
            echo -e "${GREEN}✓ $service_name stopped${NC}"
        else
            echo -e "${YELLOW}$service_name is not running (stale PID file)${NC}"
            rm "$pid_file"
        fi
    else
        echo -e "${YELLOW}$service_name is not running${NC}"
    fi
}

# Stop all services
stop_service "auth"
stop_service "user"
stop_service "workflow"
stop_service "execution"
stop_service "node"
stop_service "webhook"
stop_service "schedule"
stop_service "notification"

echo ""
echo "✨ All services stopped!"
echo ""
