# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**hedera-network-monitor** is a comprehensive Go-based monitoring and alerting tool for the Hedera blockchain network. The project provides both a long-running monitoring service (daemon) and a CLI tool for queries and configuration.

## Architecture & Design Patterns

### High-Level Architecture
The project uses a modular, interface-based design:

```
Monitor Service
├── Collectors (concurrent, extensible)
│   ├── Account collector (balance, transactions)
│   └── Network collector (node status, health)
├── Storage (pluggable backends)
│   └── Memory (MVP), future: PostgreSQL, InfluxDB, Prometheus
├── Alert Manager (rule evaluation, webhook dispatch)
└── HTTP API Server (:8080)

CLI Tool
├── Direct Hedera SDK calls (real-time queries)
└── Service API calls (historical data)
```

### Key Design Principles
1. **Interface-based**: Storage, Collectors, Client all use interfaces for pluggability
2. **Concurrent**: Uses goroutines and channels for parallel metric collection
3. **Graceful shutdown**: Context-based cancellation throughout
4. **Error handling**: All functions return explicit errors, no panics
5. **Testable**: Pure functions and dependency injection where possible

## Project Structure

```
hedera-network-monitor/
├── cmd/
│   ├── monitor/main.go          # Service daemon entry point
│   │   └── Manages collectors, storage, alerts, API server
│   │   └── Graceful shutdown via signal handling
│   │   └── Uses errgroup for goroutine management
│   │
│   └── hmon/main.go             # CLI tool entry point
│       └── Cobra-based command structure
│       └── Query service API or direct Hedera SDK
│
├── internal/
│   ├── collector/
│   │   ├── collector.go         # Collector interface & base
│   │   ├── account.go           # Account balance/transaction collector
│   │   └── network.go           # Network status collector
│   │
│   ├── alerting/
│   │   ├── manager.go           # Rule evaluation, webhook dispatch
│   │   ├── rules.go             # AlertRule & AlertEvent types
│   │   ├── webhook.go           # Webhook sender (TODO: HTTP implementation)
│   │   └── errors.go            # Error definitions
│   │
│   ├── storage/
│   │   ├── storage.go           # Storage interface
│   │   └── memory.go            # In-memory implementation (10k metrics)
│   │
│   └── api/
│       ├── server.go            # HTTP server with handlers
│       └── handlers.go          # Future: handler organization
│
├── pkg/
│   ├── hedera/
│   │   └── client.go            # Hedera SDK wrapper (TODO: implement)
│   │
│   ├── metrics/
│   │   └── metrics.go           # Aggregation: Average, Min, Max, Count
│   │
│   └── config/
│       └── config.go            # Viper-based config loading & validation
│
├── config.example.yaml          # Complete example with documentation
├── go.mod / go.sum              # Module definition & checksums
├── Makefile                     # Build, test, lint commands
├── README.md                    # Comprehensive documentation
└── LICENSE                      # MIT
```

## Common Commands

### Build & Run

```bash
# Build both service and CLI
make build

# Build individual components
make build-monitor              # Service daemon only
make build-cli                  # CLI tool only

# Run service with default config
./monitor

# Run service with custom config
./monitor --config /path/to/config.yaml --loglevel debug

# Run CLI tool
./hmon account balance 0.0.5000
./hmon --api-url http://localhost:8080 network status
```

### Development

```bash
# Format code
make fmt

# Lint code (requires golangci-lint)
make lint

# Run tests
make test

# Run tests with coverage
make test-coverage

# Clean build artifacts
make clean

# Setup development environment
make setup-dev

# Run all checks (fmt, lint, test)
make verify
```

## Configuration

Configuration is managed via `config.yaml` (see `config.example.yaml`):

```yaml
# Service will load on startup
network:
  name: testnet                   # "mainnet" or "testnet"
  operator_id: "0.0.1234"
  operator_key: "private_key"

accounts:
  - id: "0.0.5000"
    label: "Main Account"

alerting:
  enabled: true
  webhooks:
    - "https://hooks.slack.com/services/..."
  rules:
    - id: "balance_low"
      metric_name: "account_balance"
      condition: "<"
      threshold: 1000000000        # in tinybar
      severity: "warning"

api:
  port: 8080
  host: "localhost"
```

## Key Implementation Details

### Metrics System
- **Type**: `collector.Metric` with Name, Timestamp, Value, Labels
- **Flow**: Collectors → Storage → API / Alert Manager
- **Labels**: Used for filtering (account_id, label, etc.)

### Alert Rules
- **Evaluation**: `AlertManager.CheckMetric()` runs on each collected metric
- **Conditions**: ">", "<", "==", "changed", "decreased", "increased" (TODO)
- **Cooldowns**: 5-minute default to prevent spam (configurable)
- **Webhooks**: Async dispatch with error logging (TODO: retry logic)

### Storage Interface
```go
type Storage interface {
    StoreMetric(metric Metric) error
    GetMetrics(name string, limit int) ([]Metric, error)
    GetMetricsByLabel(key, value string) ([]Metric, error)
    DeleteOldMetrics(beforeTimestamp int64) error
    Close() error
}
```

## Common Development Tasks

### Adding a New Collector
1. Create `internal/collector/your_collector.go`
2. Implement `Collector` interface
3. Register in `cmd/monitor/main.go` collectors slice
4. Example: See `account.go` and `network.go`

### Adding API Endpoints
1. Add handler in `internal/api/server.go` or `handlers.go`
2. Register route: `mux.HandleFunc("/path", handler)`
3. Return JSON responses
4. Document in README.md

### Implementing Hedera SDK Calls
1. Update `pkg/hedera/client.go` methods
2. Use `hashgraph/hedera-sdk-go/v2` SDK
3. Handle context cancellation
4. Return structured responses

### Adding Configuration Options
1. Update `Config` struct in `pkg/config/config.go`
2. Add to `config.example.yaml` with documentation
3. Add Viper default in `Load()` function
4. Validate in `Validate()` method

## Dependencies

Key dependencies (in `go.mod`):
- `github.com/hashgraph/hedera-sdk-go/v2` - Hedera blockchain SDK
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `golang.org/x/sync/errgroup` - Goroutine management

## Code Style & Conventions

- **Package names**: lowercase, concise (e.g., `collector`, `alerting`)
- **Exported symbols**: PascalCase (e.g., `Collector`, `AlertManager`)
- **Unexported symbols**: camelCase (e.g., `baseCollector`)
- **Interfaces**: End with "er" where appropriate (e.g., `Collector`, `Storage`)
- **Error handling**: Always return errors, don't panic
- **Logging**: Use standard `log` package (TODO: consider structured logging)
- **Comments**: Explain "why", not "what"
- **TODO comments**: Mark future work with `// TODO: [description]`

## Testing

Standard Go testing patterns:
```bash
go test ./...                     # Run all tests
go test -v ./internal/storage    # Verbose, single package
go test -run TestMetric ./...     # Run specific test
go test -cover ./...              # Show coverage percentage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # HTML coverage report
```

## Important Notes

- **No hardcoded secrets**: Config file and environment variables only
- **Graceful shutdown**: All services respond to context cancellation
- **Error handling**: No silent failures - log and propagate errors
- **Metrics**: All collectors follow the same `Metric` type
- **Concurrency**: Safe use of goroutines, channels, and mutex where needed
- **API compatibility**: REST API follows `/api/v1/` versioning

## TODO Items in Code

Key TODOs to implement (marked throughout codebase):

**High Priority:**
- Implement Hedera SDK client methods (`pkg/hedera/client.go`)
- Implement webhook sending with retry logic
- Complete API handlers with proper JSON marshaling
- Add alert rule condition DSL evaluation
- Implement graceful metric storage eviction policy

**Medium Priority:**
- Add structured logging (consider `logrus` or similar)
- Implement PostgreSQL storage backend
- Add Prometheus metrics export
- Implement transaction history tracking
- Add WebSocket API for real-time metrics

**Low Priority:**
- Web UI dashboard
- Advanced alerting DSL
- Rule template marketplace
- Multi-network support enhancements

## Useful Commands

```bash
# Check Go version
go version

# Tidy dependencies
go mod tidy

# Check for security issues (requires gosec)
gosec ./...

# Find all TODOs in code
grep -r "TODO" --include="*.go" ./

# Check code statistics
find . -name "*.go" -not -path "./vendor/*" | xargs wc -l

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o monitor ./cmd/monitor
GOOS=darwin GOARCH=amd64 go build -o monitor ./cmd/monitor
GOOS=windows GOARCH=amd64 go build -o monitor.exe ./cmd/monitor
```

## Troubleshooting

- **Build fails**: Run `go mod download && go mod tidy`
- **Linter complains**: Run `make fmt` first, then address remaining issues
- **Tests fail**: Check configuration file exists, network settings valid
- **Service won't start**: Verify `config.yaml` exists and is valid YAML
- **CLI connection refused**: Ensure service is running on correct port

## Quick Start

```bash
# 1. Copy and edit configuration
cp config.example.yaml config.yaml
nano config.yaml                # Set your network details

# 2. Build the project
make build

# 3. Run the service
./monitor

# 4. In another terminal, use CLI
./hmon account balance 0.0.5000

# 5. Query API
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/metrics
```

## References

- README.md - Complete project documentation
- LICENSE - MIT License
- go.mod - Go module with dependencies
