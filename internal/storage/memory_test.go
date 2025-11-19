package storage

import (
	"os"
	"testing"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// mustStoreMetric stores a metric, failing the test fatally if it errors.
// This is used for test setup where metric storage failures indicate a broken test setup.
func mustStoreMetric(t *testing.T, storage *MemoryStorage, metric types.Metric) {
	t.Helper()
	if err := storage.StoreMetric(metric); err != nil {
		t.Fatalf("setup failed: could not store metric: %v", err)
	}
}

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	if storage == nil {
		t.Fatal("expected non-nil storage")
	}
	if storage.maxSize != DefaultMaxSize {
		t.Errorf("expected default max size %d, got %d", DefaultMaxSize, storage.maxSize)
	}
}

func TestNewMemoryStorage_WithEnvVar(t *testing.T) {
	// Save original env var
	origEnv := os.Getenv("COLLECTOR_MEMORY_MAX_SIZE")
	defer os.Setenv("COLLECTOR_MEMORY_MAX_SIZE", origEnv)

	// Set custom max size
	os.Setenv("COLLECTOR_MEMORY_MAX_SIZE", "5000")
	storage := NewMemoryStorage()

	if storage.maxSize != 5000 {
		t.Errorf("expected max size 5000, got %d", storage.maxSize)
	}
}

func TestStoreMetric_Single(t *testing.T) {
	storage := NewMemoryStorage()
	metric := types.Metric{
		Name:      "test_metric",
		Timestamp: time.Now().Unix(),
		Value:     42.0,
		Labels:    map[string]string{"test": "value"},
	}

	mustStoreMetric(t, storage, metric)

	// Verify metric was stored
	metrics, _ := storage.GetMetrics("test_metric", 0)
	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}
	if metrics[0].Value != 42.0 {
		t.Errorf("expected value 42.0, got %f", metrics[0].Value)
	}
}

func TestStoreMetric_Multiple(t *testing.T) {
	storage := NewMemoryStorage()

	// Store multiple metrics
	for i := 0; i < 10; i++ {
		metric := types.Metric{
			Name:      "test_metric",
			Timestamp: time.Now().Unix() + int64(i),
			Value:     float64(i),
			Labels:    map[string]string{"index": string(rune(i))},
		}
		mustStoreMetric(t, storage, metric)
	}

	metrics, _ := storage.GetMetrics("test_metric", 0)
	if len(metrics) != 10 {
		t.Errorf("expected 10 metrics, got %d", len(metrics))
	}
}

func TestStoreMetric_ExceedsMaxSize(t *testing.T) {
	// Create storage with small max size
	storage := &MemoryStorage{
		metrics: make([]types.Metric, 0),
		maxSize: 5,
	}

	// Store more metrics than max size
	for i := 0; i < 10; i++ {
		metric := types.Metric{
			Name:      "test_metric",
			Timestamp: time.Now().Unix() + int64(i),
			Value:     float64(i),
			Labels:    map[string]string{},
		}
		mustStoreMetric(t, storage, metric)
	}

	// Should only keep last 5 metrics
	if len(storage.metrics) != 5 {
		t.Errorf("expected 5 metrics after eviction, got %d", len(storage.metrics))
	}

	// Oldest metrics should be removed (values should be 5-9)
	metrics, _ := storage.GetMetrics("test_metric", 0)
	if metrics[0].Value != 5.0 {
		t.Errorf("expected oldest metric to have value 5.0, got %f", metrics[0].Value)
	}
	if metrics[4].Value != 9.0 {
		t.Errorf("expected newest metric to have value 9.0, got %f", metrics[4].Value)
	}
}

func TestGetMetrics_FilterByName(t *testing.T) {
	storage := NewMemoryStorage()

	// Store metrics with different names
	mustStoreMetric(t, storage, types.Metric{Name: "metric_a", Timestamp: 1, Value: 1.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_b", Timestamp: 2, Value: 2.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_a", Timestamp: 3, Value: 3.0, Labels: map[string]string{}})

	// Get only metric_a
	metrics, err := storage.GetMetrics("metric_a", 0)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics named 'metric_a', got %d", len(metrics))
	}
	for _, m := range metrics {
		if m.Name != "metric_a" {
			t.Errorf("expected name 'metric_a', got '%s'", m.Name)
		}
	}
}

func TestGetMetrics_WithLimit(t *testing.T) {
	storage := NewMemoryStorage()

	// Store 10 metrics
	for i := 0; i < 10; i++ {
		mustStoreMetric(t, storage, types.Metric{
			Name:      "test_metric",
			Timestamp: int64(i),
			Value:     float64(i),
			Labels:    map[string]string{},
		})
	}

	// Get with limit
	metrics, err := storage.GetMetrics("test_metric", 5)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 5 {
		t.Errorf("expected 5 metrics with limit 5, got %d", len(metrics))
	}
}

func TestGetMetrics_NoNameFilter(t *testing.T) {
	storage := NewMemoryStorage()

	mustStoreMetric(t, storage, types.Metric{Name: "metric_a", Timestamp: 1, Value: 1.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_b", Timestamp: 2, Value: 2.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_c", Timestamp: 3, Value: 3.0, Labels: map[string]string{}})

	// Get all metrics (no name filter, empty string)
	metrics, err := storage.GetMetrics("", 0)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics with no filter, got %d", len(metrics))
	}
}

func TestGetMetrics_Empty(t *testing.T) {
	storage := NewMemoryStorage()

	metrics, err := storage.GetMetrics("nonexistent", 0)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(metrics))
	}
}

func TestGetMetricsByLabel(t *testing.T) {
	storage := NewMemoryStorage()

	// Store metrics with different labels
	mustStoreMetric(t, storage, types.Metric{
		Name:      "metric_a",
		Timestamp: 1,
		Value:     1.0,
		Labels:    map[string]string{"account": "0.0.5000", "type": "balance"},
	})
	mustStoreMetric(t, storage, types.Metric{
		Name:      "metric_b",
		Timestamp: 2,
		Value:     2.0,
		Labels:    map[string]string{"account": "0.0.5001", "type": "balance"},
	})
	mustStoreMetric(t, storage, types.Metric{
		Name:      "metric_c",
		Timestamp: 3,
		Value:     3.0,
		Labels:    map[string]string{"account": "0.0.5000", "type": "transaction"},
	})

	// Get metrics by label
	metrics, err := storage.GetMetricsByLabel("account", "0.0.5000")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics for account 0.0.5000, got %d", len(metrics))
	}
	for _, m := range metrics {
		if val, exists := m.Labels["account"]; !exists || val != "0.0.5000" {
			t.Errorf("expected account label to be 0.0.5000, got %v", val)
		}
	}
}

func TestGetMetricsByLabel_NonExistent(t *testing.T) {
	storage := NewMemoryStorage()

	mustStoreMetric(t, storage, types.Metric{
		Name:      "metric_a",
		Timestamp: 1,
		Value:     1.0,
		Labels:    map[string]string{"key": "value"},
	})

	// Query non-existent label
	metrics, err := storage.GetMetricsByLabel("nonexistent_key", "nonexistent_value")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics for non-existent label, got %d", len(metrics))
	}
}

func TestDeleteOldMetrics(t *testing.T) {
	storage := NewMemoryStorage()

	now := time.Now().Unix()

	// Store metrics at different times
	mustStoreMetric(t, storage, types.Metric{Name: "metric_a", Timestamp: now - 1000, Value: 1.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_b", Timestamp: now - 500, Value: 2.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_c", Timestamp: now - 100, Value: 3.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_d", Timestamp: now, Value: 4.0, Labels: map[string]string{}})

	// Delete metrics older than now - 200
	cutoffTime := now - 200
	err := storage.DeleteOldMetrics(cutoffTime)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Should have 2 metrics left (timestamps >= cutoffTime)
	metrics, _ := storage.GetMetrics("", 0)
	if len(metrics) != 2 {
		t.Errorf("expected 2 metrics after deletion, got %d", len(metrics))
	}

	// Verify remaining metrics are newer
	for _, m := range metrics {
		if m.Timestamp < cutoffTime {
			t.Errorf("expected all remaining metrics to have timestamp >= %d, got %d", cutoffTime, m.Timestamp)
		}
	}
}

func TestDeleteOldMetrics_All(t *testing.T) {
	storage := NewMemoryStorage()

	now := time.Now().Unix()

	mustStoreMetric(t, storage, types.Metric{Name: "metric_a", Timestamp: now - 1000, Value: 1.0, Labels: map[string]string{}})
	mustStoreMetric(t, storage, types.Metric{Name: "metric_b", Timestamp: now - 500, Value: 2.0, Labels: map[string]string{}})

	// Delete all metrics
	err := storage.DeleteOldMetrics(now + 1000)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	metrics, _ := storage.GetMetrics("", 0)
	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics after deleting all, got %d", len(metrics))
	}
}

func TestClose(t *testing.T) {
	storage := NewMemoryStorage()

	err := storage.StoreMetric(types.Metric{Name: "metric_a", Timestamp: 1, Value: 1.0, Labels: map[string]string{}})
	if err != nil {
		t.Fatalf("failed to store metric: %v", err)
	}

	// Close should succeed
	err = storage.Close()
	if err != nil {
		t.Errorf("expected no error closing storage, got: %v", err)
	}

	// After close, metrics should be nil
	if storage.metrics != nil {
		t.Error("expected metrics to be nil after close")
	}
}

func TestStats(t *testing.T) {
	storage := NewMemoryStorage()

	// Add some metrics
	for i := 0; i < 100; i++ {
		mustStoreMetric(t, storage, types.Metric{
			Name:      "test_metric",
			Timestamp: int64(i),
			Value:     float64(i),
			Labels:    map[string]string{},
		})
	}

	stats, err := storage.Stats()
	if err != nil {
		t.Errorf("expected no error getting stats, got: %v", err)
	}

	// Check stats content
	count, exists := stats["metric_count"].(int)
	if !exists {
		t.Error("expected 'metric_count' in stats")
	}
	if count != 100 {
		t.Errorf("expected metric_count 100, got %d", count)
	}

	maxSize, exists := stats["max_size"].(int)
	if !exists {
		t.Error("expected 'max_size' in stats")
	}
	if maxSize != DefaultMaxSize {
		t.Errorf("expected max_size %d, got %d", DefaultMaxSize, maxSize)
	}

	utilization, exists := stats["utilization"].(string)
	if !exists {
		t.Error("expected 'utilization' in stats")
	}
	if utilization == "" {
		t.Error("expected non-empty utilization string")
	}
}

func TestStoreMetric_ConcurrentAccess(t *testing.T) {
	storage := NewMemoryStorage()

	// Concurrently store metrics from multiple goroutines
	done := make(chan bool, 10)

	for g := 0; g < 10; g++ {
		go func(goroutineID int) {
			for i := 0; i < 100; i++ {
				metric := types.Metric{
					Name:      "concurrent_metric",
					Timestamp: time.Now().Unix(),
					Value:     float64(goroutineID*100 + i),
					Labels:    map[string]string{"goroutine": string(rune(goroutineID))},
				}
				mustStoreMetric(t, storage, metric)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Storage should be consistent (accounting for eviction)
	metrics, _ := storage.GetMetrics("concurrent_metric", 0)
	if len(metrics) == 0 {
		t.Error("expected at least some metrics stored concurrently")
	}
	if len(metrics) > storage.maxSize {
		t.Errorf("expected at most %d metrics due to max size limit, got %d", storage.maxSize, len(metrics))
	}
}
