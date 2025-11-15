package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	apiURL   string
	loglevel string
	network  string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hmon",
	Short: "Hedera Network Monitor CLI",
	Long: `hmon is a command-line interface for the Hedera Network Monitor.

It provides tools to query the monitoring service for account information,
network status, and manage alert rules.

Usage:
  hmon account balance <account-id>
  hmon account transactions <account-id>
  hmon network status
  hmon alerts list
  hmon alerts add <rule>`,
	Version: "0.1.0",
}

// accountCmd represents the account command group
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Query account information",
	Long:  "Query account information from the Hedera network or monitoring service",
}

// accountBalanceCmd represents the account balance command
var accountBalanceCmd = &cobra.Command{
	Use:   "balance <account-id>",
	Short: "Get account balance",
	Long:  "Retrieve the current balance for a given account ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		fmt.Printf("Querying balance for account: %s\n", accountID)
		client, err := hedera.NewClient(getNetworkName())
		if err != nil {
			return err
		}

		balance, err := client.GetAccountBalance(accountID)
		if err != nil {
			return err
		}

		fmt.Printf("Balance for account %s: %d\n", accountID, balance)
		return nil
	},
}

// accountTransactionsCmd represents the account transactions command
var accountTransactionsCmd = &cobra.Command{
	Use:   "transactions <account-id>",
	Short: "Get account transactions",
	Long:  "Retrieve recent transactions for a given account ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		accountID := args[0]
		fmt.Printf("Querying transactions for account: %s\n", accountID)
		client, err := hedera.NewClient(getNetworkName())
		if err != nil {
			return err
		}

		transactions, err := client.GetAccountRecords(accountID, 10)
		if err != nil {
			return err
		}

		fmt.Printf("\nRecent transactions for account %s:\n", accountID)
		fmt.Println(formatTransactions(transactions))
		return nil
	},
}

// networkCmd represents the network command group
var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Query network information",
	Long:  "Query network status and information from the Hedera blockchain",
}

// networkStatusCmd represents the network status command
var networkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get network status",
	Long:  "Retrieve current network status and health metrics from the monitoring service",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Querying network status from monitoring service...")

		// Query metrics from the monitoring service
		nodeMetrics, err := queryMetricsByName("network_nodes_available")
		if err != nil {
			return fmt.Errorf("failed to query network metrics: %w", err)
		}

		consensusMetrics, err := queryMetricsByName("network_consensus_active")
		if err != nil {
			return fmt.Errorf("failed to query consensus metrics: %w", err)
		}

		// Display results
		fmt.Println("\nNetwork Status:")
		if len(nodeMetrics) > 0 {
			nodeCount := int(nodeMetrics[0].Value)
			fmt.Printf("  Available Nodes: %d\n", nodeCount)
		} else {
			fmt.Println("  Available Nodes: No data")
		}

		if len(consensusMetrics) > 0 {
			isActive := consensusMetrics[0].Value == 1.0
			status := "DOWN"
			if isActive {
				status = "UP"
			}
			fmt.Printf("  Consensus Status: %s\n", status)
		} else {
			fmt.Println("  Consensus Status: No data")
		}

		return nil
	},
}

// alertsCmd represents the alerts command group
var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alert rules",
	Long:  "View and manage alert rules in the monitoring service",
}

// alertsListCmd represents the alerts list command
var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List alert rules",
	Long:  "Display all configured alert rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleAlertsList()
	},
}

// alertsAddCmd represents the alerts add command
var alertsAddCmd = &cobra.Command{
	Use:   "add <rule-json>",
	Short: "Add new alert rule",
	Long: `Create a new alert rule.
Rule must be provided as JSON with required fields:
  - name: Rule name
  - metric_name: Metric to monitor (e.g., "account_balance")
  - condition: Comparison operator (>, <, >=, <=, ==, !=)
  - threshold: Numeric threshold value
  - severity: Alert severity (info, warning, critical)

Optional fields:
  - description: Rule description
  - cooldown_seconds: Cooldown between alerts (default: 300)

Example:
  hmon alerts add '{"name":"Low Balance","metric_name":"account_balance","condition":"<","threshold":1000000000,"severity":"warning"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handleAlertAdd(args[0])
	},
}

func getNetworkName() string {
	if network != "" {
		return network // CLI flag wins
	}
	if env := os.Getenv("NETWORK_NAME"); env != "" {
		return env // env var is second
	}
	// Fallback to default
	return "testnet"
}

// MetricResponse defines types for API queries
type MetricResponse struct {
	Name      string            `json:"name"`
	Timestamp int64             `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels"`
}

type MetricsAPIResponse struct {
	Metrics []MetricResponse `json:"metrics"`
	Count   int              `json:"count"`
	Error   string           `json:"error,omitempty"`
}

// queryMetricsByName queries the monitoring service API for metrics by name
func queryMetricsByName(metricName string) ([]MetricResponse, error) {
	params := url.Values{}
	params.Add("name", metricName)
	params.Add("limit", "10")

	fullURL := fmt.Sprintf("%s/api/v1/metrics?%s", apiURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp MetricsAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return apiResp.Metrics, nil
}

// formatTransactions formats a slice of transaction records for display
func formatTransactions(transactions []hedera.Record) string {
	if len(transactions) == 0 {
		return "No transactions found"
	}

	// Header
	output := fmt.Sprintf("%-37s %-20s %-15s %-10s\n", "Transaction ID", "Type", "Amount (ℏ)", "Status")
	output += fmt.Sprintf("%s %s %s %s\n",
		"-------------------------------------",
		"--------------------",
		"---------------",
		"----------")

	// Rows
	for _, tx := range transactions {
		// Convert tinybar to HBAR
		hbarAmount := float64(tx.AmountTinyBar) / float64(hedera.TinybarPerHbar)

		output += fmt.Sprintf("%-37s %-20s %14.2f %-10s\n",
			tx.TransactionID,
			tx.Type.String(),
			hbarAmount,
			tx.Status)
	}

	return output
}

// AlertRuleResponse represents an alert rule from the API
type AlertRuleResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	MetricName      string  `json:"metric_name"`
	Condition       string  `json:"condition"`
	Threshold       float64 `json:"threshold"`
	Severity        string  `json:"severity"`
	Enabled         bool    `json:"enabled"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

// AlertListResponse wraps alert rules
type AlertListResponse struct {
	Alerts []AlertRuleResponse `json:"alerts"`
	Count  int                 `json:"count"`
}

// CreateAlertRequest is the request payload for creating an alert
type CreateAlertRequest struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	MetricName      string  `json:"metric_name"`
	Condition       string  `json:"condition"`
	Threshold       float64 `json:"threshold"`
	Severity        string  `json:"severity"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

// handleAlertsList fetches and displays all alert rules
func handleAlertsList() error {
	fullURL := fmt.Sprintf("%s/api/v1/alerts", apiURL)

	resp, err := http.Get(fullURL)
	if err != nil {
		return fmt.Errorf("failed to query API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response AlertListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Display results
	if response.Count == 0 {
		fmt.Println("No alert rules configured")
		return nil
	}

	fmt.Printf("\nConfigured Alert Rules (%d):\n", response.Count)
	fmt.Println("------------------------------------")

	for i, rule := range response.Alerts {
		fmt.Printf("\n[%d] %s\n", i+1, rule.Name)
		fmt.Printf("    ID:              %s\n", rule.ID)
		if rule.Description != "" {
			fmt.Printf("    Description:     %s\n", rule.Description)
		}
		fmt.Printf("    Metric:          %s\n", rule.MetricName)
		fmt.Printf("    Condition:       %s %.0f\n", rule.Condition, rule.Threshold)
		fmt.Printf("    Severity:        %s\n", rule.Severity)
		fmt.Printf("    Enabled:         %v\n", rule.Enabled)
		if rule.CooldownSeconds > 0 {
			fmt.Printf("    Cooldown:        %d seconds\n", rule.CooldownSeconds)
		}
	}

	return nil
}

// handleAlertAdd creates a new alert rule
func handleAlertAdd(ruleJSON string) error {
	// Parse JSON into CreateAlertRequest
	var request CreateAlertRequest
	if err := json.Unmarshal([]byte(ruleJSON), &request); err != nil {
		return fmt.Errorf("failed to parse rule JSON: %w (expected JSON format)", err)
	}

	// Make POST request to API
	fullURL := fmt.Sprintf("%s/api/v1/alerts", apiURL)

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var response AlertRuleResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("\nAlert rule created successfully!")
	fmt.Printf("ID:        %s\n", response.ID)
	fmt.Printf("Name:      %s\n", response.Name)
	fmt.Printf("Metric:    %s\n", response.MetricName)
	fmt.Printf("Condition: %s %.0f\n", response.Condition, response.Threshold)
	fmt.Printf("Severity:  %s\n", response.Severity)

	return nil
}

func init() {
	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")
	rootCmd.PersistentFlags().StringVar(&loglevel, "loglevel", "info", "Log level (debug, info, warn, error)")

	rootCmd.PersistentFlags().StringVar(&network, "network", "", "Hedera network name (mainnet/testnet), defaults to NETWORK_NAME in .env or testnet")

	// Add command groups
	rootCmd.AddCommand(accountCmd)
	rootCmd.AddCommand(networkCmd)
	rootCmd.AddCommand(alertsCmd)

	// Add account subcommands
	accountCmd.AddCommand(accountBalanceCmd)
	accountCmd.AddCommand(accountTransactionsCmd)

	// Add network subcommands
	networkCmd.AddCommand(networkStatusCmd)

	// Add alerts subcommands
	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsAddCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, err := fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}
