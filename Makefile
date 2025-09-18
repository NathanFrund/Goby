.PHONY: dev build clean tidy test generate-routes

# ==============================================================================
# DEVELOPMENT
# ==============================================================================

# Run the development server with live-reloading for Go and Tailwind CSS.
dev: generate-routes
	@overmind start

# ==============================================================================
# BUILD
# ==============================================================================

# Build the Go binary and the production CSS.
build: generate-routes
	@echo "Building Go binary..."
	@go build -o ./tmp/goby ./cmd/server
	@echo "Building production assets..."
	@npm run build:js
	@npm exec tailwindcss -- --input=./web/src/css/input.css --output=./web/static/css/style.css --minify

# ==============================================================================
# HELPERS
# ==============================================================================

# Generate route imports file
generate-routes:
	@echo "Generating route imports..."
	@go run internal/tools/genroutes/main.go -modules internal/modules -module github.com/nfrund/goby

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
