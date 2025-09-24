.PHONY: dev build clean tidy test test-unit

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
