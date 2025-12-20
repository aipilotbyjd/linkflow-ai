# Multi-stage build for optimization
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o service cmd/services/${SERVICE_NAME}/main.go

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/service .
COPY --from=builder /app/configs ./configs
# Service ports range from 8000-8019, expose based on HTTP_PORT env var
EXPOSE 8000-8019
CMD ["./service"]
