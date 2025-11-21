# Learning Document - Hedera Network Monitor

This document captures interesting patterns, discoveries, and best practices discovered while building the Hedera Network Monitor project.

---

## Table of Contents

1. [Go Routines and Concurrency](#go-routines-and-concurrency)
   - [Goroutines vs Threading in Other Languages](#goroutines-vs-threading-in-other-languages)
2. [Context and Cancellation](#context-and-cancellation)
3. [Error Handling Patterns](#error-handling-patterns)
4. [Interface-Based Design](#interface-based-design)
5. [Testing Patterns](#testing-patterns)
6. [Go JSON Serialization](#go-json-serialization)
7. [Hedera SDK Discoveries](#hedera-sdk-discoveries)

---

## Go Routines and Concurrency

### What Are Goroutines?

Goroutines are **lightweight threads managed by the Go runtime**. They're not OS threads - the runtime multiplexes many goroutines onto a smaller number of OS threads. This makes them extremely cheap to create and manage.

**Key characteristics:**
- Much lighter weight than OS threads (you can safely spawn thousands)
- Managed by the Go runtime scheduler
- Communicate via channels
- Can be paused/resumed by the runtime

### Goroutines vs Threading in Other Languages

Go's goroutine design was intentionally created to address frustrations Google engineers experienced with threading in C++, Java, and Python. Here's how they compare:

#### Java Threads

**Problem Java solved (but created new ones):**
- Java threads are OS-level threads wrapped by the JVM
- Creating threads is expensive (each one reserves significant memory - ~1MB stack)
- Context switching between threads is costly
- Result: Developers limit thread count, making concurrency harder

**Go's solution:**
```go
// Java: You create 1000 threads, you get 1000 OS threads (expensive!)
// Go: You create 10,000 goroutines, they run on maybe 4 OS threads (cheap!)

for i := 0; i < 10000; i++ {
    go func() {
        // Lightweight concurrent operation
    }()
}
```

**Key difference:** Java threads are 1:1 mapped to OS threads. Go goroutines are M:N scheduled (many goroutines on fewer OS threads).

#### C++ Threads

**Problem C++ had:**
- Manual thread management with `std::thread` requires careful synchronization
- No built-in concurrency primitives (had to use mutexes, condition variables, or external libraries)
- Developers often avoided threading due to complexity
- Data races and deadlocks are common and hard to debug

**Go's solution:**
```go
// C++: Manual synchronization nightmare
// std::mutex, condition_variable, std::lock_guard...

// Go: Simple, safe channel communication
resultChan := make(chan int)
go func() {
    resultChan <- computeExpensiveValue()
}()
result := <-resultChan  // Wait for result - clean, deadlock-free
```

**Key difference:** Go provides first-class primitives for concurrent communication (channels), not just low-level synchronization. This encourages safe communication patterns.

#### Python Threads (and the GIL)

**Problem Python had:**
- Global Interpreter Lock (GIL) prevents true parallelism
- Multiple threads can't execute Python bytecode simultaneously
- Threading is nearly useless for CPU-bound work
- Developers had to use multiprocessing (separate processes, expensive)

**Go's solution:**
```python
# Python: Can't use threads for CPU work due to GIL
# Must use multiprocessing (heavy, separate processes)

# Go: True parallelism without extra overhead
import "runtime"

func init() {
    runtime.GOMAXPROCS(runtime.NumCPU())  // Use all cores
}

for i := 0; i < runtime.NumCPU(); i++ {
    go cpuIntensiveTask()  // Each runs in parallel on different core
}
```

**Key difference:** Go doesn't have a GIL - multiple goroutines can execute Go code in parallel on multiple CPU cores.

#### The Google Design Philosophy

Google engineers (Rob Pike, Ken Thompson, Robert Griesemer) created Go specifically because:

1. **C++ was too complex for concurrent systems** → Go made concurrency simple and safe
2. **Java's thread model was too heavy** → Go made concurrency cheap
3. **Python's GIL prevented parallelism** → Go enabled true parallelism
4. **No language had good built-in concurrency** → Go made channels a language feature

**The result:** A language where spawning thousands of lightweight concurrent operations is not only possible but *encouraged*.

#### Comparison Table

| Feature | Go Goroutines | Java Threads | C++ Threads | Python Threads |
|---------|---------------|--------------|-------------|-----------------|
| **Creation cost** | Microseconds, ~2KB stack | Milliseconds, ~1MB stack | Expensive | Expensive |
| **Safe by default** | ✓ Yes (channels) | ⚠️ No (requires careful syncing) | ✗ No (manual sync) | ✓ With GIL (but limited) |
| **Parallelism** | ✓ True parallelism | ✓ True parallelism | ✓ True parallelism | ✗ GIL prevents it |
| **Can spawn 100k+** | ✓ Yes, easily | ✗ No (system limits) | ✗ No (system limits) | ✗ No (GIL overhead) |
| **Communication** | ✓ Channels (safe) | Shared memory (risky) | Shared memory (risky) | Queues (works around GIL) |
| **Complexity** | Simple | Medium-High | High | Medium |

#### Why This Matters for This Project

In this Hedera Monitor, we spawn:
- **1 goroutine** for API server
- **2 goroutines** for collectors (account, network)
- **1 goroutine** for alert manager
- **N goroutines** for webhook dispatch (one per webhook)

**In Java:** We'd need to manage 4-5 threads carefully, with shared state protection
**In C++:** We'd manage raw threads with mutexes and condition variables
**In Python:** Threading wouldn't work well for CPU tasks; we'd need multiprocessing
**In Go:** We trivially spawn lightweight goroutines and let the runtime handle scheduling

This is why Go shines for backend services - you can have thousands of concurrent operations (handling requests, collecting metrics, processing events) without the overhead and complexity of traditional threading models.

### Pattern 1: Using `errgroup.WithContext()` for Coordinated Goroutines

**Location:** `cmd/monitor/main.go:52-68`

```go
// Create an error group with context
eg, egCtx := errgroup.WithContext(ctx)

// Start API server in a goroutine
eg.Go(func() error {
    log.Printf("Starting API server on port %d", cfg.API.Port)
    return server.Start(egCtx)
})

// Start collectors in goroutines
for _, c := range collectors {
    coll := c  // IMPORTANT: Capture in local variable
    eg.Go(func() error {
        log.Printf("Starting collector: %s", coll.Name())
        return coll.Collect(egCtx, store, alertManager)
    })
}

// Start alert manager in a goroutine
eg.Go(func() error {
    log.Println("Starting alert manager")
    return alertManager.Run(egCtx)
})

// Wait for all goroutines to complete
if err := eg.Wait(); err != nil {
    log.Printf("Service error: %v", err)
}
```

**Why this is excellent:**

1. **Coordinated lifecycle** - `errgroup.WithContext()` ties all goroutines together
2. **Error propagation** - If ANY goroutine returns an error, `eg.Wait()` captures it
3. **Graceful shutdown** - When the main context is cancelled, all goroutines get `egCtx.Done()`
4. **Loop variable capture** - Notice `coll := c` before the closure - this prevents all goroutines from using the same loop variable!

### Pattern 2: The Loop Variable Capture Bug

**The Problem:**

```go
// ❌ WRONG - All goroutines reference the SAME loop variable
for _, c := range collectors {
    eg.Go(func() error {
        return coll.Collect(egCtx, store, alertManager)  // coll is the SAME for all!
    })
}
// By the time goroutines run, the loop has finished
// and 'coll' points to the LAST item in the slice
```

**The Solution:**

```go
// ✓ CORRECT - Capture the loop variable in a local variable
for _, c := range collectors {
    coll := c  // Create a new variable for each iteration
    eg.Go(func() error {
        return coll.Collect(egCtx, store, alertManager)  // Each closure has its own 'coll'
    })
}
```

**Why this happens:** Goroutines don't run immediately - they're scheduled to run later. By the time they execute, the loop variable `c` has changed.

### Pattern 3: Parallel Webhook Dispatch

**Location:** `internal/alerting/manager.go:180-183`

```go
// Send to webhooks in parallel using goroutines
for _, webhook := range m.webhooks {
    go m.sendWebhook(webhook, alert)  // Fire and forget
}
```

**Characteristics:**
- Uses `go` keyword for fire-and-forget goroutines
- No error handling (intentional - logging is in sendWebhook)
- No waiting - parent continues immediately
- Each webhook sent in parallel

**When to use fire-and-forget:**
- Webhook dispatch (non-critical side effects)
- Logging operations
- Cleanup tasks
- When you don't need to know if the operation succeeded

### Pattern 4: Channel-Based Communication with `select`

**Location:** `internal/alerting/manager.go:170-184`

```go
for {
    select {
    case <-ctx.Done():
        log.Println("[AlertManager] Stopping alert processor")
        return ctx.Err()
    case alert := <-m.alertQueue:
        log.Printf("[AlertManager] Alert triggered: %s", alert.RuleName)
        for _, webhook := range m.webhooks {
            go m.sendWebhook(webhook, alert)
        }
    }
}
```

**What's happening:**
1. `select` waits for ANY channel to be ready
2. `<-ctx.Done()` - Context cancellation signal
3. `<-m.alertQueue` - Incoming alert
4. Whichever arrives first gets handled

**Channel types used:**
- `alertQueue chan AlertEvent` - Buffered channel (100 capacity)
- `ctx.Done()` - Cancellation signal
- Both are non-blocking readers - select checks them without waiting

### Key Goroutine Patterns Summary

| Pattern | Use Case | Safety | Example |
|---------|----------|--------|---------|
| `errgroup.WithContext()` | Coordinated work | ✓ Safe with context | Monitor service startup |
| Fire-and-forget `go` | Background tasks | ⚠️ No error handling | Webhook dispatch |
| Channels with `select` | Multiplexing | ✓ Safe communication | Alert queue processing |
| Mutex protection | Shared state | ✓ Serialized access | lastAlerts map |

---

## Context and Cancellation

### Why Context Matters

The `context.Context` is Go's mechanism for **cancellation, timeouts, and passing values**.

**Location:** `cmd/monitor/main.go:26-32`

```go
// Setup context with cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Setup signal handling for graceful shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// Wait for shutdown signal
go func() {
    sig := <-sigChan
    log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
    cancel()  // Cancel all contexts
}()
```

**What happens:**
1. Create a context that can be cancelled
2. When user presses Ctrl+C or sends SIGTERM
3. Call `cancel()` to signal all goroutines
4. Each goroutine's `<-ctx.Done()` receives the signal
5. All services shut down gracefully

**Key insight:** Context cancellation is **not** forcing goroutines to stop - it's a **signal** they can listen to. Good goroutines respect this signal and clean up.

---

## Error Handling Patterns

### Pattern 1: Error Propagation with `errgroup`

```go
eg, egCtx := errgroup.WithContext(ctx)

eg.Go(func() error {
    return server.Start(egCtx)  // Return error if something fails
})

// Wait collects the first error returned
if err := eg.Wait(); err != nil {
    log.Fatalf("Service error: %v", err)
}
```

**Benefits:**
- First error stops waiting
- Parent knows when something failed
- Clean error propagation

### Pattern 2: No Silent Failures

**Location:** `pkg/hedera/client.go:61-72`

```go
// Get operator credentials from environment
operatorID := os.Getenv("OPERATOR_ID")
operatorKey := os.Getenv("OPERATOR_KEY")

// Validate environment variables
if operatorID == "" || operatorKey == "" {
    return nil, fmt.Errorf("OPERATOR_ID and OPERATOR_KEY environment variables required")
}
```

**Key principle:** Never silently fail. Always return an error so the caller knows something went wrong.

### Pattern 3: Test Error Checking

**Discovery during linter fixes:** Using `t.Fatal()` vs `t.Error()`

```go
// ❌ WRONG - Continues execution even if nil
if config == nil {
    t.Error("expected config to not be nil")
}
if config.Network.Name != "testnet" {  // Crashes here!
    t.Errorf("expected testnet, got: %s", config.Network.Name)
}

// ✓ CORRECT - Stops test immediately on nil
if config == nil {
    t.Fatal("expected config to not be nil")  // Test stops here
}
if config.Network.Name != "testnet" {  // Safe to access
    t.Errorf("expected testnet, got: %s", config.Network.Name)
}
```

**When to use:**
- `t.Error()` - For assertion failures that don't prevent further testing
- `t.Fatal()` - For setup failures that would cause nil pointer dereference

---

## Interface-Based Design

### Why Interfaces Matter in This Project

**Location:** Multiple files - `storage.Storage`, `collector.Collector`, `hedera.Client`

```go
// Pluggable storage
type Storage interface {
    StoreMetric(metric Metric) error
    GetMetrics(name string, limit int) ([]Metric, error)
    GetMetricsByLabel(key, value string) ([]Metric, error)
    DeleteOldMetrics(beforeTimestamp int64) error
    Close() error
}

// Can be swapped without changing code that uses it
var store Storage
store = storage.NewMemoryStorage()  // MVP
// store = storage.NewPostgresStorage()  // Future
// store = storage.NewInfluxDBStorage()  // Future
```

**Benefits:**
1. **Testability** - Mock implementations for tests
2. **Extensibility** - New backends without changing logic
3. **Separation of concerns** - Business logic doesn't know about storage details

### Mutex for Thread Safety

**Location:** `internal/alerting/manager.go:19-23`

```go
type Manager struct {
    rules           []AlertRule
    ruleMutex       sync.RWMutex        // Protects access to rules
    lastAlerts      map[string]time.Time
    alertMutex      sync.Mutex           // Protects access to lastAlerts
}

// Protected read access
func (m *Manager) GetRules() []AlertRule {
    m.ruleMutex.RLock()        // Multiple readers
    defer m.ruleMutex.RUnlock()

    rules := make([]AlertRule, len(m.rules))
    copy(rules, m.rules)
    return rules
}

// Protected write access
func (m *Manager) AddRule(rule AlertRule) error {
    m.ruleMutex.Lock()         // Exclusive access
    defer m.ruleMutex.Unlock()

    m.rules = append(m.rules, rule)
    return nil
}
```

**Key insight:** Use `sync.RWMutex` when you have many readers and few writers. `sync.Mutex` for general protection.

---

## Testing Patterns

### Pattern 1: Mock Implementations

**Location:** `pkg/hedera/client_test.go:70-130`

```go
// MockClient implements the Client interface
type MockClient struct {
    mockBalance       int64
    mockInfo          *hiero.AccountInfo
    mockRecords       []Record
    getBalanceCalls   int  // Track call count
}

// Implements Client interface
func (m *MockClient) GetAccountBalance(accountID string) (int64, error) {
    m.getBalanceCalls++
    return m.mockBalance, nil
}
```

**Benefits:**
- Test without hitting Hedera network
- Verify methods are called correct number of times
- Control return values precisely

### Pattern 2: Table-Driven Tests

While not extensively used in this project, we could use them for condition evaluation:

```go
var tests = []struct {
    name      string
    value     float64
    threshold float64
    want      bool
}{
    {"greater", 100, 50, true},
    {"equal", 50, 50, false},  // > not >=
    {"less", 25, 50, false},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := tt.value > tt.threshold
        if got != tt.want {
            t.Errorf("got %v, want %v", got, tt.want)
        }
    })
}
```

---

## Go JSON Serialization

### Important Discovery: Exported Fields Are Capitalized in JSON

**Problem:** Script was looking for `"name"` but API returns `"Name"`

```go
type Metric struct {
    Name      string            // Exported = "Name" in JSON
    Timestamp int64             // Exported = "Timestamp" in JSON
    Value     float64           // Exported = "Value" in JSON
    Labels    map[string]string // Exported = "Labels" in JSON
}

// Produces JSON:
// {"Name":"account_balance","Timestamp":1234567890,"Value":1000,"Labels":{...}}
//   ^^^^  - Capitalized!
```

**Solution:** Control field names with struct tags

```go
type Metric struct {
    Name      string            `json:"name"`       // Output as "name"
    Timestamp int64             `json:"timestamp"`  // Output as "timestamp"
    Value     float64           `json:"value"`      // Output as "value"
    Labels    map[string]string `json:"labels"`     // Output as "labels"
}
```

**Rule:** In Go, only **exported** (capitalized) fields are included in JSON by default. Use struct tags to control the JSON key names.

---

## Hedera SDK Discoveries

### Discovery 1: Address Book File ID Required

**Problem:**
```
rpc error: code = InvalidArgument desc = getNodes.addressBookFilter.fileId: must not be null
```

**Solution:**
```go
// Address book is stored in file 0.0.102 on all Hedera networks
addressBookFileID, _ := hiero.FileIDFromString("0.0.102")
query := hiero.NewAddressBookQuery().
    SetFileID(addressBookFileID).
    SetMaxAttempts(5)
```

**Lesson:** Some Hedera SDK queries require additional parameters that aren't obvious from the API. Check the examples and official docs.

### Discovery 2: Constants for Conversions

```go
// In pkg/hedera/client.go
const TinybarPerHbar = 100_000_000

// Use it everywhere
hbarAmount := float64(tinybarAmount) / float64(hedera.TinybarPerHbar)
```

**Better than magic numbers like `100_000_000`**

### Discovery 3: SDK Handles Network Selection

```go
// SDK knows about testnet, mainnet, previewnet
client, err := hiero.ClientForName("testnet")
// Automatically configured with correct nodes
```

---

## Configuration and Defaults

### Pattern: Config with Environment Variable Fallback

**Location:** `cmd/hmon/main.go:204-220`

```go
func getCredentials() (operatorID, operatorKey string) {
    // Try to load from config file first
    if configFile != "" {
        cfg, err := config.Load(configFile)
        if err != nil {
            log.Printf("Failed to load config from %s: %v. Falling back to environment variables.", configFile, err)
        } else if cfg != nil {
            return cfg.Network.OperatorID, cfg.Network.OperatorKey
        }
    }

    // Fall back to environment variables
    operatorID = os.Getenv("OPERATOR_ID")
    operatorKey = os.Getenv("OPERATOR_KEY")
    return operatorID, operatorKey
}
```

**Advantages:**
1. Config file is preferred (easier to manage)
2. Environment variables as backup (CI/CD friendly)
3. Error logging helps debugging
4. No silent failures

---

## Defer and Cleanup

### Pattern: Deferred Cleanup

**Location:** `cmd/monitor/main.go:48-53`

```go
defer func(client *hiero.Client) {
    err := client.Close()
    if err != nil {
        log.Printf("Failed to close client: %v", err)
    }
}(client)
```

**Why this is good:**
1. **Guaranteed execution** - runs even if function panics
2. **Reverse order** - if multiple defers, they run in LIFO order
3. **Error handling** - can handle errors from cleanup
4. **RAII pattern** - Acquire resources, defer cleanup

**Alternative (simpler):**
```go
defer client.Close()  // Ignores errors
```

Use the verbose version when cleanup can fail.

---

## Key Takeaways

1. **Goroutines are cheap** - Spawn thousands if needed
2. **Context is essential** - Use it for cancellation and timeouts
3. **Interfaces enable flexibility** - Design for swappable implementations
4. **Channels for communication** - Use `select` for multiplexing
5. **Error groups for coordination** - Tie goroutines together with errgroup
6. **Mutexes for shared state** - Use RWMutex for read-heavy workloads
7. **Exported fields capitalize in JSON** - Use struct tags to control names
8. **Test with mocks** - Avoid network calls in unit tests
9. **Use Fatal() in tests** - When nil checks would crash
10. **Defer cleanup** - Always ensure resources are released

---

## Additional Resources

- **Concurrency in Go** - Read the chapter on goroutines and channels
- **Effective Go** - https://golang.org/doc/effective_go#concurrency
- **Context package** - https://pkg.go.dev/context
- **Errgroup package** - https://pkg.go.dev/golang.org/x/sync/errgroup

---

**Last Updated:** November 19, 2025
**Session:** Concurrency patterns and SDK discovery
