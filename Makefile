.PHONY: build test clean

# Build variables
BINARY_NAME=gochess
GO=go
GOFLAGS=-v

# Build the application
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/gochess

# Run all tests
test:
	$(GO) test $(GOFLAGS) ./...

# Run tests with coverage
test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out
	rm -f *.prof

# Format code
fmt:
	$(GO) fmt ./...

# Run linter
lint:
	$(GO) vet ./...

# Install dependencies
deps:
	$(GO) mod tidy

# All targets for CI
ci: deps fmt lint test build
