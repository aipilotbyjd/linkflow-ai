# Multi-stage Dockerfile for LinkFlow AI Services

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY go.work go.work.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build argument for service name
ARG SERVICE_NAME

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o service ./cmd/services/${SERVICE_NAME}

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 linkflow && \
    adduser -D -u 1000 -G linkflow linkflow

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs

# Change ownership
RUN chown -R linkflow:linkflow /app

# Switch to non-root user
USER linkflow

# Expose port (will be overridden by docker-compose)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Run the service
ENTRYPOINT ["./service"]
