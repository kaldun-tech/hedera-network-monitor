package collector

import (
	"context"

	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
)

// Collector is the interface that all metric collectors must implement
type Collector interface {
	// Name returns the name of the collector
	Name() string

	// Collect runs the collection loop, sending metrics to storage and alerts
	// The context signals when the collector should stop
	Collect(ctx context.Context, store storage.Storage, alertMgr *alerting.Manager) error
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
