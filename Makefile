.PHONY: help build test lint proto docker-build docker-push k8s-deploy k8s-delete clean install-tools fmt vet coverage

# Variables
GO := go
GOFLAGS := -v
LDFLAGS := -ldflags="-s -w"
DOCKER_REGISTRY ?= docker.io
DOCKER_IMAGE ?= event-gateway
DOCKER_TAG ?= latest
DOCKER_FULL := $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
KUBECTL := kubectl
NAMESPACE ?= default
PROTOC_GEN_GO := $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(shell go env GOPATH)/bin/protoc-gen-go-grpc

# Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m

help: ## Show this help message
	@echo "$(BLUE)Event Streaming Gateway - Build targets$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

install-tools: ## Install required tools
	@echo "$(BLUE)Installing tools...$(NC)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	$(GO) fmt ./...
	goimports -w .

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GO) vet ./...

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(YELLOW)golangci-lint not found. Run 'make install-tools'$(NC)" && exit 1)
	golangci-lint run --timeout=10m

build: vet ## Build gateway binary
	@echo "$(BLUE)Building gateway...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) \
		-o gateway cmd/gateway/main.go
	@echo "$(GREEN)Build complete: gateway$(NC)"

build-dev: vet ## Build gateway binary for development
	@echo "$(BLUE)Building gateway (dev)...$(NC)"
	$(GO) build $(GOFLAGS) -o gateway cmd/gateway/main.go
	@echo "$(GREEN)Build complete: gateway$(NC)"

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	$(GO) test -v -race -count=1 ./...

coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

test-short: ## Run short tests
	@echo "$(BLUE)Running short tests...$(NC)"
	$(GO) test -v -short ./...

bench: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GO) test -v -bench=. -benchmem ./...

proto: ## Generate protobuf code
	@echo "$(BLUE)Generating protobuf code...$(NC)"
	@./scripts/generate-proto.sh
	@echo "$(GREEN)Protobuf generation complete$(NC)"

docker-build: build ## Build Docker image
	@echo "$(BLUE)Building Docker image: $(DOCKER_FULL)$(NC)"
	docker build -t $(DOCKER_FULL) -f Dockerfile .
	@echo "$(GREEN)Docker image built: $(DOCKER_FULL)$(NC)"

docker-build-dev: build-dev ## Build Docker image for development
	@echo "$(BLUE)Building Docker image (dev): $(DOCKER_FULL)$(NC)"
	docker build -t $(DOCKER_FULL) -f Dockerfile .
	@echo "$(GREEN)Docker image built: $(DOCKER_FULL)$(NC)"

docker-push: docker-build ## Push Docker image to registry
	@echo "$(BLUE)Pushing Docker image: $(DOCKER_FULL)$(NC)"
	docker push $(DOCKER_FULL)
	@echo "$(GREEN)Docker image pushed$(NC)"

docker-run: docker-build ## Run gateway in Docker
	@echo "$(BLUE)Running gateway in Docker...$(NC)"
	docker run -it --rm \
		-p 50051:50051 \
		-p 8080:8080 \
		-p 8081:8081 \
		-e KAFKA_BROKERS=kafka:9092 \
		-e LOG_LEVEL=info \
		$(DOCKER_FULL)

docker-compose-up: ## Start docker-compose stack
	@echo "$(BLUE)Starting Docker Compose stack...$(NC)"
	docker-compose -f deployments/docker/docker-compose.yml up -d
	@echo "$(GREEN)Docker Compose stack started$(NC)"
	@echo "$(YELLOW)Services:$(NC)"
	@echo "  - Event Gateway: http://localhost:8080"
	@echo "  - Prometheus: http://localhost:9090"
	@echo "  - Kafka: localhost:9092"

docker-compose-down: ## Stop docker-compose stack
	@echo "$(BLUE)Stopping Docker Compose stack...$(NC)"
	docker-compose -f deployments/docker/docker-compose.yml down
	@echo "$(GREEN)Docker Compose stack stopped$(NC)"

docker-compose-logs: ## Show docker-compose logs
	docker-compose -f deployments/docker/docker-compose.yml logs -f

k8s-deploy: ## Deploy to Kubernetes
	@echo "$(BLUE)Deploying to Kubernetes (namespace: $(NAMESPACE))...$(NC)"
	$(KUBECTL) apply -f deployments/kubernetes/configmap.yaml -n $(NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/deployment.yaml -n $(NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/service.yaml -n $(NAMESPACE)
	$(KUBECTL) apply -f deployments/kubernetes/hpa.yaml -n $(NAMESPACE)
	@echo "$(GREEN)Kubernetes deployment complete$(NC)"
	@echo "$(YELLOW)Check status with:$(NC)"
	@echo "  kubectl get pods -n $(NAMESPACE) -l app=event-gateway"

k8s-delete: ## Delete from Kubernetes
	@echo "$(BLUE)Deleting from Kubernetes (namespace: $(NAMESPACE))...$(NC)"
	$(KUBECTL) delete -f deployments/kubernetes/ -n $(NAMESPACE) --ignore-not-found
	@echo "$(GREEN)Kubernetes resources deleted$(NC)"

k8s-logs: ## Show Kubernetes logs
	@echo "$(BLUE)Showing logs from Kubernetes (namespace: $(NAMESPACE))...$(NC)"
	$(KUBECTL) logs -f -n $(NAMESPACE) -l app=event-gateway --all-containers=true

k8s-status: ## Show Kubernetes status
	@echo "$(BLUE)Kubernetes status (namespace: $(NAMESPACE)):$(NC)"
	$(KUBECTL) get all -n $(NAMESPACE) -l app=event-gateway
	$(KUBECTL) describe hpa -n $(NAMESPACE) event-gateway

k8s-shell: ## Open shell in Kubernetes pod
	@echo "$(BLUE)Opening shell in Kubernetes pod (namespace: $(NAMESPACE))...$(NC)"
	$(KUBECTL) exec -it -n $(NAMESPACE) \
		$$($(KUBECTL) get pod -n $(NAMESPACE) -l app=event-gateway -o jsonpath='{.items[0].metadata.name}') \
		-- /bin/sh

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f gateway
	rm -f dist/*
	rm -f coverage.out coverage.html
	$(GO) clean -cache
	@echo "$(GREEN)Clean complete$(NC)"

deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify
	@echo "$(GREEN)Dependencies verified$(NC)"

tidy: ## Tidy go modules
	@echo "$(BLUE)Tidying modules...$(NC)"
	$(GO) mod tidy
	@echo "$(GREEN)Modules tidied$(NC)"

ci: lint test coverage ## Run CI pipeline (lint, test, coverage)

all: clean deps lint test build ## Run full build pipeline

.PHONY: all ci help build build-dev test test-short bench coverage fmt vet lint proto docker-build docker-build-dev docker-push docker-run docker-compose-up docker-compose-down docker-compose-logs k8s-deploy k8s-delete k8s-logs k8s-status k8s-shell clean deps tidy install-tools
