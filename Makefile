# LinkFlow AI - Makefile for Developer Productivity
.PHONY: help dev test build clean docker-up docker-down migrate

# Variables
SERVICES := auth user workflow execution node webhook schedule notification
DOCKER_REGISTRY := linkflow
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(wildcard *.go)

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

dev: ## Start development environment
	@echo "${GREEN}Starting development environment...${NC}"
	docker-compose up -d postgres redis kafka elasticsearch
	@echo "${GREEN}Waiting for services to be ready...${NC}"
	@sleep 10
	@$(MAKE) migrate
	@echo "${GREEN}Starting services with hot reload...${NC}"
	air -c .air.toml

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

docker-up: ## Start all services with docker-compose
	@echo "${GREEN}Starting Docker services...${NC}"
	docker-compose up -d
	@echo "${GREEN}Services started. Run 'docker-compose logs -f' to view logs${NC}"

docker-down: ## Stop all docker-compose services
	@echo "${YELLOW}Stopping Docker services...${NC}"
	docker-compose down
	@echo "${GREEN}Services stopped${NC}"

migrate: ## Run database migrations
	@echo "${GREEN}Running migrations...${NC}"
	@for service in $(SERVICES); do \
		echo "Migrating $$service schema..."; \
		migrate -path migrations/$$service -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable&search_path=$$service" up || exit 1; \
	done
	@echo "${GREEN}Migrations completed${NC}"

migrate-down: ## Rollback database migrations
	@echo "${YELLOW}Rolling back migrations...${NC}"
	@for service in $(SERVICES); do \
		echo "Rolling back $$service schema..."; \
		migrate -path migrations/$$service -database "postgresql://postgres:postgres@localhost:5432/linkflow?sslmode=disable&search_path=$$service" down 1; \
	done

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
	docker-compose logs -f

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning build artifacts...${NC}"
	rm -rf bin/ dist/ coverage.* vendor/ *.out
	@echo "${GREEN}Cleanup completed${NC}"

.DEFAULT_GOAL := help
