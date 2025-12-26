package collector

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/logger"
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

const DefaultInterval = 30 * time.Second

// ParseInterval parses an interval string and returns a time.Duration
// If the string is empty, invalid, or non-positive, returns the default interval
func ParseInterval(s string) time.Duration {
	// Return default if string is empty
	if s == "" {
		return DefaultInterval
	}

	// Parse the first numeric value from the string
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return DefaultInterval
	}

	seconds, err := strconv.Atoi(fields[0])
	if err != nil {
		logger.Warn("Invalid interval format, using default",
			"component", "AccountCollector",
			"format", s)
		return DefaultInterval
	}

	if seconds <= 0 {
		logger.Warn("Interval must be positive, using default",
			"component", "AccountCollector",
			"interval", seconds)
		return DefaultInterval
	}

	return time.Duration(seconds) * time.Second
}

// NewAccountCollector creates a new account collector
func NewAccountCollector(client hedera.Client, accounts []AccountConfig) *AccountCollector {
	return &AccountCollector{
		BaseCollector: NewBaseCollector("AccountCollector"),
		client:        client,
		accounts:      accounts,
		interval:      ParseInterval(os.Getenv("COLLECTOR_INTERVAL")),
	}
}

func (ac *AccountCollector) buildTransactionTypeMetric(accountRecords []hedera.Record,
	accountID, label string) []types.Metric {

	typeCounts := make(map[hedera.TransactionType]int)

	// Count transactions by type
	for _, record := range accountRecords {
		typeCounts[record.Type]++
	}

	// Build the metrics
	metrics := make([]types.Metric, 0, len(typeCounts))
	for txType, count := range typeCounts {
		nextMetric := types.Metric{
			Name:      "account_transaction_type_count",
			Timestamp: time.Now().Unix(),
			Value:     float64(count),
			Labels: map[string]string{
				"account_id":       accountID,
				"label":            label,
				"transaction_type": txType.String(),
			},
		}
		metrics = append(metrics, nextMetric)
	}
	return metrics
}

// Collect implements the Collector interface
func (ac *AccountCollector) Collect(ctx context.Context, store storage.Storage, alertMgr AlertManager) error {
	ticker := time.NewTicker(ac.interval)
	defer ticker.Stop()

	logger.Info("Starting account collector",
		"component", ac.Name(),
		"interval", ac.interval,
		"accounts", len(ac.accounts))

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping collector", "component", ac.Name())
			return ctx.Err()
		case <-ticker.C:
			// Collect metrics for each account
			for _, accountCfg := range ac.accounts {
				allMetrics := make([]types.Metric, 0)

				// 1. Query account balance
				balance, err := ac.client.GetAccountBalance(accountCfg.ID)
				if err != nil {
					logger.Error("Error getting balance",
						"component", ac.Name(),
						"account_id", accountCfg.ID,
						"error", err)
					return err
				}

				allMetrics = append(allMetrics, types.Metric{
					Name:      "account_balance",
					Timestamp: time.Now().Unix(),
					Value:     float64(balance),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				})

				// 2. Query recent transactions (limit to 50 records per query)
				accountRecords, err := ac.client.GetAccountRecords(accountCfg.ID, 50)
				if err != nil {
					logger.Error("Error getting account records",
						"component", ac.Name(),
						"account_id", accountCfg.ID,
						"error", err)
					// Continue to next account on error instead of failing entire collection
					continue
				}

				// 3. Calculate derived metrics from transaction records
				transactionCount := len(accountRecords)
				allMetrics = append(allMetrics, types.Metric{
					Name:      "account_transaction_count",
					Timestamp: time.Now().Unix(),
					Value:     float64(transactionCount),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				})

				// TASK 2 - Transaction type breakdown
				typeMetrics := ac.buildTransactionTypeMetric(accountRecords,
					accountCfg.ID, accountCfg.Label)
				allMetrics = append(allMetrics, typeMetrics...)

				// TASK 3 - Volume metrics
				// Sum the total amount transferred in this interval
				total := int64(0)
				for _, rec := range accountRecords {
					total += rec.AmountTinyBar
				}
				allMetrics = append(allMetrics, types.Metric{
					Name:      "account_total_volume",
					Timestamp: time.Now().Unix(),
					Value:     float64(total),
					Labels: map[string]string{
						"account_id": accountCfg.ID,
						"label":      accountCfg.Label,
					},
				})

				// Store and check all metrics
				for _, metric := range allMetrics {
					if err := store.StoreMetric(metric); err != nil {
						logger.Error("Error storing metric",
							"component", ac.Name(),
							"metric_name", metric.Name,
							"error", err)
					}
					if err := alertMgr.CheckMetric(metric); err != nil {
						logger.Error("Error checking alerts",
							"component", ac.Name(),
							"metric_name", metric.Name,
							"error", err)
					}
				}

				// Bonus idea: track inflows vs outflows separately if possible
			}
		}
	}
}
