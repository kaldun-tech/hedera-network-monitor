# Phase 2: Storage & Performance Action Plan

## Current State
✅ **Phase 1 Complete:**
- State tracking bug fixed
- Structured logging implemented
- Code refactored to use MetricState struct

## Phase 2 Overview: Storage & Performance 🟡

### Goals
1. Improve memory storage with LRU eviction
2. Add persistent storage (PostgreSQL)
3. Add observability (Prometheus metrics)

---

## Task Breakdown

### **Task 2.1: Implement LRU Cache for Memory Storage**
**Priority:** Medium
**Effort:** 1-2 days
**Current Issue:** Simple FIFO eviction (memory.go:64) - removes oldest metric regardless of usage

**Action Items:**
1. Add LRU tracking to MemoryStorage:
   ```go
   type MemoryStorage struct {
       metrics   map[string]*list.Element // key -> list element
       lruList   *list.List               // doubly-linked list for LRU
       maxSize   int
       mu        sync.RWMutex
   }
   ```

2. Update `StoreMetric()` to:
   - Move accessed items to front of list
   - Evict from back when full (least recently used)

3. Benefits:
   - ✅ Keeps frequently queried metrics in memory longer
   - ✅ Better cache hit rate for dashboards
   - ✅ More predictable performance

**Files to modify:**
- `internal/storage/memory.go`

**Testing:**
- Unit tests for LRU behavior
- Benchmark tests (before/after)

---

### **Task 2.2: PostgreSQL Storage Backend**
**Priority:** High
**Effort:** 3-5 days
**Current Issue:** In-memory only - data lost on restart, no historical queries

**Action Items:**

#### **Step 1: Design Schema** (4 hours)
```sql
CREATE TABLE metrics (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    timestamp BIGINT NOT NULL,
    value DOUBLE PRECISION NOT NULL,
    labels JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    INDEX idx_name_timestamp (name, timestamp DESC),
    INDEX idx_labels (labels) USING GIN
);

CREATE TABLE metric_labels (
    metric_id BIGINT REFERENCES metrics(id),
    key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (metric_id, key)
);
```

#### **Step 2: Implement PostgresStorage** (2 days)
1. Create `internal/storage/postgres.go`:
   ```go
   type PostgresStorage struct {
       db     *sql.DB
       config PostgresConfig
   }

   func NewPostgresStorage(config PostgresConfig) (*PostgresStorage, error)
   func (ps *PostgresStorage) StoreMetric(metric types.Metric) error
   func (ps *PostgresStorage) GetMetrics(name string, limit int) ([]types.Metric, error)
   // ... implement all Storage interface methods
   ```

2. Add to `pkg/config/config.go`:
   ```go
   type StorageConfig struct {
       Type     string // "memory" or "postgres"
       Postgres PostgresConfig
   }

   type PostgresConfig struct {
       Host     string
       Port     int
       Database string
       User     string
       Password string
       SSLMode  string
   }
   ```

#### **Step 3: Add Database Migrations** (1 day)
- Use `golang-migrate/migrate` or similar
- Version schema changes
- Create initial migration files

#### **Step 4: Update cmd/monitor/main.go** (2 hours)
```go
var store storage.Storage
switch cfg.Storage.Type {
case "postgres":
    store, err = storage.NewPostgresStorage(cfg.Storage.Postgres)
case "memory":
    store = storage.NewMemoryStorage()
default:
    return fmt.Errorf("unknown storage type: %s", cfg.Storage.Type)
}
```

**Dependencies:**
- `github.com/lib/pq` (PostgreSQL driver)
- `github.com/golang-migrate/migrate/v4` (migrations)

**Files to create:**
- `internal/storage/postgres.go`
- `internal/storage/postgres_test.go`
- `migrations/000001_create_metrics_table.up.sql`
- `migrations/000001_create_metrics_table.down.sql`

**Files to modify:**
- `pkg/config/config.go`
- `cmd/monitor/main.go`
- `config.example.yaml`

**Testing:**
- Integration tests with test database
- Migration up/down tests
- Performance tests (bulk inserts)

---

### **Task 2.3: Prometheus Metrics Export**
**Priority:** Medium
**Effort:** 2-3 days
**Current Issue:** No way to integrate with existing monitoring tools

**Action Items:**

#### **Step 1: Add Prometheus Client** (1 day)
```go
// pkg/metrics/prometheus.go
type PrometheusExporter struct {
    registry *prometheus.Registry
    metrics  map[string]prometheus.Gauge
}

func NewPrometheusExporter() *PrometheusExporter
func (pe *PrometheusExporter) RecordMetric(metric types.Metric)
func (pe *PrometheusExporter) Handler() http.Handler
```

#### **Step 2: Add /metrics Endpoint** (4 hours)
Update `internal/api/server.go`:
```go
mux.HandleFunc("/metrics", promHandler.ServeHTTP)
```

#### **Step 3: Wire into Collectors** (4 hours)
Collectors push to both Storage AND Prometheus:
```go
// In collector
if err := store.StoreMetric(metric); err != nil {
    logger.Error("Error storing metric", "error", err)
}
promExporter.RecordMetric(metric) // Also export to Prometheus
```

**Dependencies:**
- `github.com/prometheus/client_golang/prometheus`

**Files to create:**
- `pkg/metrics/prometheus.go`
- `pkg/metrics/prometheus_test.go`

**Files to modify:**
- `internal/api/server.go`
- `internal/collector/account.go`
- `internal/collector/network.go`
- `cmd/monitor/main.go`

**Benefits:**
- ✅ Grafana dashboards
- ✅ Standard metrics format
- ✅ Alert integration with Prometheus AlertManager

---

## Recommended Order

### **Week 1: PostgreSQL (Highest Value)**
- Days 1-2: Schema design + postgres.go implementation
- Days 3-4: Migrations + configuration
- Day 5: Testing + integration

### **Week 2: Prometheus (Integration)**
- Days 1-2: Prometheus exporter
- Day 3: Wire into collectors
- Days 4-5: Testing + documentation

### **Week 3: LRU Cache (Optimization)**
- Days 1-2: Implement LRU
- Day 3: Benchmarking
- Days 4-5: Polish + code review

---

## Success Criteria

**PostgreSQL:**
- ✅ Data persists across restarts
- ✅ Can query historical metrics (last 30 days)
- ✅ < 100ms p95 latency for writes
- ✅ All Storage interface methods implemented

**Prometheus:**
- ✅ /metrics endpoint works
- ✅ Metrics appear in Prometheus
- ✅ Can create Grafana dashboard
- ✅ No performance degradation

**LRU Cache:**
- ✅ Cache hit rate > 80% for typical queries
- ✅ Evicts least recently used items
- ✅ Benchmark shows improvement

---

## Configuration Example (Phase 2)

```yaml
storage:
  type: postgres  # "memory" or "postgres"
  postgres:
    host: localhost
    port: 5432
    database: hedera_monitor
    user: monitor
    password: ${DB_PASSWORD}
    sslmode: require
    max_connections: 25

prometheus:
  enabled: true
  namespace: hedera
  subsystem: monitor
```

---

## Notes

- All changes should maintain backward compatibility with existing memory storage
- PostgreSQL should be optional - memory storage remains the default
- Each task should have comprehensive tests before merging
- Update CLAUDE.md and README.md as features are added
