package main

import (
	"flag"
	"log"
	"time"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
)

const (
	defaultTransactionCount = 5
	defaultIntervalSeconds  = 5
	defaultAmountTinybar    = 1000000 // ~0.01 HBAR
)

func main() {
	// Command line flags
	configFile := flag.String("config", "config/config.yaml", "Path to config file")
	fromAccountID := flag.String("from", "", "Account ID to send from (defaults to operator account)")
	toAccountID := flag.String("to", "", "Account ID to send to (defaults to first monitored account)")
	count := flag.Int("count", defaultTransactionCount, "Number of transactions to send")
	intervalSeconds := flag.Int("interval", defaultIntervalSeconds, "Seconds between transactions")
	amountTinybar := flag.Int64("amount", defaultAmountTinybar, "Amount in tinybar to transfer")
	network := flag.String("network", "", "Override network from config (mainnet/testnet)")

	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Determine network
	networkName := cfg.Network.Name
	if *network != "" {
		networkName = *network
	}

	// Create Hedera client
	client, err := hiero.ClientForName(networkName)
	if err != nil {
		log.Fatalf("Failed to create Hedera client: %v", err)
	}
	defer func(client *hiero.Client) {
		err := client.Close()
		if err != nil {
			log.Printf("Failed to close client: %v", err)
		}
	}(client)

	setOperator(client, cfg)
	fromAccount, toAccount := determineAccounts(cfg, fromAccountID, toAccountID)
	logConfiguration(networkName, fromAccount, toAccount, *count, *intervalSeconds, *amountTinybar)
	sendTransactions(client, fromAccount, toAccount, *count, *intervalSeconds, *amountTinybar)

	log.Println()
	log.Printf("✅ Completed %d transactions", *count)
	log.Println("Metrics should now be updating. Check ./hmon account balance <account-id> to verify")
}

// setOperator configures the client with operator credentials from config
func setOperator(client *hiero.Client, cfg *config.Config) {
	operatorAccountID, err := hiero.AccountIDFromString(cfg.Network.OperatorID)
	if err != nil {
		log.Fatalf("Invalid OPERATOR_ID: %v", err)
	}

	privateKey, err := hiero.PrivateKeyFromString(cfg.Network.OperatorKey)
	if err != nil {
		log.Fatalf("Invalid OPERATOR_KEY: %v", err)
	}

	client.SetOperator(operatorAccountID, privateKey)
}

// determineAccounts resolves the from and to account IDs, with sensible defaults
// from account defaults to operator (we have private key for it)
// to account defaults to first monitored account (to generate metrics for it)
func determineAccounts(cfg *config.Config, fromFlag, toFlag *string) (hiero.AccountID, hiero.AccountID) {
	fromID := *fromFlag
	if fromID == "" {
		fromID = cfg.Network.OperatorID
	}

	toID := *toFlag
	if toID == "" {
		if 0 < len(cfg.Accounts) {
			toID = cfg.Accounts[0].ID
		} else {
			log.Fatal("No destination account specified and no monitored accounts in config")
		}
	}

	fromAccount, err := hiero.AccountIDFromString(fromID)
	if err != nil {
		log.Fatalf("Invalid from account ID: %v", err)
	}

	toAccount, err := hiero.AccountIDFromString(toID)
	if err != nil {
		log.Fatalf("Invalid to account ID: %v", err)
	}

	return fromAccount, toAccount
}

// logConfiguration prints the transaction generator configuration
func logConfiguration(networkName string, fromAccount, toAccount hiero.AccountID, count, intervalSeconds int,
	amountTinybar int64) {
	log.Printf("Starting transaction generator")
	log.Printf("Network: %s", networkName)
	log.Printf("From: %s", fromAccount)
	log.Printf("To: %s", toAccount)
	log.Printf("Count: %d transactions", count)
	log.Printf("Interval: %d seconds", intervalSeconds)
	log.Printf("Amount: %d tinybar (%.2f HBAR)", amountTinybar,
		float64(amountTinybar)/float64(hedera.TinybarPerHbar))
	log.Println()
}

// sendTransactions sends HBAR transfers from one account to another
func sendTransactions(client *hiero.Client, fromAccount, toAccount hiero.AccountID, count, intervalSeconds int,
	amountTinybar int64) {
	for i := 1; i <= count; i++ {
		log.Printf("[%d/%d] Sending transaction...", i, count)

		txn := hiero.NewTransferTransaction().
			AddHbarTransfer(fromAccount, hiero.HbarFromTinybar(-amountTinybar)).
			AddHbarTransfer(toAccount, hiero.HbarFromTinybar(amountTinybar))

		txResponse, err := txn.Execute(client)
		if err != nil {
			log.Printf("  ❌ Failed to execute transaction: %v", err)
			continue
		}

		transactionID := txResponse.TransactionID
		receipt, err := txResponse.GetReceipt(client)
		if err != nil {
			log.Printf("  ❌ Failed to get receipt: %v", err)
		} else {
			status := receipt.Status.String()
			log.Printf("  ✅ Transaction successful (ID: %s, Status: %s)", transactionID, status)
		}

		// Wait before next transaction (except for last one)
		if i < count {
			log.Printf("  Waiting %d seconds before next transaction...\n", intervalSeconds)
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
		}
	}
}
