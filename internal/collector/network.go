package collector

import (
	"context"
	"log"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
)

// NetworkCollector collects network-wide metrics from the Hedera network
type NetworkCollector struct {
	*BaseCollector
	client   hedera.Client
	interval time.Duration
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector(client hedera.Client) *NetworkCollector {
	return &NetworkCollector{
		BaseCollector: NewBaseCollector("NetworkCollector"),
		client:        client,
		interval:      60 * time.Second, // TODO: Make configurable
	}
}

// Collect implements the Collector interface
func (nc *NetworkCollector) Collect(ctx context.Context, store storage.Storage, alertMgr *alerting.Manager) error {
	ticker := time.NewTicker(nc.interval)
	defer ticker.Stop()

	log.Printf("[%s] Starting collection with interval %v", nc.Name(), nc.interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Stopping collector", nc.Name())
			return ctx.Err()
		case <-ticker.C:
			// TODO: Implement actual network metrics collection
			// This should:
			// 1. Query network info (available nodes, versions, etc.)
			// 2. Check node availability/health
			// 3. Monitor transaction success rates
			// 4. Track network fees
			// 5. Send metrics to storage

			// Example metric for demonstration
			metric := types.Metric{
				Name:      "network_nodes_available",
				Timestamp: time.Now().Unix(),
				Value:     25.0, // TODO: Get actual node count from Hedera
				Labels: map[string]string{
					"network": "testnet", // TODO: Get from config
				},
			}

			// Store the metric
			if err := store.StoreMetric(metric); err != nil {
				log.Printf("[%s] Error storing metric: %v", nc.Name(), err)
			}

			// TODO: Check metric against alert rules
			// if err := alertMgr.CheckMetric(metric); err != nil {
			//     log.Printf("[%s] Error checking alerts: %v", nc.Name(), err)
			// }
		}
	}
}
