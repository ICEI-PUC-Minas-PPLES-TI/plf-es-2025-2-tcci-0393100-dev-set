.PHONY: help build test test-coverage clean install run fmt lint vet mod-tidy

# Variables
BINARY_NAME=set
BINARY_PATH=./bin/$(BINARY_NAME)
VERSION?=0.1.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS=-ldflags "-X 'set/cmd.version=$(VERSION)' -X 'set/cmd.buildTime=$(BUILD_TIME)' -X 'set/cmd.gitCommit=$(GIT_COMMIT)'"

help: ## Display this help screen
	@echo "SET CLI - Makefile Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o $(BINARY_PATH) main.go
	@echo "✓ Binary built: $(BINARY_PATH)"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 main.go
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 main.go
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe main.go
	@echo "✓ All binaries built in ./bin/"

test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report, run: go tool cover -html=coverage.out"

test-coverage-html: test-coverage ## Generate HTML coverage report
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

run: build ## Build and run the CLI
	@$(BINARY_PATH)

install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	@golangci-lint run
	@echo "✓ Linting complete"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

mod-tidy: ## Tidy go modules
	@echo "Tidying modules..."
	@go mod tidy
	@echo "✓ Modules tidied"

mod-download: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@echo "✓ Dependencies downloaded"

check: fmt vet test ## Run formatting, vetting, and tests

dev: ## Run in development mode
	@go run main.go

# Configuration commands
configure: build ## Run interactive configuration
	@$(BINARY_PATH) configure --initial

configure-list: build ## List current configuration
	@$(BINARY_PATH) configure --list

# Version information
version: build ## Show version information
	@$(BINARY_PATH) version

# Development helpers
watch: ## Watch for changes and rebuild (requires entr)
	@which entr > /dev/null || (echo "entr not installed. Install from http://eradman.com/entrproject/" && exit 1)
	@find . -name '*.go' | entr -r make run

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	@godoc -http=:6060 &
	@echo "✓ Documentation server running at http://localhost:6060"
	@echo "  Press Ctrl+C to stop"

# Git hooks
install-hooks: ## Install git hooks
	@echo "Installing git hooks..."
	@echo '#!/bin/sh\nmake check' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✓ Git hooks installed"

# Default target
all: clean check build ## Clean, check, and build
