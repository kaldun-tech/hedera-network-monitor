.PHONY: help build build-monitor build-cli test test-unit test-integration test-coverage lint fmt clean run run-monitor run-cli

# Variables
GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOCLEAN := $(GO) clean
GOGET := $(GO) get
GOMOD := $(GO) mod
GOLINT := golangci-lint

MONITOR_OUTPUT := monitor
CLI_OUTPUT := hmon

# Build flags
LDFLAGS := -ldflags="-X main.Version=0.1.0"

# Default target
help:
	@echo "Hedera Network Monitor - Makefile targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build both monitor and CLI"
	@echo "  make build-monitor      - Build monitor service only"
	@echo "  make build-cli          - Build CLI tool only"
	@echo ""
	@echo "Development targets:"
	@echo "  make fmt                - Format code with gofmt"
	@echo "  make lint               - Run linter (requires golangci-lint)"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run unit tests only (fast, ~4s)"
	@echo "  make test-integration   - Run integration tests only (slow, ~30s)"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make clean              - Remove build artifacts"
	@echo ""
	@echo "Run targets:"
	@echo "  make run-monitor        - Build and run monitor service"
	@echo "  make run-cli            - Run CLI tool (requires arguments)"
	@echo ""
	@echo "Dependency targets:"
	@echo "  make deps-download      - Download dependencies"
	@echo "  make deps-update        - Update dependencies"

# Build targets
build: build-monitor build-cli
	@echo "✓ Build complete"

build-monitor:
	@echo "Building monitor service..."
	$(GOBUILD) $(LDFLAGS) -o $(MONITOR_OUTPUT) ./cmd/monitor

build-cli:
	@echo "Building CLI tool..."
	$(GOBUILD) $(LDFLAGS) -o $(CLI_OUTPUT) ./cmd/hmon

# Development targets
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "✓ Code formatted"

lint:
	@echo "Running linter..."
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	$(GOLINT) run ./...
	@echo "✓ Linting complete"

test-unit:
	@echo "Running unit tests (excluding integration tests)..."
	$(GOTEST) -v -race ./...
	@echo "✓ Unit tests complete"

test-integration:
	@echo "Running integration tests (slow, may take 30+ seconds)..."
	$(GOTEST) -v -race -tags integration ./...
	@echo "✓ Integration tests complete"

test: test-unit test-integration
	@echo "✓ All tests complete"

test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated (coverage.html)"

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(MONITOR_OUTPUT) $(CLI_OUTPUT)
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

# Run targets
run-monitor: build-monitor
	@echo "Starting monitor service..."
	./$(MONITOR_OUTPUT)

run-cli: build-cli
	@echo "Running CLI tool..."
	./$(CLI_OUTPUT) $(ARGS)

# Dependency targets
deps-download:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	@echo "✓ Dependencies downloaded"

deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy
	@echo "✓ Dependencies updated"

# Development setup
setup-dev:
	@echo "Setting up development environment..."
	@echo "Installing golangci-lint..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Development environment ready"

# Verify all checks (fast unit tests only)
verify: fmt lint test-unit
	@echo "✓ All checks passed"

# Quick build and test (fast unit tests only)
quick: build test-unit
	@echo "✓ Quick check complete"

# CI/CD target - strict checks (includes integration tests)
ci: deps-download fmt lint test-coverage build
	@echo "✓ CI checks complete"

# Docker targets (optional)
docker-build:
	@echo "Building Docker image..."
	docker build -t hedera-network-monitor:latest .
	@echo "✓ Docker image built"

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -v $(PWD)/config.yaml:/config.yaml hedera-network-monitor:latest

# Stats target
stats:
	@echo "Code statistics:"
	@find . -name "*.go" -not -path "./vendor/*" | xargs wc -l | tail -1
	@echo ""
	@echo "File count:"
	@find . -name "*.go" -not -path "./vendor/*" | wc -l

.PHONY: all ci setup-dev verify quick stats docker-build docker-run
