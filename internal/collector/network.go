package collector

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
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
		interval:      ParseInterval(os.Getenv("COLLECTOR_INTERVAL")),
	}
}

// Per-Node Availability and Endpoint Metrics
// Consider future: actually ping/query each node to verify active status
func buildPerNodeMetrics(NodeAddresses []hiero.NodeAddress, networkName string) []types.Metric {
	metrics := make([]types.Metric, 0)

	for _, nodeAddress := range NodeAddresses {
		// Extract node info
		nodeID := nodeAddress.NodeID
		nodeString := strconv.FormatInt(nodeID, 10)
		nodeAccountID := nodeAddress.AccountID

		// Count available endpoints for this node
		endpointCount := len(nodeAddress.Addresses)

		// Create per-node availability metric
		nodeAvailableMetric := types.Metric{
			Name:      "network_node_available",
			Timestamp: time.Now().Unix(),
			Value:     1.0, // 1.0 = node is in address book
			Labels: map[string]string{
				"network":         networkName,
				"node_id":         nodeString,
				"node_account_id": nodeAccountID.String(),
			},
		}
		metrics = append(metrics, nodeAvailableMetric)

		// Create endpoint count metric for this node
		endpointMetric := types.Metric{
			Name:      "network_node_endpoints",
			Timestamp: time.Now().Unix(),
			Value:     float64(endpointCount),
			Labels: map[string]string{
				"network":         networkName,
				"node_id":         nodeString,
				"node_account_id": nodeAccountID.String(),
			},
		}
		metrics = append(metrics, endpointMetric)
	}

	return metrics
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
			log.Printf("[%s] Collecting metrics", nc.Name())

			// Track if address book query was successful (for consensus status metric)
			consensusValue := 0.0
			allMetrics := make([]types.Metric, 0)

			// 1. Query network info (available nodes, versions, etc.)
			addressBook, err := nc.client.GetNodeAddressBook()
			if err == nil {
				// Network is up
				consensusValue = 1.0

				// TASK 1 - Node Count Metric
				nodeCount := len(addressBook.NodeAddresses)
				allMetrics = append(allMetrics, types.Metric{
					Name:      "network_nodes_available",
					Timestamp: time.Now().Unix(),
					Value:     float64(nodeCount),
					Labels: map[string]string{
						"network": nc.Name(),
					},
				})

				// TASK 2 & 3 - Per-Node Availability and Endpoint Metrics
				perNodeMetrics := buildPerNodeMetrics(addressBook.NodeAddresses, nc.Name())
				allMetrics = append(allMetrics, perNodeMetrics...)

				log.Printf("[%s] Completed metric collection from address book (%d nodes)",
					nc.Name(), len(addressBook.NodeAddresses))
			} else {
				log.Printf("[%s] Skipped metric collection due to address book error: %v", nc.Name(), err)
				// Network is down -> report 0 for consensus metric
			}

			// TASK 4 - Network Consensus Status
			allMetrics = append(allMetrics, types.Metric{
				Name:      "network_consensus_active",
				Timestamp: time.Now().Unix(),
				Value:     consensusValue,
				Labels:    map[string]string{"network": nc.Name()},
			})

			// Store and check all metrics
			for _, metric := range allMetrics {
				if err := store.StoreMetric(metric); err != nil {
					log.Printf("[%s] Error storing metric %s: %v", nc.Name(), metric.Name, err)
				}
				if err := alertMgr.CheckMetric(metric); err != nil {
					log.Printf("[%s] Error checking alerts for %s: %v", nc.Name(), metric.Name, err)
				}
			}
		}
	}
}
