package collector

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
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

const DEFAULT_INTERVAL_SECONDS = 30 * time.Second

func parseInterval(s string) time.Duration {
	// Return default if string is empty
	if s == "" {
		return DEFAULT_INTERVAL_SECONDS
	}

	// Parse the first numeric value from the string
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return DEFAULT_INTERVAL_SECONDS
	}

	seconds, err := strconv.Atoi(fields[0])
	if err != nil {
		log.Printf("Invalid interval format: %q, using default", s)
		return DEFAULT_INTERVAL_SECONDS
	}

	if seconds <= 0 {
		log.Printf("Interval must be positive, got %d, using default", seconds)
		return DEFAULT_INTERVAL_SECONDS
	}

	return time.Duration(seconds) * time.Second
}

// NewAccountCollector creates a new account collector
func NewAccountCollector(client hedera.Client, accounts []AccountConfig) *AccountCollector {
	return &AccountCollector{
		BaseCollector: NewBaseCollector("AccountCollector"),
		client:        client,
		accounts:      accounts,
		interval:      parseInterval(os.Getenv("COLLECTOR_INTERVAL")),
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
				// 1. Query account balance
				balance, err := ac.client.GetAccountBalance(accountCfg.ID)
				if err != nil {
					log.Printf("[%s] Error getting balance: %v", ac.Name(), err)
					return err
				}

				accountBalanceMetric := types.Metric{
					Name:      "account_balance",
					Timestamp: time.Now().Unix(),
					Value:     float64(balance),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				}

				// 2. Query recent transactions (limit to 50 records per query)
				accountRecords, err := ac.client.GetAccountRecords(accountCfg.ID, 50)
				if err != nil {
					log.Printf("[%s] Error getting account records: %v", ac.Name(), err)
					// Continue to next account on error instead of failing entire collection
					continue
				}

				// 3. Calculate derived metrics from transaction records
				transactionCount := len(accountRecords)
				txnCountMetric := types.Metric{
					Name:      "account_transaction_count",
					Timestamp: time.Now().Unix(),
					Value:     float64(transactionCount),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				}

				// TODO: TASK 2 - Transaction type breakdown
				// Count transactions by type: CryptoTransfer, TokenTransfer, ContractCall, etc.
				// Iterate: for _, rec := range accountRecords { /* count by rec.Type */ }
				// Store each type as a separate metric with label "transaction_type": "TypeName"
				// Helps identify which types of transactions are most active

				// TASK 3 - Volume metrics
				// Sum the total amount transferred in this interval
				total := int64(0)
				for _, rec := range accountRecords {
					total += rec.AmountTinyBar
				}
				// Store as "account_total_volume" metric in tinybar units
				volumeMetric := types.Metric{
					Name:      "account_total_volume",
					Timestamp: time.Now().Unix(),
					Value:     float64(total),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				}
				// Bonus idea: track inflows vs outflows separately if possible

				// 4. Store metrics to storage
				if err := store.StoreMetric(accountBalanceMetric); err != nil {
					log.Printf("[%s] Error storing balance metric: %v", ac.Name(), err)
				}

				if err := store.StoreMetric(txnCountMetric); err != nil {
					log.Printf("[%s] Error storing transaction count metric: %v", ac.Name(), err)
				}

				if err := store.StoreMetric(volumeMetric); err != nil {
					log.Printf("[%s] Error storing volume metric: %v", ac.Name(), err)
				}

				// 5. Check metrics against alert rules
				if err := alertMgr.CheckMetric(accountBalanceMetric); err != nil {
					log.Printf("[%s] Error checking balance alerts: %v", ac.Name(), err)
				}

				if err := alertMgr.CheckMetric(txnCountMetric); err != nil {
					log.Printf("[%s] Error checking transaction count alerts: %v", ac.Name(), err)
				}

				if err := alertMgr.CheckMetric(volumeMetric); err != nil {
					log.Printf("[%s] Error checking volume alerts: %v", ac.Name(), err)
				}
			}
		}
	}
}
