# Futures Engine Makefile

# Variables
BINARY_NAME=futures_engine
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "v1.0.0")
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	go run .

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Release builds for multiple platforms
.PHONY: release
release: clean
	@echo "Building release versions..."
	@mkdir -p $(BUILD_DIR)/release
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 .
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 .
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe .
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 .
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 .
	
	@echo "Release builds complete in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, running go vet instead..."; \
		go vet ./...; \
	fi

# Check code quality
.PHONY: check
check: fmt lint test

# Development target - run with file watching
.PHONY: dev
dev:
	@echo "Starting development mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed, using go run instead..."; \
		go run .; \
	fi

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  release        - Build release versions for multiple platforms"
	@echo "  clean          - Clean build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code"
	@echo "  check          - Run fmt, lint, and test"
	@echo "  dev            - Run in development mode with file watching"
	@echo "  help           - Show this help message"