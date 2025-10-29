package hedera

import (
	"fmt"
	"log"
	"os"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
)

// Client is a wrapper around the Hedera SDK client
type Client interface {
	// GetAccountBalance retrieves the balance for a given account in tinybar
	GetAccountBalance(accountID string) (int64, error)

	// GetAccountInfo retrieves detailed information about an account
	GetAccountInfo(accountID string) (map[string]interface{}, error)

	// GetNetworkInfo retrieves information about the Hedera network
	GetNetworkInfo() (map[string]interface{}, error)

	// Close closes the Hedera client connection
	Close() error
}

type HederaClient struct {
	client *hiero.Client
}

// NewClient creates a new Hedera SDK client wrapper
func NewClient(network string) (Client, error) {
	log.Printf("Creating Hedera client for network: %s", network)
	client, err := hiero.ClientForName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Get operator credentials from environment
	operatorID := os.Getenv("OPERATOR_ID")
	operatorKey := os.Getenv("OPERATOR_KEY")

	// Validate environment variables
	if operatorID == "" || operatorKey == "" {
		return nil, fmt.Errorf("OPERATOR_ID and OPERATOR_KEY environment variables required")
	}

	// Parse credentials
	operatorAccountID, err := hiero.AccountIDFromString(operatorID)
	if err != nil {
		return nil, fmt.Errorf("invalid OPERATOR_ID: %w", err)
	}

	privateKey, err := hiero.PrivateKeyFromString(operatorKey)
	if err != nil {
		return nil, fmt.Errorf("invalid OPERATOR_KEY: %w", err)
	}

	// Set the client operator ID and key
	client.SetOperator(operatorAccountID, privateKey)

	return &HederaClient{client: client}, nil
}

func getAccount(accountID string) (hiero.AccountID, error) {
	return hiero.AccountIDFromString(accountID)
}

// GetAccountBalance implements Client interface
func (hc *HederaClient) GetAccountBalance(accountID string) (int64, error) {
	log.Printf("Querying balance for account: %s", accountID)
	parsedAccount, err := getAccount(accountID)
	if err != nil {
		return 0, fmt.Errorf("invalid accountID: %w", err)
	}

	query := hiero.NewAccountBalanceQuery()
	query.SetAccountID(parsedAccount)

	balance, err := query.Execute(hc.client)
	if err != nil {
		return 0, fmt.Errorf("failed to execute balance query: %w", err)
	}

	return balance.Hbars.AsTinybar(), nil
}

// GetAccountInfo implements Client interface
func (hc *HederaClient) GetAccountInfo(accountID string) (map[string]interface{}, error) {
	log.Printf("Querying account info for: %s", accountID)
	parsedAccount, err := getAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid accountID: %w", err)
	}

	query := hiero.NewAccountInfoQuery().
		SetAccountID(parsedAccount)
	info, err := query.Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute info query: %w", err)
	}

	return map[string]interface{}{
		info.AccountID.String(): info,
	}, err
}

// GetNetworkInfo implements Client interface
func (hc *HederaClient) GetNetworkInfo() (map[string]interface{}, error) {
	// 1. List of available nodes
	// 2. Address book
	// TODO: consider what else to retrieve
	// 3. Current consensus node information
	// 4. Network fee information

	// Empty map
	result := make(map[string]interface{})

	result["nodes"] = hc.client.GetNetwork()

	addressBook, err := hiero.NewAddressBookQuery().
		SetMaxAttempts(5).
		Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute address book query: %w", err)
	}
	result["addressBook"] = addressBook

	return result, nil
}

// Close implements Client interface
func (hc *HederaClient) Close() error {
	return hc.client.Close()
}
