.PHONY: build run test clean docker-build docker-run help

# Variables
BINARY_NAME=github-copilot-metrics-exporter
DOCKER_IMAGE=github-copilot-metrics-exporter
VERSION?=latest

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o $(BINARY_NAME) .

run: ## Run the application
	go run .

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-run: ## Run Docker container
	docker run -d \
		-p 9101:9101 \
		-e GITHUB_TOKEN=$(GITHUB_TOKEN) \
		-e GITHUB_ORG=$(GITHUB_ORG) \
		$(DOCKER_IMAGE):$(VERSION)

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...

lint: ## Run linters
	go vet ./...
	go fmt ./...

all: clean deps fmt lint build test ## Run all tasks
