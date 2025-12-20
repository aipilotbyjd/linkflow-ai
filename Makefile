# LinkFlow AI - Makefile for Developer Productivity
.PHONY: help dev prod stop restart status logs health shell build clean docker-up docker-down migrate

# Variables
SERVICES := gateway auth user execution workflow node tenant executor webhook schedule credential notification integration analytics search storage config migration backup admin monitoring
DOCKER_REGISTRY := linkflow
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)
COMPOSE_DIR := deployments/docker/compose
LINKFLOW := ./scripts/linkflow.sh

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Start development environment (all ports exposed)
	@$(LINKFLOW) -d dev

prod: ## Start production environment (Kong only)
	@$(LINKFLOW) -d prod

stop: ## Stop all services
	@$(LINKFLOW) stop

restart: ## Restart all services
	@$(LINKFLOW) restart

up: dev ## Alias for dev
down: stop ## Alias for stop

test: ## Run all tests
	@echo "${GREEN}Running tests...${NC}"
	go test -v -race -cover ./...

test-coverage: ## Run tests with coverage report
	@echo "${GREEN}Running tests with coverage...${NC}"
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

lint: ## Run linters
	@echo "${GREEN}Running linters...${NC}"
	golangci-lint run --fix
	staticcheck ./...

build: ## Build all services
	@echo "${GREEN}Building services...${NC}"
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$$service cmd/services/$$service/main.go || exit 1; \
	done
	@echo "${GREEN}All services built successfully${NC}"

build-docker: ## Build Docker images for all services
	@echo "${GREEN}Building Docker images...${NC}"
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build -f deployments/docker/Dockerfile \
			--build-arg SERVICE_NAME=$$service \
			-t $(DOCKER_REGISTRY)/$$service:$(VERSION) . || exit 1; \
	done
	@echo "${GREEN}All Docker images built successfully${NC}"

docker-up: dev ## Start all services (alias for dev)

docker-down: stop ## Stop all services (alias for stop)

migrate: ## Run database migrations
	@echo "${GREEN}Running migrations...${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" up
	@echo "${GREEN}Migrations completed${NC}"

migrate-down: ## Rollback database migrations (1 step)
	@echo "${YELLOW}Rolling back migration...${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" down 1

migrate-down-all: ## Rollback all migrations
	@echo "${YELLOW}Rolling back all migrations...${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" down -all

migrate-version: ## Show current migration version
	@echo "${GREEN}Current migration version:${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" version

migrate-force: ## Force migration version (usage: make migrate-force V=000001)
	@echo "${YELLOW}Forcing migration version to $(V)...${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" force $(V)

seed: ## Seed development data
	@echo "${GREEN}Seeding development data...${NC}"
	go run cmd/tools/seed/main.go

generate: ## Generate code (mocks, protobufs, etc.)
	@echo "${GREEN}Generating code...${NC}"
	go generate ./...
	@if [ -f "api/grpc/*.proto" ]; then \
		protoc --go_out=. --go-grpc_out=. api/grpc/*.proto; \
	fi
	@if [ -f "api/openapi/*.yaml" ]; then \
		oapi-codegen -generate types,server,spec api/openapi/*.yaml; \
	fi
	@echo "${GREEN}Code generation completed${NC}"

install-tools: ## Install required development tools
	@echo "${GREEN}Installing development tools...${NC}"
	go install github.com/cosmtrek/air@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golang/mock/mockgen@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "${GREEN}Tools installed successfully${NC}"

run-auth: ## Run auth service
	go run cmd/services/auth/main.go

run-workflow: ## Run workflow service
	go run cmd/services/workflow/main.go

run-execution: ## Run execution service
	go run cmd/services/execution/main.go

k8s-deploy: ## Deploy to Kubernetes
	@echo "${GREEN}Deploying to Kubernetes...${NC}"
	kubectl apply -k deployments/kubernetes/overlays/dev/
	@echo "${GREEN}Deployment completed${NC}"

k8s-delete: ## Delete Kubernetes deployment
	@echo "${YELLOW}Deleting Kubernetes deployment...${NC}"
	kubectl delete -k deployments/kubernetes/overlays/dev/

logs: ## Show logs for all services
	@$(LINKFLOW) logs

logs-%: ## Show logs for specific service (e.g., make logs-workflow)
	@$(LINKFLOW) logs $*

status: ## Show status of all services
	@$(LINKFLOW) status

health: ## Check health of all services
	@$(LINKFLOW) health

shell-%: ## Open shell in service (e.g., make shell-postgres)
	@$(LINKFLOW) shell $*

db-migrate: ## Run database migrations
	@$(LINKFLOW) db migrate

db-psql: ## Open PostgreSQL shell
	@$(LINKFLOW) db psql

db-reset: ## Reset database (WARNING: deletes all data)
	@$(LINKFLOW) db reset

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning build artifacts...${NC}"
	rm -rf bin/ dist/ coverage.* vendor/ *.out
	@echo "${GREEN}Cleanup completed${NC}"

# Additional targets

build-all: ## Build all 21 services
	@echo "${GREEN}Building all services...${NC}"
	@mkdir -p bin
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/$$service ./cmd/services/$$service || exit 1; \
	done
	@echo "${GREEN}All 21 services built successfully${NC}"
	@ls -lh bin/

start-all: dev ## Start all services (alias for dev)

stop-all: stop ## Stop all services (alias for stop)

test-unit: ## Run unit tests only
	@echo "${GREEN}Running unit tests...${NC}"
	go test -v ./tests/unit/...

test-integration: ## Run integration tests
	@echo "${GREEN}Running integration tests...${NC}"
	go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run end-to-end tests
	@echo "${GREEN}Running E2E tests...${NC}"
	go test -v -tags=e2e ./tests/e2e/...

test-all: test-unit test-integration test-e2e ## Run all tests

validate: lint test ## Validate code (lint + test)

proto: ## Generate gRPC code from proto files
	@echo "${GREEN}Generating gRPC code...${NC}"
	protoc --go_out=. --go-grpc_out=. api/grpc/*.proto

openapi: ## Generate OpenAPI client code
	@echo "${GREEN}Generating OpenAPI code...${NC}"
	@for spec in api/openapi/*.yaml; do \
		echo "Processing $$spec..."; \
	done

docker-build-all: ## Build all Docker images
	@echo "${GREEN}Building all Docker images...${NC}"
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build --build-arg SERVICE_NAME=$$service -t linkflow/$$service:$(VERSION) . || exit 1; \
	done

docker-push: ## Push Docker images to registry
	@echo "${GREEN}Pushing Docker images...${NC}"
	@for service in $(SERVICES); do \
		docker push $(DOCKER_REGISTRY)/$$service:$(VERSION); \
	done

migrate-all: ## Run all migrations
	@echo "${GREEN}Running all migrations...${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" up

migrate-status: ## Show migration status
	@echo "${GREEN}Migration status:${NC}"
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable" version

.DEFAULT_GOAL := help
