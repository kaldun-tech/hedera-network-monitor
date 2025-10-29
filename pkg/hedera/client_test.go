package hedera

import (
	"testing"
)

// MockClient implements the Client interface for testing
type MockClient struct {
	mockBalance      int64
	mockInfo         map[string]interface{}
	mockNetworkInfo  map[string]interface{}
	mockBalanceErr   error
	mockInfoErr      error
	mockNetworkErr   error
	mockCloseErr     error
	getBalanceCalls  int
	getInfoCalls     int
	getNetworkCalls  int
	closeCalls       int
}

func (m *MockClient) GetAccountBalance(accountID string) (int64, error) {
	m.getBalanceCalls++
	if m.mockBalanceErr != nil {
		return 0, m.mockBalanceErr
	}
	return m.mockBalance, nil
}

func (m *MockClient) GetAccountInfo(accountID string) (map[string]interface{}, error) {
	m.getInfoCalls++
	if m.mockInfoErr != nil {
		return nil, m.mockInfoErr
	}
	return m.mockInfo, nil
}

func (m *MockClient) GetNetworkInfo() (map[string]interface{}, error) {
	m.getNetworkCalls++
	if m.mockNetworkErr != nil {
		return nil, m.mockNetworkErr
	}
	return m.mockNetworkInfo, nil
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
	mockInfo := map[string]interface{}{
		"account_id": "0.0.5000",
		"balance":    1000000,
		"is_deleted": false,
	}

	mock := &MockClient{mockInfo: mockInfo}

	info, err := mock.GetAccountInfo("0.0.5000")

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if info == nil {
		t.Error("expected info to not be nil")
	}

	if accountID, ok := info["account_id"]; !ok || accountID != "0.0.5000" {
		t.Error("expected account_id in info")
	}

	if mock.getInfoCalls != 1 {
		t.Errorf("expected 1 call to GetAccountInfo, got: %d", mock.getInfoCalls)
	}
}

// TestGetNetworkInfo tests querying network information
func TestGetNetworkInfo(t *testing.T) {
	mockNetInfo := map[string]interface{}{
		"network": "testnet",
		"nodes":   3,
	}

	mock := &MockClient{mockNetworkInfo: mockNetInfo}

	info, err := mock.GetNetworkInfo()

	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if info == nil {
		t.Error("expected info to not be nil")
	}

	if mock.getNetworkCalls != 1 {
		t.Errorf("expected 1 call to GetNetworkInfo, got: %d", mock.getNetworkCalls)
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
		mock.GetAccountBalance("0.0.5000")
	}
}

// BenchmarkGetAccountInfo measures the performance of account info queries
func BenchmarkGetAccountInfo(b *testing.B) {
	mockInfo := map[string]interface{}{
		"account_id": "0.0.5000",
		"balance":    1000000,
	}
	mock := &MockClient{mockInfo: mockInfo}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.GetAccountInfo("0.0.5000")
	}
}

// BenchmarkGetNetworkInfo measures the performance of network info queries
func BenchmarkGetNetworkInfo(b *testing.B) {
	mockNetInfo := map[string]interface{}{
		"network": "testnet",
		"nodes":   3,
	}
	mock := &MockClient{mockNetworkInfo: mockNetInfo}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.GetNetworkInfo()
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
