#!/bin/bash
set -e

echo "Building all LinkFlow services..."
mkdir -p bin

SERVICES=(gateway auth user execution workflow node schedule webhook notification analytics search storage integration monitoring config migration backup admin credential tenant executor)

for SERVICE in "${SERVICES[@]}"; do
    echo "Building $SERVICE..."
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/$SERVICE ./cmd/services/$SERVICE
done

echo "Build complete!"
ls -lh bin/
