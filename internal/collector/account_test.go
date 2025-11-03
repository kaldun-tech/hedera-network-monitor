package collector

import (
	"testing"
	"time"

	hiero "github.com/hiero-ledger/hiero-sdk-go/v2/sdk"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
)

// TestParseInterval_EmptyString tests parsing an empty interval string
func TestParseInterval_EmptyString(t *testing.T) {
	result := ParseInterval("")
	expected := DefaultInterval

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestParseInterval_ValidInterval tests parsing a valid interval string
func TestParseInterval_ValidInterval(t *testing.T) {
	result := ParseInterval("60")
	expected := 60 * time.Second

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestParseInterval_ValidIntervalWithText tests parsing interval with extra text
func TestParseInterval_ValidIntervalWithText(t *testing.T) {
	result := ParseInterval("45 seconds")
	expected := 45 * time.Second

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestParseInterval_InvalidFormat tests parsing with invalid format
func TestParseInterval_InvalidFormat(t *testing.T) {
	result := ParseInterval("not a number")
	expected := DefaultInterval

	if result != expected {
		t.Errorf("expected default %v, got %v", expected, result)
	}
}

// TestParseInterval_ZeroValue tests parsing with zero value (should use default)
func TestParseInterval_ZeroValue(t *testing.T) {
	result := ParseInterval("0")
	expected := DefaultInterval

	if result != expected {
		t.Errorf("expected default %v for zero value, got %v", expected, result)
	}
}

// TestParseInterval_NegativeValue tests parsing with negative value (should use default)
func TestParseInterval_NegativeValue(t *testing.T) {
	result := ParseInterval("-10")
	expected := DefaultInterval

	if result != expected {
		t.Errorf("expected default %v for negative value, got %v", expected, result)
	}
}

// TestParseInterval_LargeValue tests parsing with large interval value
func TestParseInterval_LargeValue(t *testing.T) {
	result := ParseInterval("3600")
	expected := 3600 * time.Second

	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

// TestBuildTransactionTypeMetric_EmptyRecords tests building metrics from empty records
func TestBuildTransactionTypeMetric_EmptyRecords(t *testing.T) {
	collector := &AccountCollector{}
	records := []hedera.Record{}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	if len(metrics) != 0 {
		t.Errorf("expected 0 metrics for empty records, got %d", len(metrics))
	}
}

// TestBuildTransactionTypeMetric_SingleType tests building metrics with one transaction type
func TestBuildTransactionTypeMetric_SingleType(t *testing.T) {
	collector := &AccountCollector{}
	records := []hedera.Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Timestamp:     1234567890,
			AmountTinyBar: 1000000,
			Type:          hedera.TransactionTypeCryptoTransfer,
			Status:        "SUCCESS",
		},
		{
			TransactionID: "0.0.5000-1234567891-000000",
			Timestamp:     1234567891,
			AmountTinyBar: 2000000,
			Type:          hedera.TransactionTypeCryptoTransfer,
			Status:        "SUCCESS",
		},
	}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	if len(metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]
	if metric.Value != 2.0 {
		t.Errorf("expected count 2.0, got %v", metric.Value)
	}

	if metric.Labels["transaction_type"] != hedera.TransactionTypeCryptoTransfer.String() {
		t.Errorf("expected transaction_type %s, got %s", hedera.TransactionTypeCryptoTransfer.String(), metric.Labels["transaction_type"])
	}
}

// TestBuildTransactionTypeMetric_MultipleTypes tests building metrics with multiple transaction types
func TestBuildTransactionTypeMetric_MultipleTypes(t *testing.T) {
	collector := &AccountCollector{}
	records := []hedera.Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Timestamp:     1234567890,
			AmountTinyBar: 1000000,
			Type:          hedera.TransactionTypeCryptoTransfer,
			Status:        "SUCCESS",
		},
		{
			TransactionID: "0.0.5000-1234567891-000000",
			Timestamp:     1234567891,
			AmountTinyBar: 2000000,
			Type:          hedera.TransactionTypeTokenTransfer,
			Status:        "SUCCESS",
		},
		{
			TransactionID: "0.0.5000-1234567892-000000",
			Timestamp:     1234567892,
			AmountTinyBar: 3000000,
			Type:          hedera.TransactionTypeCryptoTransfer,
			Status:        "SUCCESS",
		},
		{
			TransactionID: "0.0.5000-1234567893-000000",
			Timestamp:     1234567893,
			AmountTinyBar: 0,
			Type:          hedera.TransactionTypeContractCall,
			Status:        "SUCCESS",
		},
	}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	if len(metrics) != 3 {
		t.Errorf("expected 3 metrics (3 types), got %d", len(metrics))
	}

	// Build a map of transaction type -> count for easier testing
	typeCountMap := make(map[string]float64)
	for _, metric := range metrics {
		typeCountMap[metric.Labels["transaction_type"]] = metric.Value
	}

	if typeCountMap[hedera.TransactionTypeCryptoTransfer.String()] != 2.0 {
		t.Errorf("expected CryptoTransfer count 2.0, got %v", typeCountMap[hedera.TransactionTypeCryptoTransfer.String()])
	}

	if typeCountMap[hedera.TransactionTypeTokenTransfer.String()] != 1.0 {
		t.Errorf("expected TokenTransfer count 1.0, got %v", typeCountMap[hedera.TransactionTypeTokenTransfer.String()])
	}

	if typeCountMap[hedera.TransactionTypeContractCall.String()] != 1.0 {
		t.Errorf("expected ContractCall count 1.0, got %v", typeCountMap[hedera.TransactionTypeContractCall.String()])
	}
}

// TestBuildTransactionTypeMetric_AllTypes tests with all transaction types
func TestBuildTransactionTypeMetric_AllTypes(t *testing.T) {
	collector := &AccountCollector{}
	records := []hedera.Record{
		{Type: hedera.TransactionTypeCryptoTransfer},
		{Type: hedera.TransactionTypeTokenTransfer},
		{Type: hedera.TransactionTypeContractCreate},
		{Type: hedera.TransactionTypeContractCall},
		{Type: hedera.TransactionTypeConsensusSubmitMessage},
		{Type: hedera.TransactionTypeFileOperation},
		{Type: hedera.TransactionTypeUnknown},
	}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	if len(metrics) != 7 {
		t.Errorf("expected 7 metrics for all types, got %d", len(metrics))
	}

	// Verify each metric has correct count of 1
	for _, metric := range metrics {
		if metric.Value != 1.0 {
			t.Errorf("expected each count to be 1.0, got %v for type %s", metric.Value, metric.Labels["transaction_type"])
		}
	}
}

// TestBuildTransactionTypeMetric_Labels tests that metrics have correct labels
func TestBuildTransactionTypeMetric_Labels(t *testing.T) {
	collector := &AccountCollector{}
	accountID := "0.0.5000"
	label := "Test Account"
	records := []hedera.Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Timestamp:     1234567890,
			AmountTinyBar: 1000000,
			Type:          hedera.TransactionTypeCryptoTransfer,
			Status:        "SUCCESS",
		},
	}

	metrics := collector.buildTransactionTypeMetric(records, accountID, label)

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]

	// Check metric name
	if metric.Name != "account_transaction_type_count" {
		t.Errorf("expected metric name 'account_transaction_type_count', got '%s'", metric.Name)
	}

	// Check labels
	if metric.Labels["account_id"] != accountID {
		t.Errorf("expected account_id '%s', got '%s'", accountID, metric.Labels["account_id"])
	}

	if metric.Labels["label"] != label {
		t.Errorf("expected label '%s', got '%s'", label, metric.Labels["label"])
	}

	if metric.Labels["transaction_type"] != hedera.TransactionTypeCryptoTransfer.String() {
		t.Errorf("expected transaction_type '%s', got '%s'", hedera.TransactionTypeCryptoTransfer.String(), metric.Labels["transaction_type"])
	}
}

// TestBuildTransactionTypeMetric_Timestamp tests that metrics have recent timestamps
func TestBuildTransactionTypeMetric_Timestamp(t *testing.T) {
	collector := &AccountCollector{}
	beforeTime := time.Now().Unix()

	records := []hedera.Record{
		{
			TransactionID: "0.0.5000-1234567890-000000",
			Type:          hedera.TransactionTypeCryptoTransfer,
		},
	}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	afterTime := time.Now().Unix()

	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}

	metric := metrics[0]

	// Timestamp should be recent (between beforeTime and afterTime)
	if metric.Timestamp < beforeTime || metric.Timestamp > afterTime+1 {
		t.Errorf("timestamp %d is not within expected range [%d, %d]", metric.Timestamp, beforeTime, afterTime+1)
	}
}

// TestBuildTransactionTypeMetric_MetricType tests that returned metrics are valid types.Metric
func TestBuildTransactionTypeMetric_MetricType(t *testing.T) {
	collector := &AccountCollector{}
	records := []hedera.Record{
		{Type: hedera.TransactionTypeCryptoTransfer},
	}

	metrics := collector.buildTransactionTypeMetric(records, "0.0.5000", "Test Account")

	if len(metrics) == 0 {
		t.Fatal("expected at least one metric")
	}

	// This is just verifying the type is correct - Go's type system will catch real errors
	var _ types.Metric = metrics[0]
}

// MockClient is a mock implementation of the hedera.Client interface for testing
type MockClient struct {
	mockRecords []hedera.Record
	mockErr     error
}

func (m *MockClient) GetAccountBalance(accountID string) (int64, error) {
	return 0, m.mockErr
}

func (m *MockClient) GetAccountInfo(accountID string) (*hiero.AccountInfo, error) {
	return nil, m.mockErr
}

func (m *MockClient) GetAccountRecords(accountID string, limit int) ([]hedera.Record, error) {
	if m.mockErr != nil {
		return nil, m.mockErr
	}
	return m.mockRecords, nil
}

func (m *MockClient) GetTransactionReceipt(transactionID string) (*hiero.TransactionReceipt, error) {
	return nil, m.mockErr
}

func (m *MockClient) GetAccountExpiry(accountID string) (int64, error) {
	return 0, m.mockErr
}

func (m *MockClient) GetNodeAddressBook() (*hiero.NodeAddressBook, error) {
	return nil, m.mockErr
}

func (m *MockClient) Close() error {
	return m.mockErr
}

// TestNewAccountCollector_Initialization tests account collector creation
func TestNewAccountCollector_Initialization(t *testing.T) {
	// Mock client
	mockClient := &MockClient{}
	accounts := []AccountConfig{
		{ID: "0.0.5000", Label: "Account 1"},
		{ID: "0.0.5001", Label: "Account 2"},
	}

	collector := NewAccountCollector(mockClient, accounts)

	if collector == nil {
		t.Fatal("expected collector to be created")
	}

	if len(collector.accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(collector.accounts))
	}

	if collector.accounts[0].ID != "0.0.5000" {
		t.Errorf("expected first account ID to be '0.0.5000', got '%s'", collector.accounts[0].ID)
	}

	if collector.interval == 0 {
		t.Error("expected interval to be set")
	}
}
