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

// AccountConfig represents configuration for account monitoring
type AccountConfig struct {
	ID    string // Account ID in format "0.0.123"
	Label string // Human-readable label for the account
}

// AccountCollector collects metrics for specified Hedera accounts
type AccountCollector struct {
	*BaseCollector
	client   hedera.Client
	accounts []AccountConfig
	interval time.Duration
}

// NewAccountCollector creates a new account collector
func NewAccountCollector(client hedera.Client, accounts []AccountConfig) *AccountCollector {
	return &AccountCollector{
		BaseCollector: NewBaseCollector("AccountCollector"),
		client:        client,
		accounts:      accounts,
		interval:      30 * time.Second, // TODO: Make configurable
	}
}

// Collect implements the Collector interface
func (ac *AccountCollector) Collect(ctx context.Context, store storage.Storage, alertMgr *alerting.Manager) error {
	ticker := time.NewTicker(ac.interval)
	defer ticker.Stop()

	log.Printf("[%s] Starting collection with interval %v", ac.Name(), ac.interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Stopping collector", ac.Name())
			return ctx.Err()
		case <-ticker.C:
			// Collect metrics for each account
			for _, accountCfg := range ac.accounts {
				// TODO: Implement actual metric collection using Hedera SDK
				// This should:
				// 1. Query account balance
				// 2. Query recent transactions
				// 3. Calculate derived metrics (e.g., transaction rate)
				// 4. Send metrics to storage
				// 5. Check against alert rules and send alerts if needed

				metric := types.Metric{
					Name:      "account_balance",
					Timestamp: time.Now().Unix(),
					Value:     0.0, // TODO: Get actual balance from Hedera
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				}

				// Store the metric
				if err := store.StoreMetric(metric); err != nil {
					log.Printf("[%s] Error storing metric: %v", ac.Name(), err)
				}

				// TODO: Check metric against alert rules
				// if err := alertMgr.CheckMetric(metric); err != nil {
				//     log.Printf("[%s] Error checking alerts: %v", ac.Name(), err)
				// }
			}
		}
	}
}
