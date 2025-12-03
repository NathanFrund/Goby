.PHONY: help dev build build-embed run run-embed clean tidy test test-unit test-generator generate-routes install-cli new-module

# Default target - show help
.DEFAULT_GOAL := help

# ==============================================================================
# HELP
# ==============================================================================

# Show this help message
help:
	@echo "Goby Framework - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev              Run development server with hot-reloading"
	@echo "  make install-cli      Build and install goby-cli tool"
	@echo "  make new-module NAME=<name>  Create a new module"
	@echo ""
	@echo "Building:"
	@echo "  make build            Build production binary and assets"
	@echo "  make run              Build and run the application"
	@echo ""
	@echo "Testing:"
	@echo "  make test             Run all tests"
	@echo "  make test-unit        Run unit tests only"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean            Remove build artifacts"
	@echo "  make tidy             Tidy go.mod and go.sum"
	@echo ""

# ==============================================================================
# DEVELOPMENT
# ==============================================================================

# Run the development server with live-reloading for Go and Tailwind CSS.
dev: 
	@ENV=development overmind start

# Build and install the CLI tool
install-cli:
	@echo "Building and installing goby-cli..."
	@go install ./cmd/goby-cli
	@echo "✓ goby-cli installed successfully"

# Create a new module using the CLI tool
new-module:
ifndef NAME
	$(error NAME is required. Usage: make new-module NAME=mymodule)
endif
	@go run ./cmd/goby-cli new-module --name=$(NAME)

# ==============================================================================
# BUILD
# ==============================================================================

# Build the Go binary and production assets
build: 
	@echo "Building Go binary..."
	@ENV=production go build -o ./tmp/goby ./cmd/server
	@echo "Building production assets..."
	@npm run build:js
	@npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify

# Generate embedded assets
.PHONY: generate-embed
generate-embed:
	@echo "Generating embedded assets..."
	@go generate ./...

# ==============================================================================
# RUN
# ==============================================================================

# Run the application
run: build
	@echo "Starting server..."
	@./tmp/goby

# ==============================================================================
# HELPERS
# ==============================================================================

# Remove build artifacts.
clean:
	@rm -rf ./tmp ./coverage.* ./.tailwindcss
	@find . -name "*.generated.go" -type f -delete

# Tidy go.mod and go.sum.
tidy:
	@go mod tidy

# ==============================================================================
# TESTING
# ==============================================================================

# Run all tests
test: test-unit test-generator

# Run unit tests
test-unit:
	go test ./... -v

# Test the route generator
test-generator:
	cd internal/tools/genroutes && go test -v
