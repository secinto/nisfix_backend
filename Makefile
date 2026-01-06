# NisFix Backend Makefile
# Comprehensive build system with versioning, testing, and deployment

# ============================================================================
# Variables
# ============================================================================

# Application info
APP_NAME := nisfix-backend
PACKAGE := github.com/checkfix-tools/nisfix_backend

# Build directories
BIN_DIR := bin
DIST_DIR := dist

# Go settings
GO := go
GOFLAGS := -v
CGO_ENABLED := 0

# Version info (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Linker flags for versioning
LDFLAGS := -ldflags "\
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.GitBranch=$(GIT_BRANCH) \
	-s -w"

# Docker settings
DOCKER_REGISTRY ?=
DOCKER_IMAGE := $(APP_NAME)
DOCKER_TAG ?= $(VERSION)
DOCKER_FULL_IMAGE := $(if $(DOCKER_REGISTRY),$(DOCKER_REGISTRY)/$(DOCKER_IMAGE),$(DOCKER_IMAGE))

# Test settings
TEST_FLAGS := -v -race -cover
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Tools
GOLANGCI_LINT := golangci-lint
SWAG := swag

# ============================================================================
# Default target
# ============================================================================

.DEFAULT_GOAL := help

# ============================================================================
# Build targets
# ============================================================================

.PHONY: build
build: ## Build the binary for current platform
	@echo "Building $(APP_NAME) v$(VERSION)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME) ./cmd/server
	@echo "Binary: $(BIN_DIR)/$(APP_NAME)"

.PHONY: build-linux
build-linux: ## Cross-compile for Linux (amd64)
	@echo "Building $(APP_NAME) for Linux amd64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-amd64 ./cmd/server
	@echo "Binary: $(BIN_DIR)/$(APP_NAME)-linux-amd64"

.PHONY: build-linux-arm64
build-linux-arm64: ## Cross-compile for Linux (arm64)
	@echo "Building $(APP_NAME) for Linux arm64..."
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-linux-arm64 ./cmd/server
	@echo "Binary: $(BIN_DIR)/$(APP_NAME)-linux-arm64"

.PHONY: build-darwin
build-darwin: ## Cross-compile for macOS (arm64)
	@echo "Building $(APP_NAME) for macOS arm64..."
	@mkdir -p $(BIN_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/server
	@echo "Binary: $(BIN_DIR)/$(APP_NAME)-darwin-arm64"

.PHONY: build-all
build-all: build-linux build-linux-arm64 build-darwin ## Build for all platforms
	@echo "All platform builds complete!"

# ============================================================================
# Development targets
# ============================================================================

.PHONY: dev
dev: ## Run in development mode with hot reload (requires air)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not found, running directly..."; \
		$(GO) run ./cmd/server; \
	fi

.PHONY: run
run: build ## Build and run the server
	./$(BIN_DIR)/$(APP_NAME)

# ============================================================================
# Test targets
# ============================================================================

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	$(GO) test $(TEST_FLAGS) ./...

.PHONY: test-short
test-short: ## Run short tests only
	@echo "Running short tests..."
	$(GO) test -v -short ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

.PHONY: test-coverage-func
test-coverage-func: ## Show coverage by function
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=$(COVERAGE_FILE) ./...
	$(GO) tool cover -func=$(COVERAGE_FILE)

.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -v -bench=. -benchmem ./...

# ============================================================================
# Code quality targets
# ============================================================================

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted!"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "All checks passed!"

# ============================================================================
# Documentation targets
# ============================================================================

.PHONY: swagger
swagger: ## Generate Swagger documentation
	@echo "Generating Swagger documentation..."
	@if command -v $(SWAG) > /dev/null; then \
		$(SWAG) init -g cmd/server/main.go -o docs --parseDependency --parseInternal; \
		echo "Swagger docs generated in docs/"; \
	else \
		echo "swag not found. Install with: go install github.com/swaggo/swag/cmd/swag@latest"; \
		exit 1; \
	fi

# ============================================================================
# Dependency targets
# ============================================================================

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "Dependencies downloaded!"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated!"

.PHONY: deps-tidy
deps-tidy: ## Tidy up dependencies
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "Dependencies tidied!"

# ============================================================================
# Docker targets
# ============================================================================

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_FULL_IMAGE):$(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg GIT_BRANCH=$(GIT_BRANCH) \
		-t $(DOCKER_FULL_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_FULL_IMAGE):latest \
		.
	@echo "Docker image built: $(DOCKER_FULL_IMAGE):$(DOCKER_TAG)"

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
		echo "DOCKER_REGISTRY not set. Usage: make docker-push DOCKER_REGISTRY=your-registry.com"; \
		exit 1; \
	fi
	@echo "Pushing Docker image $(DOCKER_FULL_IMAGE):$(DOCKER_TAG)..."
	docker push $(DOCKER_FULL_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_FULL_IMAGE):latest
	@echo "Docker image pushed!"

.PHONY: docker-run
docker-run: ## Run Docker container locally
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 \
		--env-file .env \
		--name $(APP_NAME) \
		$(DOCKER_FULL_IMAGE):$(DOCKER_TAG)

.PHONY: docker-compose-up
docker-compose-up: ## Start local development stack
	@echo "Starting development stack..."
	docker-compose up -d
	@echo "Development stack started!"
	@echo "API: http://localhost:8080"
	@echo "Swagger: http://localhost:8080/swagger/index.html"

.PHONY: docker-compose-down
docker-compose-down: ## Stop local development stack
	@echo "Stopping development stack..."
	docker-compose down
	@echo "Development stack stopped!"

.PHONY: docker-compose-logs
docker-compose-logs: ## View development stack logs
	docker-compose logs -f

# ============================================================================
# Key generation targets
# ============================================================================

.PHONY: generate-keys
generate-keys: ## Generate RSA key pair for JWT
	@echo "Generating RSA key pair..."
	@./scripts/generate-keys.sh
	@echo "Keys generated!"

# ============================================================================
# Clean targets
# ============================================================================

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	$(GO) clean -cache -testcache
	@echo "Cleaned!"

.PHONY: clean-docker
clean-docker: ## Remove Docker images
	@echo "Removing Docker images..."
	docker rmi $(DOCKER_FULL_IMAGE):$(DOCKER_TAG) $(DOCKER_FULL_IMAGE):latest 2>/dev/null || true
	@echo "Docker images removed!"

# ============================================================================
# Tool installation targets
# ============================================================================

.PHONY: install-tools
install-tools: ## Install required development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/air-verse/air@latest
	@echo "Tools installed!"

# ============================================================================
# Info targets
# ============================================================================

.PHONY: version
version: ## Display version information
	@echo "App:      $(APP_NAME)"
	@echo "Version:  $(VERSION)"
	@echo "Commit:   $(GIT_COMMIT)"
	@echo "Branch:   $(GIT_BRANCH)"
	@echo "Build:    $(BUILD_TIME)"

.PHONY: info
info: ## Display build information
	@echo "=== Build Information ==="
	@echo "App:        $(APP_NAME)"
	@echo "Package:    $(PACKAGE)"
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(GIT_COMMIT)"
	@echo "Branch:     $(GIT_BRANCH)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo ""
	@echo "=== Go Information ==="
	@echo "Go Version: $(shell $(GO) version)"
	@echo "GOOS:       $(shell $(GO) env GOOS)"
	@echo "GOARCH:     $(shell $(GO) env GOARCH)"
	@echo ""
	@echo "=== Docker Information ==="
	@echo "Image:      $(DOCKER_FULL_IMAGE)"
	@echo "Tag:        $(DOCKER_TAG)"

# ============================================================================
# Help target
# ============================================================================

.PHONY: help
help: ## Display this help message
	@echo "NisFix Backend - Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
