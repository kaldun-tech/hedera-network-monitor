package storage

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/logger"
)

// MemoryStorage is an in-memory implementation of Storage
// It's suitable for MVP and testing, but not for production use.
// For production, consider: PostgreSQL, InfluxDB, Prometheus, or similar.
type MemoryStorage struct {
	metrics []types.Metric
	mu      sync.RWMutex
	maxSize int // Maximum number of metrics to keep in memory
}

const DefaultMaxSize = 10000

func parseMaxSize(s string) (maxSize int) {
	// Return default if string is empty
	if s == "" {
		return DefaultMaxSize
	}

	// Parse the first numeric value from the string
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return DefaultMaxSize
	}

	maxSize, err := strconv.Atoi(fields[0])
	if err != nil {
		logger.Warn("Invalid max size format, using default",
			"component", "MemoryStorage",
			"format", s,
			"default", DefaultMaxSize)
		return DefaultMaxSize
	}
	return maxSize
}

// NewMemoryStorage creates a new in-memory storage
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		metrics: make([]types.Metric, 0, 10000),
		maxSize: parseMaxSize(os.Getenv("COLLECTOR_MEMORY_MAX_SIZE")),
	}
}

// StoreMetric implements Storage interface
func (ms *MemoryStorage) StoreMetric(metric types.Metric) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Check if we need to remove old metrics to stay under size limit
	if len(ms.metrics) >= ms.maxSize {
		// Remove oldest metrics (assuming they are sorted by timestamp)
		// TODO: Implement more sophisticated eviction policy (LRU, etc.)
		ms.metrics = ms.metrics[1:]
	}

	ms.metrics = append(ms.metrics, metric)
	return nil
}

// GetMetrics implements Storage interface
func (ms *MemoryStorage) GetMetrics(name string, limit int) ([]types.Metric, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make([]types.Metric, 0)

	for _, metric := range ms.metrics {
		// Filter by name if specified
		if name != "" && metric.Name != name {
			continue
		}

		result = append(result, metric)

		// Check limit
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

// GetMetricsByLabel implements Storage interface
func (ms *MemoryStorage) GetMetricsByLabel(key, value string) ([]types.Metric, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	result := make([]types.Metric, 0)

	for _, metric := range ms.metrics {
		if metricValue, exists := metric.Labels[key]; exists && metricValue == value {
			result = append(result, metric)
		}
	}

	return result, nil
}

// DeleteOldMetrics implements Storage interface
func (ms *MemoryStorage) DeleteOldMetrics(beforeTimestamp int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	newMetrics := make([]types.Metric, 0, len(ms.metrics))

	for _, metric := range ms.metrics {
		if metric.Timestamp >= beforeTimestamp {
			newMetrics = append(newMetrics, metric)
		}
	}

	ms.metrics = newMetrics
	return nil
}

// Close implements Storage interface
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.metrics = nil
	return nil
}

// Stats returns storage statistics (useful for debugging and monitoring)
func (ms *MemoryStorage) Stats() (map[string]interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	return map[string]interface{}{
		"metric_count": len(ms.metrics),
		"max_size":     ms.maxSize,
		"utilization":  fmt.Sprintf("%.2f%%", float64(len(ms.metrics))/float64(ms.maxSize)*100),
	}, nil
}
