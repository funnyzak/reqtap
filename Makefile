# ReqTap Makefile

# Variable definitions
BINARY_NAME=reqtap
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Build directories
BUILD_DIR=build
DIST_DIR=dist

# Go related
GO_FILES=$(shell find . -name "*.go" -type f)
GO_MOD=$(shell go list -m)

# Platform list
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 linux/s390x linux/riscv64 linux/arm linux/ppc64le

.PHONY: help build build-all test test-coverage clean install deps lint fmt check

# Default target
help: ## Show help information
	@echo "ReqTap Build Tool"
	@echo ""
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependency management
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Code formatting
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

# Code checking
lint: ## Run code checking
	@echo "Running code checking..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping code checking"; \
	fi

# Building
build: ## Build current platform binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/reqtap

# Cross-compilation
build-all: ## Cross-compile all platforms
	@echo "Cross-compiling all platforms..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		echo "Building $$os/$$arch..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(DIST_DIR)/$$output_name ./cmd/reqtap; \
	done

# Testing
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

# Test coverage
test-coverage: ## Run tests and generate coverage report
	@echo "Running test coverage..."
	@mkdir -p coverage
	go test -v -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

# Install locally
install: ## Install to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/reqtap

# Run
run: build ## Build and run
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Clean
clean: ## Clean build files
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage

# Code checking
check: fmt lint test ## Run all checks (formatting, code checking, tests)

# Release preparation
release-prep: clean deps check build-all ## Prepare for release (clean, dependencies, checks, build all platforms)
	@echo "Release preparation complete!"
	@echo "Binary files location:"
	@ls -la $(DIST_DIR)/

# Create release packages
package: build-all ## Create release packages
	@echo "Creating release packages..."
	@mkdir -p $(DIST_DIR)/packages
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name=$(BINARY_NAME)-$$os-$$arch; \
		if [ $$os = "windows" ]; then output_name=$$output_name.exe; fi; \
		package_name=$(BINARY_NAME)-$(VERSION)-$$os-$$arch.tar.gz; \
		echo "Packaging $$package_name..."; \
		tar -czf $(DIST_DIR)/packages/$$package_name -C $(DIST_DIR) $$output_name; \
	done
	@echo "Release packages created: $(DIST_DIR)/packages/"

# Development mode
dev: ## Development mode run (monitor file changes)
	@echo "Starting development mode..."
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "air not installed, using normal run mode"; \
		go run $(LDFLAGS) ./cmd/reqtap; \
	fi

# Docker related
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -p 38888:38888 $(BINARY_NAME):latest

# Version information
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_DATE)"

# Quick start
quick-start: build ## Quick start (build and run example)
	@echo "Quick starting ReqTap..."
	@echo "Server will start at http://localhost:38888"
	@echo "Press Ctrl+C to stop"
	./$(BUILD_DIR)/$(BINARY_NAME) --log-level info