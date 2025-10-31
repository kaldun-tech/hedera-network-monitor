package hedera

import (
	"fmt"
	"log"
	"os"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
)

// Record represents a transaction record for an account
type Record struct {
	TransactionID string
	Timestamp     int64
	AmountTinyBar int64
	Type          string
	Status        string
}

// Client is a wrapper around the Hedera SDK client
type Client interface {
	// GetAccountBalance retrieves the balance for a given account in tinybar
	GetAccountBalance(accountID string) (int64, error)

	// GetAccountInfo retrieves detailed information about an account
	GetAccountInfo(accountID string) (*hiero.AccountInfo, error)

	// GetAccountRecords retrieves recent transaction records for an account
	// limit: maximum number of records to return
	GetAccountRecords(accountID string, limit int) ([]Record, error)

	// GetTransactionReceipt retrieves the receipt for a specific transaction
	GetTransactionReceipt(transactionID string) (*hiero.TransactionReceipt, error)

	// GetAccountExpiry retrieves the auto-renew expiry timestamp for an account
	GetAccountExpiry(accountID string) (int64, error)

	// GetNodeAddressBook retrieves information about network nodes
	GetNodeAddressBook() (*hiero.NodeAddressBook, error)

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

func getTransactionID(transactionID string) (hiero.TransactionID, error) {
	return hiero.TransactionIdFromString(transactionID)
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
func (hc *HederaClient) GetAccountInfo(accountID string) (*hiero.AccountInfo, error) {
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

	return &info, nil
}

func getTransactionType(rec *hiero.TransactionRecord) string {
	if rec.CallResult != nil {
		if rec.CallResultIsCreate {
			return "ContractCreate"
		}
		return "ContractCall"
	}
	if len(rec.TokenTransfers) > 0 {
		return "TokenTransfer"
	}
	if len(rec.Transfers) > 0 {
		return "CryptoTransfer"
	}
	if rec.Receipt.TopicID != nil {
		return "ConsensusSubmitMessage"
	}
	if rec.Receipt.FileID != nil {
		return "FileOperation"
	}
	return "Unknown"
}

func buildRecordStruct(nextRec hiero.TransactionRecord) Record {
	// Get amount from transfers as tiny bar
	var amountTinyBar int64
	if nextRec.Transfers != nil {
		// Transfers is typically a map[AccountID]Hbar
		for _, transfer := range nextRec.Transfers {
			amountTinyBar += transfer.Amount.AsTinybar()
		}
	}

	return Record{
		TransactionID: nextRec.TransactionID.String(),
		Timestamp:     nextRec.ConsensusTimestamp.Unix(),
		AmountTinyBar: amountTinyBar,
		Type:          getTransactionType(&nextRec),
		Status:        nextRec.Receipt.Status.String(),
	}
}

// GetAccountRecords implements Client interface
func (hc *HederaClient) GetAccountRecords(accountID string, limit int) ([]Record, error) {
	// Parse accountID
	log.Printf("Querying account records for: %s (limit: %d)", accountID, limit)
	parsedAccount, err := getAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid accountID: %w", err)
	}

	query := hiero.NewAccountRecordsQuery()
	query.SetAccountID(parsedAccount)
	records, err := query.Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("error retrieving records: %w", err)
	}

	// Convert hiero records to Record struct slice
	result := make([]Record, 0, len(records))
	for _, nextRec := range records {
		nextRes := buildRecordStruct(nextRec)
		result = append(result, nextRes)
	}

	return result, nil
}

// GetTransactionReceipt implements Client interface
func (hc *HederaClient) GetTransactionReceipt(transactionID string) (*hiero.TransactionReceipt, error) {
	log.Printf("Querying transaction receipt for: %s", transactionID)
	// Parse transactionID, execute query
	tID, err := getTransactionID(transactionID)
	if err != nil {
		return nil, fmt.Errorf("error parsing transaction ID: %w", err)
	}

	query := hiero.NewTransactionReceiptQuery().SetTransactionID(tID)
	receipt, err := query.Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("error retrieving transaction receipt: %w", err)
	}

	return &receipt, nil
}

// GetAccountExpiry implements Client interface
func (hc *HederaClient) GetAccountExpiry(accountID string) (int64, error) {
	log.Printf("Querying account expiry for: %s", accountID)
	info, err := hc.GetAccountInfo(accountID)
	if err != nil {
		return 0, fmt.Errorf("error retrieving account info: %w", err)
	}
	return info.ExpirationTime.Unix(), nil
}

// GetNodeAddressBook implements Client interface
func (hc *HederaClient) GetNodeAddressBook() (*hiero.NodeAddressBook, error) {
	log.Println("Querying node address book")
	query := hiero.NewAddressBookQuery().SetMaxAttempts(5)
	addressBook, err := query.Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("error retrieving address book query: %w", err)
	}
	return &addressBook, nil
}

// Close implements Client interface
func (hc *HederaClient) Close() error {
	return hc.client.Close()
}
