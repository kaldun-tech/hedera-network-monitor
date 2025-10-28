package storage

import (
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// Storage is the interface for storing and retrieving metrics
type Storage interface {
	// StoreMetric persists a metric to storage
	StoreMetric(metric types.Metric) error

	// GetMetrics retrieves metrics matching the given criteria
	// name: metric name filter (empty string = all)
	// limit: maximum number of metrics to return (0 = unlimited)
	GetMetrics(name string, limit int) ([]types.Metric, error)

	// GetMetricsByLabel retrieves metrics matching the given label key-value pair
	GetMetricsByLabel(key, value string) ([]types.Metric, error)

	// DeleteOldMetrics removes metrics older than the given timestamp
	// This is useful for cleanup and managing storage size
	DeleteOldMetrics(beforeTimestamp int64) error

	// Close closes the storage backend (cleanup, close connections, etc.)
	Close() error
}
