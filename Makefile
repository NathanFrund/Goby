.PHONY: test test-short test-cover test-watch

# Run all tests
test:
	go test ./... -v

# Run only unit tests (skip integration tests)
test-short:
	go test -short ./... -v

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Watch for changes and run tests (requires modd)
test-watch:
	modd
