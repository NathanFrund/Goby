.PHONY: dev build build-embed run run-embed clean tidy test test-unit

# ==============================================================================
# DEVELOPMENT
# ==============================================================================

# Run the development server with live-reloading for Go and Tailwind CSS.
dev:
	@overmind start

# ==============================================================================
# BUILD
# ==============================================================================

# Build the Go binary and the production CSS.
build:
	@echo "Building Go binary..."
	@go build -o ./tmp/goby ./cmd/server
	@echo "Building production assets..."
	@npm run build:js
	@npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify

# Build the Go binary with embedded templates enabled at build time.
build-embed:
	@echo "Building Go binary with embedded templates..."
	@go build -ldflags "-X 'main.AppTemplates=embed'" -o ./tmp/goby ./cmd/server
	@echo "Building production assets..."
	@npm run build:js
	@npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify

# ==============================================================================
# RUN
# ==============================================================================

# Run the application with templates loaded from disk.
run:
	@echo "Running with disk templates..."
	@APP_TEMPLATES=disk go run ./cmd/server

# Run the application with embedded templates
run-embed:
	@echo "Running with embedded templates..."
	@APP_TEMPLATES=embed go run ./cmd/server

# ==============================================================================
# HELPERS
# ==============================================================================

# Remove build artifacts.
clean:
	@rm -rf ./tmp ./coverage.* ./.tailwindcss

# Tidy go.mod and go.sum.
tidy:
	@go mod tidy

# ==============================================================================
# TESTING
# ==============================================================================

# Run all tests
test: test-unit

# Run unit tests
test-unit:
	go test ./... -v
