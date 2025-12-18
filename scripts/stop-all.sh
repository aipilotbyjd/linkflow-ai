#!/bin/bash

# Stop All LinkFlow AI Services
echo "🛑 Stopping LinkFlow AI Platform Services"
echo "========================================="

# Kill all service processes
for service in gateway auth user execution workflow node schedule webhook notification analytics search storage integration monitoring config migration backup admin; do
    echo "Stopping $service..."
    pkill -f "bin/$service" 2>/dev/null || echo "  $service not running"
done

echo ""
echo "✅ All services stopped"
