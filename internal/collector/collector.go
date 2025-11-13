package collector

import (
	"context"

	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// AlertManager is an interface for alert management
// This interface allows collectors to depend on abstraction rather than concrete alerting.Manager
type AlertManager interface {
	// CheckMetric evaluates a metric against alert rules
	CheckMetric(metric types.Metric) error
}

// Collector is the interface that all metric collectors must implement
type Collector interface {
	// Name returns the name of the collector
	Name() string

	// Collect runs the collection loop, sending metrics to storage and alerts
	// The context signals when the collector should stop
	Collect(ctx context.Context, store storage.Storage, alertMgr AlertManager) error
}

// BaseCollector provides common functionality for collectors
type BaseCollector struct {
	name string
}

// Name returns the collector's name
func (bc *BaseCollector) Name() string {
	return bc.name
}

// NewBaseCollector creates a new base collector
func NewBaseCollector(name string) *BaseCollector {
	return &BaseCollector{name: name}
}
