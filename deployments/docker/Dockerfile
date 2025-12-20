# Multi-stage build for LinkFlow AI services
# Usage: docker build --build-arg SERVICE_NAME=auth -t linkflow/auth .

# =============================================================================
# Build Stage
# =============================================================================
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
COPY go.work go.work.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build argument for service name
ARG SERVICE_NAME

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/service \
    ./cmd/services/${SERVICE_NAME}

# =============================================================================
# Runtime Stage
# =============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1000 linkflow && \
    adduser -D -u 1000 -G linkflow linkflow

WORKDIR /app

# Copy binary and configs
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs

# Set ownership
RUN chown -R linkflow:linkflow /app

# Switch to non-root user
USER linkflow

# Expose default port range
EXPOSE 8000-8022

# Health check (services expose /health/live or /health)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:${HTTP_PORT:-8080}/health/live || \
        wget -q --spider http://localhost:${HTTP_PORT:-8080}/health || exit 1

# Run service
CMD ["./service"]
