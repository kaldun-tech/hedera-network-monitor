package hedera

import (
	"fmt"
	"os"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/logger"
)

// TinybarPerHbar is the conversion constant: 1 HBAR = 100,000,000 tinybar
const TinybarPerHbar = 100_000_000
const getAddressBookMaxAttempts = 5

// Record represents a transaction record for an account
type Record struct {
	TransactionID string
	Timestamp     int64
	AmountTinyBar int64
	Type          TransactionType
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
// operatorID and operatorKey can come from config file or environment variables
func NewClient(network, operatorID, operatorKey string) (Client, error) {
	logger.Info("Creating Hedera client", "network", network)
	client, err := hiero.ClientForName(network)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Use provided credentials, fallback to environment variables
	if operatorID == "" {
		operatorID = os.Getenv("OPERATOR_ID")
	}
	if operatorKey == "" {
		operatorKey = os.Getenv("OPERATOR_KEY")
	}

	// Validate credentials
	if operatorID == "" || operatorKey == "" {
		return nil, fmt.Errorf("OPERATOR_ID and OPERATOR_KEY must be provided in config file or environment variables")
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
	logger.Debug("Querying balance", "account_id", accountID)
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
	logger.Debug("Querying account info", "account_id", accountID)
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
		Type:          GetTransactionType(&nextRec),
		Status:        nextRec.Receipt.Status.String(),
	}
}

// GetAccountRecords implements Client interface
func (hc *HederaClient) GetAccountRecords(accountID string, limit int) ([]Record, error) {
	// Parse accountID
	logger.Debug("Querying account records", "account_id", accountID, "limit", limit)
	parsedAccount, err := getAccount(accountID)
	if err != nil {
		return nil, fmt.Errorf("invalid accountID: %w", err)
	}

	query := hiero.NewAccountRecordsQuery().
		SetAccountID(parsedAccount)
	records, err := query.Execute(hc.client)
	if err != nil {
		return nil, fmt.Errorf("error retrieving records: %w", err)
	}

	// Convert hiero records to Record struct slice, respecting the limit
	result := make([]Record, 0, min(len(records), limit))
	for i, nextRec := range records {
		if i >= limit {
			break
		}
		nextRes := buildRecordStruct(nextRec)
		result = append(result, nextRes)
	}

	return result, nil
}

// GetTransactionReceipt implements Client interface
func (hc *HederaClient) GetTransactionReceipt(transactionID string) (*hiero.TransactionReceipt, error) {
	logger.Debug("Querying transaction receipt", "transaction_id", transactionID)
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
	logger.Debug("Querying account expiry", "account_id", accountID)
	info, err := hc.GetAccountInfo(accountID)
	if err != nil {
		return 0, fmt.Errorf("error retrieving account info: %w", err)
	}
	return info.ExpirationTime.Unix(), nil
}

// GetNodeAddressBook implements Client interface
func (hc *HederaClient) GetNodeAddressBook() (*hiero.NodeAddressBook, error) {
	logger.Debug("Querying node address book")
	// Address book is stored in file 0.0.102 on all Hedera networks
	addressBookFileID, _ := hiero.FileIDFromString("0.0.102")
	query := hiero.NewAddressBookQuery().
		SetFileID(addressBookFileID).
		SetMaxAttempts(getAddressBookMaxAttempts)
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
