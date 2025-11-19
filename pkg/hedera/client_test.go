package hedera

import (
	"fmt"
	"testing"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"
)

// MockClient implements the Client interface for testing
type MockClient struct {
	mockBalance             int64
	mockInfo                *hiero.AccountInfo
	mockRecords             []Record
	mockReceipt             *hiero.TransactionReceipt
	mockExpiry              int64
	mockNodeAddressBook     *hiero.NodeAddressBook
	mockBalanceErr          error
	mockInfoErr             error
	mockRecordsErr          error
	mockReceiptErr          error
	mockExpiryErr           error
	mockNodeAddressBookErr  error
	mockCloseErr            error
	getBalanceCalls         int
	getInfoCalls            int
	getRecordsCalls         int
	getReceiptCalls         int
	getExpiryCalls          int
	getNodeAddressBookCalls int
	closeCalls              int
}

func (m *MockClient) GetAccountBalance(accountID string) (int64, error) {
	m.getBalanceCalls++
	if m.mockBalanceErr != nil {
		return 0, m.mockBalanceErr
	}
	return m.mockBalance, nil
}

func (m *MockClient) GetAccountInfo(accountID string) (*hiero.AccountInfo, error) {
	m.getInfoCalls++
	if m.mockInfoErr != nil {
		return nil, m.mockInfoErr
	}
	return m.mockInfo, nil
}

func (m *MockClient) GetAccountRecords(accountID string, limit int) ([]Record, error) {
	m.getRecordsCalls++
	if m.mockRecordsErr != nil {
		return nil, m.mockRecordsErr
	}
	return m.mockRecords, nil
}

func (m *MockClient) GetTransactionReceipt(transactionID string) (*hiero.TransactionReceipt, error) {
	m.getReceiptCalls++
	if m.mockReceiptErr != nil {
		return nil, m.mockReceiptErr
	}
	return m.mockReceipt, nil
}

func (m *MockClient) GetAccountExpiry(accountID string) (int64, error) {
	m.getExpiryCalls++
	if m.mockExpiryErr != nil {
		return 0, m.mockExpiryErr
	}
	return m.mockExpiry, nil
}

func (m *MockClient) GetNodeAddressBook() (*hiero.NodeAddressBook, error) {
	m.getNodeAddressBookCalls++
	if m.mockNodeAddressBookErr != nil {
		return nil, m.mockNodeAddressBookErr
	}
	return m.mockNodeAddressBook, nil
}

func (m *MockClient) Close() error {
	m.closeCalls++
	return m.mockCloseErr
}

// TestGetAccountBalance_ValidAccountID tests querying a valid account balance
func TestGetAccountBalance_ValidAccountID(t *testing.T) {
	mock := &MockClient{mockBalance: 1000000}

	balance, err := mock.GetAccountBalance("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if balance != 1000000 {
		t.Errorf("expected balance 1000000, got: %d", balance)
	}

	if mock.getBalanceCalls != 1 {
		t.Errorf("expected 1 call to GetAccountBalance, got: %d", mock.getBalanceCalls)
	}
}

// TestGetAccountBalance_ZeroBalance tests querying account with zero balance
func TestGetAccountBalance_ZeroBalance(t *testing.T) {
	mock := &MockClient{mockBalance: 0}

	balance, err := mock.GetAccountBalance("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if balance != 0 {
		t.Errorf("expected balance 0, got: %d", balance)
	}
}

// TestGetAccountBalance_LargeBalance tests querying account with large balance
func TestGetAccountBalance_LargeBalance(t *testing.T) {
	largeBalance := int64(50000000000000) // 50 million HBAR in tinybar

	mock := &MockClient{mockBalance: largeBalance}

	balance, err := mock.GetAccountBalance("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if balance != largeBalance {
		t.Errorf("expected balance %d, got: %d", largeBalance, balance)
	}
}

// TestGetAccountInfo_ValidAccountID tests querying valid account info
func TestGetAccountInfo_ValidAccountID(t *testing.T) {
	mockInfo := &hiero.AccountInfo{
		AccountID: hiero.AccountID{Shard: 0, Realm: 0, Account: 5000},
	}

	mock := &MockClient{mockInfo: mockInfo}

	info, err := mock.GetAccountInfo("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if info == nil {
		t.Fatal("expected info to not be nil")
	}

	if info.AccountID.Account != 5000 {
		t.Error("expected correct account info")
	}

	if mock.getInfoCalls != 1 {
		t.Errorf("expected 1 call to GetAccountInfo, got: %d", mock.getInfoCalls)
	}
}

// TestGetAccountInfo_Error tests error handling
func TestGetAccountInfo_Error(t *testing.T) {
	mockErr := fmt.Errorf("account not found")
	mock := &MockClient{mockInfoErr: mockErr}

	info, err := mock.GetAccountInfo("0.0.5000")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if info != nil {
		t.Error("expected nil info on error")
	}
}

// TestClose tests closing the client
func TestClose(t *testing.T) {
	mock := &MockClient{}

	err := mock.Close()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if mock.closeCalls != 1 {
		t.Errorf("expected 1 call to Close, got: %d", mock.closeCalls)
	}
}

// TestGetAccountRecords_ValidAccountID tests querying account records
func TestGetAccountRecords_ValidAccountID(t *testing.T) {
	mockRecords := []Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Timestamp:     1234567890,
			AmountTinyBar: 1000000,
			Type:          "CRYPTO_TRANSFER",
			Status:        "SUCCESS",
		},
		{
			TransactionID: "0.0.5000-1234567891-000000",
			Timestamp:     1234567891,
			AmountTinyBar: 2000000,
			Type:          "CRYPTO_TRANSFER",
			Status:        "SUCCESS",
		},
	}

	mock := &MockClient{mockRecords: mockRecords}

	records, err := mock.GetAccountRecords("0.0.5000", 10)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if records == nil {
		t.Error("expected records to not be nil")
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got: %d", len(records))
	}

	if mock.getRecordsCalls != 1 {
		t.Errorf("expected 1 call to GetAccountRecords, got: %d", mock.getRecordsCalls)
	}
}

// TestGetAccountRecords_EmptyRecords tests querying account with no records
func TestGetAccountRecords_EmptyRecords(t *testing.T) {
	mock := &MockClient{mockRecords: []Record{}}

	records, err := mock.GetAccountRecords("0.0.5000", 10)

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if records == nil {
		t.Error("expected empty slice, got nil")
	}

	if len(records) != 0 {
		t.Errorf("expected 0 records, got: %d", len(records))
	}
}

// TestGetAccountRecords_Error tests error handling
func TestGetAccountRecords_Error(t *testing.T) {
	mockErr := fmt.Errorf("query failed")
	mock := &MockClient{mockRecordsErr: mockErr}

	records, err := mock.GetAccountRecords("0.0.5000", 10)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if records != nil {
		t.Error("expected nil records on error")
	}
}

// TestGetTransactionReceipt_ValidTransaction tests querying transaction receipt
func TestGetTransactionReceipt_ValidTransaction(t *testing.T) {
	mockReceipt := &hiero.TransactionReceipt{
		Status: hiero.StatusSuccess,
	}

	mock := &MockClient{mockReceipt: mockReceipt}

	receipt, err := mock.GetTransactionReceipt("0.0.5000-1234567890-000000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if receipt == nil {
		t.Error("expected receipt to not be nil")
	}

	if mock.getReceiptCalls != 1 {
		t.Errorf("expected 1 call to GetTransactionReceipt, got: %d", mock.getReceiptCalls)
	}
}

// TestGetTransactionReceipt_Error tests error handling
func TestGetTransactionReceipt_Error(t *testing.T) {
	mockErr := fmt.Errorf("receipt not found")
	mock := &MockClient{mockReceiptErr: mockErr}

	receipt, err := mock.GetTransactionReceipt("0.0.5000-1234567890-000000")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if receipt != nil {
		t.Error("expected nil receipt on error")
	}
}

// TestGetAccountExpiry_ValidAccountID tests querying account expiry
func TestGetAccountExpiry_ValidAccountID(t *testing.T) {
	mockExpiry := int64(1735689600) // Future timestamp

	mock := &MockClient{mockExpiry: mockExpiry}

	expiry, err := mock.GetAccountExpiry("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if expiry != mockExpiry {
		t.Errorf("expected expiry %d, got: %d", mockExpiry, expiry)
	}

	if mock.getExpiryCalls != 1 {
		t.Errorf("expected 1 call to GetAccountExpiry, got: %d", mock.getExpiryCalls)
	}
}

// TestGetAccountExpiry_Error tests error handling
func TestGetAccountExpiry_Error(t *testing.T) {
	mockErr := fmt.Errorf("account not found")
	mock := &MockClient{mockExpiryErr: mockErr}

	expiry, err := mock.GetAccountExpiry("0.0.5000")

	if err == nil {
		t.Error("expected error, got nil")
	}

	if expiry != 0 {
		t.Errorf("expected expiry 0 on error, got: %d", expiry)
	}
}

// TestGetNodeAddressBook_ValidQuery tests querying node address book
func TestGetNodeAddressBook_ValidQuery(t *testing.T) {
	mockAddressBook := &hiero.NodeAddressBook{}

	mock := &MockClient{mockNodeAddressBook: mockAddressBook}

	book, err := mock.GetNodeAddressBook()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if book == nil {
		t.Error("expected address book to not be nil")
	}

	if mock.getNodeAddressBookCalls != 1 {
		t.Errorf("expected 1 call to GetNodeAddressBook, got: %d", mock.getNodeAddressBookCalls)
	}
}

// TestGetNodeAddressBook_Error tests error handling
func TestGetNodeAddressBook_Error(t *testing.T) {
	mockErr := fmt.Errorf("network unreachable")
	mock := &MockClient{mockNodeAddressBookErr: mockErr}

	book, err := mock.GetNodeAddressBook()

	if err == nil {
		t.Error("expected error, got nil")
	}

	if book != nil {
		t.Error("expected nil address book on error")
	}
}

// TestClientInterface_Compliance tests that the interface contract is properly defined
func TestClientInterface_Compliance(t *testing.T) {
	// This test ensures the interface has the expected methods
	var _ Client = (*MockClient)(nil)
}

// Benchmark tests for performance monitoring

// BenchmarkGetAccountBalance measures the performance of balance queries
func BenchmarkGetAccountBalance(b *testing.B) {
	mock := &MockClient{mockBalance: 1000000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetAccountBalance("0.0.5000")
	}
}

// BenchmarkGetAccountInfo measures the performance of account info queries
func BenchmarkGetAccountInfo(b *testing.B) {
	mockInfo := &hiero.AccountInfo{
		AccountID: hiero.AccountID{Shard: 0, Realm: 0, Account: 5000},
	}
	mock := &MockClient{mockInfo: mockInfo}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetAccountInfo("0.0.5000")
	}
}

// BenchmarkGetNodeAddressBook measures the performance of node address book queries
func BenchmarkGetNodeAddressBook(b *testing.B) {
	mockAddressBook := &hiero.NodeAddressBook{}
	mock := &MockClient{mockNodeAddressBook: mockAddressBook}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetNodeAddressBook()
	}
}

// BenchmarkGetAccountRecords measures the performance of account records queries
func BenchmarkGetAccountRecords(b *testing.B) {
	mockRecords := []Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Timestamp:     1234567890,
			AmountTinyBar: 1000000,
			Type:          "CRYPTO_TRANSFER",
			Status:        "SUCCESS",
		},
	}
	mock := &MockClient{mockRecords: mockRecords}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetAccountRecords("0.0.5000", 10)
	}
}

// BenchmarkGetTransactionReceipt measures the performance of transaction receipt queries
func BenchmarkGetTransactionReceipt(b *testing.B) {
	mockReceipt := &hiero.TransactionReceipt{
		Status: hiero.StatusSuccess,
	}
	mock := &MockClient{mockReceipt: mockReceipt}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetTransactionReceipt("0.0.5000-1234567890-000000")
	}
}

// BenchmarkGetAccountExpiry measures the performance of account expiry queries
func BenchmarkGetAccountExpiry(b *testing.B) {
	mock := &MockClient{mockExpiry: 1735689600}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mock.GetAccountExpiry("0.0.5000")
	}
}

// Integration-style tests (would need actual SDK setup)

// TestNewClient_ValidNetwork tests client creation with valid network
// NOTE: This test requires OPERATOR_ID and OPERATOR_KEY environment variables
// For now, it's commented out. Uncomment and set env vars to test.
/*
func TestNewClient_ValidNetwork(t *testing.T) {
	// Set required environment variables
	t.Setenv("OPERATOR_ID", "0.0.3")
	t.Setenv("OPERATOR_KEY", "your_private_key")

	client, err := NewClient("testnet")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if client == nil {
		t.Error("expected client to not be nil")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("expected no error closing client, got: %v", err)
	}
}
*/

// TestNewClient_MissingOperatorID tests client creation without OPERATOR_ID
// NOTE: This test requires actual SDK integration to verify
/*
func TestNewClient_MissingOperatorID(t *testing.T) {
	t.Setenv("OPERATOR_ID", "")
	t.Setenv("OPERATOR_KEY", "key123")

	client, err := NewClient("testnet")

	if err == nil {
		t.Error("expected error for missing OPERATOR_ID")
	}

	if client != nil {
		t.Error("expected client to be nil on error")
	}
}
*/

// TestNewClient_MissingOperatorKey tests client creation without OPERATOR_KEY
/*
func TestNewClient_MissingOperatorKey(t *testing.T) {
	t.Setenv("OPERATOR_ID", "0.0.3")
	t.Setenv("OPERATOR_KEY", "")

	client, err := NewClient("testnet")

	if err == nil {
		t.Error("expected error for missing OPERATOR_KEY")
	}

	if client != nil {
		t.Error("expected client to be nil on error")
	}
}
*/
