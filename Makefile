.PHONY: dev build clean tidy test

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
	@echo "Building production CSS..."
	@npm exec tailwindcss -- -i ./web/src/css/input.css -o ./web/static/css/style.css --minify

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
test:
	go test ./... -v
