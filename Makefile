.PHONY: dev build build-embed run run-embed clean tidy test test-unit test-generator generate-routes

# ==============================================================================
# DEVELOPMENT
# ==============================================================================

# Run the development server with live-reloading for Go and Tailwind CSS.
dev: 
	@overmind start

# ==============================================================================
# BUILD
# ==============================================================================

# Build the Go binary and the production CSS with disk-based templates (default).
build: 
	@echo "Building Go binary with disk-based templates..."
	@go build -o ./tmp/goby ./cmd/server
	@echo "Building production assets..."
	@npm run build:js
	@npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify

# Build the Go binary and the production CSS with embedded templates.
build-embed: generate-embed
	@echo "Building Go binary with embedded templates..."
	@go build -o ./tmp/goby-embed ./cmd/server
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

# Run the application with disk-based templates (default)
run: build
	@echo "Starting server with disk-based templates..."
	@./tmp/goby

# Run the application with embedded templates
run-embed: build-embed
	@echo "Starting server with embedded templates..."
	@APP_TEMPLATES=embed ./tmp/goby-embed

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
