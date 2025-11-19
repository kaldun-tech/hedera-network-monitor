package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// MockStorage is a mock implementation of the Storage interface for testing
type MockStorage struct {
	metrics        []types.Metric
	getMetricsErr  error
	getByLabelErr  error
	storeMetricErr error
	deleteOldErr   error
	closeErr       error
	statsErr       error
}

// MockAlertManager is a mock implementation of the AlertManager interface for testing
type MockAlertManager struct {
	rules           []alerting.AlertRule
	addRuleErr      error
	removeRuleErr   error
	getRulesErr     error
	addRuleCalls    int                 // Track how many times AddRule was called
	removeRuleCalls int                 // Track how many times RemoveRule was called
	getRulesCalls   int                 // Track how many times GetRules was called
	lastAddedRule   *alerting.AlertRule // Track the last rule that was added
	lastRemovedID   string              // Track the last rule ID that was removed
}

// GetRules returns all configured alert rules
func (m *MockAlertManager) GetRules() []alerting.AlertRule {
	m.getRulesCalls++
	if m.getRulesErr != nil {
		return nil
	}
	// Return a copy to prevent external modification
	rules := make([]alerting.AlertRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// AddRule adds a new alert rule
func (m *MockAlertManager) AddRule(rule alerting.AlertRule) error {
	m.addRuleCalls++
	m.lastAddedRule = &rule
	if m.addRuleErr != nil {
		return m.addRuleErr
	}
	m.rules = append(m.rules, rule)
	return nil
}

// RemoveRule removes an alert rule by ID
func (m *MockAlertManager) RemoveRule(ruleID string) error {
	m.removeRuleCalls++
	m.lastRemovedID = ruleID
	if m.removeRuleErr != nil {
		return m.removeRuleErr
	}

	for i, rule := range m.rules {
		if rule.ID == ruleID {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}

	return alerting.ErrRuleNotFound
}

func (m *MockStorage) StoreMetric(metric types.Metric) error {
	if m.storeMetricErr != nil {
		return m.storeMetricErr
	}
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *MockStorage) GetMetrics(name string, limit int) ([]types.Metric, error) {
	if m.getMetricsErr != nil {
		return nil, m.getMetricsErr
	}

	result := make([]types.Metric, 0)
	for _, metric := range m.metrics {
		if name != "" && metric.Name != name {
			continue
		}
		result = append(result, metric)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *MockStorage) GetMetricsByLabel(key, value string) ([]types.Metric, error) {
	if m.getByLabelErr != nil {
		return nil, m.getByLabelErr
	}

	result := make([]types.Metric, 0)
	for _, metric := range m.metrics {
		if metricValue, exists := metric.Labels[key]; exists && metricValue == value {
			result = append(result, metric)
		}
	}
	return result, nil
}

func (m *MockStorage) DeleteOldMetrics(beforeTimestamp int64) error {
	return m.deleteOldErr
}

func (m *MockStorage) Close() error {
	return m.closeErr
}

// Stats returns storage statistics (only if supportsStats is true)
func (m *MockStorage) Stats() (map[string]interface{}, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return map[string]interface{}{
		"metric_count": len(m.metrics),
		"max_size":     10000,
		"utilization":  fmt.Sprintf("%.2f%%", float64(len(m.metrics))/float64(10000)*100),
	}, nil
}

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	store := &MockStorage{}
	alertManager := &MockAlertManager{}
	server := NewServer(8080, store, alertManager)

	if server == nil {
		t.Fatal("expected server to be created")
	}

	if server.port != 8080 {
		t.Errorf("expected port 8080, got %d", server.port)
	}

	if server.store != store {
		t.Error("expected store to be set correctly")
	}
}

// TestHandleHealth tests the /health endpoint
func TestHandleHealth_Success(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", response.Status)
	}

	if response.Version != "0.1.0" {
		t.Errorf("expected version '0.1.0', got '%s'", response.Version)
	}
}

// TestHandleHealth_MethodNotAllowed tests health endpoint with wrong method
func TestHandleHealth_MethodNotAllowed(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("POST", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleMetrics_Success tests retrieving all metrics
func TestHandleMetrics_Success(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{
				Name:      "account_balance",
				Value:     1000000,
				Timestamp: 1234567890,
				Labels: map[string]string{
					"account_id": "0.0.5000",
				},
			},
			{
				Name:      "account_balance",
				Value:     2000000,
				Timestamp: 1234567891,
				Labels: map[string]string{
					"account_id": "0.0.5001",
				},
			},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 2 {
		t.Errorf("expected 2 metrics, got %d", response.Count)
	}

	if len(response.Metrics) != 2 {
		t.Errorf("expected 2 metrics in array, got %d", len(response.Metrics))
	}
}

// TestHandleMetrics_WithNameFilter tests retrieving metrics with name filter
func TestHandleMetrics_WithNameFilter(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{
				Name:      "account_balance",
				Value:     1000000,
				Timestamp: 1234567890,
				Labels:    map[string]string{"account_id": "0.0.5000"},
			},
			{
				Name:      "network_status",
				Value:     1,
				Timestamp: 1234567891,
				Labels:    map[string]string{},
			},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics?name=account_balance", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 metric, got %d", response.Count)
	}

	if response.Metrics[0].Name != "account_balance" {
		t.Errorf("expected metric name 'account_balance', got '%s'", response.Metrics[0].Name)
	}
}

// TestHandleMetrics_WithLimit tests retrieving metrics with limit
func TestHandleMetrics_WithLimit(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{Name: "m1", Value: 1, Timestamp: 1, Labels: map[string]string{}},
			{Name: "m2", Value: 2, Timestamp: 2, Labels: map[string]string{}},
			{Name: "m3", Value: 3, Timestamp: 3, Labels: map[string]string{}},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics?limit=2", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 2 {
		t.Errorf("expected 2 metrics, got %d", response.Count)
	}
}

// TestHandleMetrics_InvalidLimit_Default tests invalid limit uses default
func TestHandleMetrics_InvalidLimit_Default(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{Name: "m1", Value: 1, Timestamp: 1, Labels: map[string]string{}},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics?limit=invalid", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should use default limit (no error in response)
	if response.Error != "" {
		t.Errorf("expected no error for invalid limit, got: %s", response.Error)
	}
}

// TestHandleMetrics_LimitTooHigh_Capped tests limit exceeding max is capped
func TestHandleMetrics_LimitTooHigh_Capped(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{Name: "m1", Value: 1, Timestamp: 1, Labels: map[string]string{}},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics?limit=99999", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should use max limit
	if response.Error != "" {
		t.Errorf("expected no error for too-high limit, got: %s", response.Error)
	}
}

// TestHandleMetrics_EmptyResult tests metrics endpoint with no metrics
func TestHandleMetrics_EmptyResult(t *testing.T) {
	store := &MockStorage{metrics: []types.Metric{}}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("expected 0 metrics, got %d", response.Count)
	}

	if response.Metrics == nil {
		t.Error("expected metrics array to not be nil (should be empty slice)")
	}
}

// TestHandleMetrics_StorageError tests metrics endpoint when storage fails
func TestHandleMetrics_StorageError(t *testing.T) {
	store := &MockStorage{
		getMetricsErr: fmt.Errorf("storage error"),
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleMetrics_MethodNotAllowed tests metrics endpoint with wrong method
func TestHandleMetrics_MethodNotAllowed(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("POST", "/api/v1/metrics", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestHandleMetricsByLabel_Success tests retrieving metrics by label
func TestHandleMetricsByLabel_Success(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{
				Name:      "account_balance",
				Value:     1000000,
				Timestamp: 1234567890,
				Labels: map[string]string{
					"account_id": "0.0.5000",
					"label":      "Account 1",
				},
			},
			{
				Name:      "account_balance",
				Value:     2000000,
				Timestamp: 1234567891,
				Labels: map[string]string{
					"account_id": "0.0.5001",
					"label":      "Account 2",
				},
			},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics/account?key=account_id&value=0.0.5000", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 1 {
		t.Errorf("expected 1 metric, got %d", response.Count)
	}

	if response.Metrics[0].Labels["account_id"] != "0.0.5000" {
		t.Errorf("expected account_id '0.0.5000', got '%s'", response.Metrics[0].Labels["account_id"])
	}
}

// TestHandleMetricsByLabel_MissingKey tests endpoint with missing key parameter
func TestHandleMetricsByLabel_MissingKey(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics/account?value=0.0.5000", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleMetricsByLabel_MissingValue tests endpoint with missing value parameter
func TestHandleMetricsByLabel_MissingValue(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics/account?key=account_id", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleMetricsByLabel_NoMatches tests endpoint when no metrics match
func TestHandleMetricsByLabel_NoMatches(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{
				Name:      "account_balance",
				Value:     1000000,
				Timestamp: 1234567890,
				Labels: map[string]string{
					"account_id": "0.0.5000",
				},
			},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics/account?key=account_id&value=0.0.9999", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("expected 0 metrics, got %d", response.Count)
	}

	if response.Metrics == nil {
		t.Error("expected metrics array to not be nil")
	}
}

// TestHandleMetricsByLabel_StorageError tests endpoint when storage fails
func TestHandleMetricsByLabel_StorageError(t *testing.T) {
	store := &MockStorage{
		getByLabelErr: fmt.Errorf("storage error"),
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics/account?key=account_id&value=0.0.5000", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestHandleMetricsByLabel_MethodNotAllowed tests endpoint with wrong method
func TestHandleMetricsByLabel_MethodNotAllowed(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("DELETE", "/api/v1/metrics/account", nil)
	w := httptest.NewRecorder()

	server.handleMetricsByLabel(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestHandleStorageStats_Success tests getting storage stats
func TestHandleStorageStats_Success(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{Name: "m1", Value: 1, Timestamp: 1, Labels: map[string]string{}},
			{Name: "m2", Value: 2, Timestamp: 2, Labels: map[string]string{}},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/storage/stats", nil)
	w := httptest.NewRecorder()

	server.handleStorageStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response StatsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.MetricCount != 2 {
		t.Errorf("expected metric_count 2, got %d", response.MetricCount)
	}

	if response.MaxSize != 10000 {
		t.Errorf("expected max_size 10000, got %d", response.MaxSize)
	}

	if response.Utilization == "" {
		t.Error("expected utilization to be set")
	}
}

// TestHandleStorageStats_EmptyStorage tests stats with empty storage
func TestHandleStorageStats_EmptyStorage(t *testing.T) {
	store := &MockStorage{metrics: []types.Metric{}}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/storage/stats", nil)
	w := httptest.NewRecorder()

	server.handleStorageStats(w, req)

	var response StatsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.MetricCount != 0 {
		t.Errorf("expected metric_count 0, got %d", response.MetricCount)
	}
}

// TestHandleStorageStats_StorageError tests stats when storage fails
func TestHandleStorageStats_StorageError(t *testing.T) {
	store := &MockStorage{
		statsErr: fmt.Errorf("stats error"),
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/storage/stats", nil)
	w := httptest.NewRecorder()

	server.handleStorageStats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestHandleStorageStats_NotSupported tests stats when storage doesn't support stats
func TestHandleStorageStats_NotSupported(t *testing.T) {
	// Create a storage that doesn't implement Stats()
	store := &simpleStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/storage/stats", nil)
	w := httptest.NewRecorder()

	server.handleStorageStats(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected status 501, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}
}

// TestHandleStorageStats_MethodNotAllowed tests stats endpoint with wrong method
func TestHandleStorageStats_MethodNotAllowed(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("POST", "/api/v1/storage/stats", nil)
	w := httptest.NewRecorder()

	server.handleStorageStats(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// TestWriteJSON tests the writeJSON helper
func TestWriteJSON(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	w := httptest.NewRecorder()
	response := HealthResponse{Status: "test", Version: "1.0"}

	server.writeJSON(w, http.StatusOK, response)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var decoded HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if decoded.Status != "test" {
		t.Errorf("expected status 'test', got '%s'", decoded.Status)
	}
}

// TestWriteError tests the writeError helper
func TestWriteError(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	w := httptest.NewRecorder()
	server.writeError(w, http.StatusInternalServerError, "test error")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error != "test error" {
		t.Errorf("expected error 'test error', got '%s'", response.Error)
	}
}

// TestHandleMetrics_NegativeLimit tests metrics with negative limit
func TestHandleMetrics_NegativeLimit(t *testing.T) {
	store := &MockStorage{
		metrics: []types.Metric{
			{Name: "m1", Value: 1, Timestamp: 1, Labels: map[string]string{}},
		},
	}
	server := NewServer(8080, store, &MockAlertManager{})

	req := httptest.NewRequest("GET", "/api/v1/metrics?limit=-10", nil)
	w := httptest.NewRecorder()

	server.handleMetrics(w, req)

	var response MetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Negative limit should use default
	if response.Error != "" {
		t.Errorf("expected no error for negative limit, got: %s", response.Error)
	}
}

// simpleStorage is a minimal storage implementation without Stats() for testing
type simpleStorage struct{}

func (s *simpleStorage) StoreMetric(metric types.Metric) error {
	return nil
}

func (s *simpleStorage) GetMetrics(name string, limit int) ([]types.Metric, error) {
	return []types.Metric{}, nil
}

func (s *simpleStorage) GetMetricsByLabel(key, value string) ([]types.Metric, error) {
	return []types.Metric{}, nil
}

func (s *simpleStorage) DeleteOldMetrics(beforeTimestamp int64) error {
	return nil
}

func (s *simpleStorage) Close() error {
	return nil
}

// BenchmarkHandleMetrics benchmarks metrics retrieval
func BenchmarkHandleMetrics(b *testing.B) {
	// Create storage with 1000 metrics
	metrics := make([]types.Metric, 1000)
	for i := 0; i < 1000; i++ {
		metrics[i] = types.Metric{
			Name:      "account_balance",
			Value:     float64(i),
			Timestamp: int64(i),
			Labels: map[string]string{
				"account_id": fmt.Sprintf("0.0.%d", i),
			},
		}
	}

	store := &MockStorage{metrics: metrics}
	server := NewServer(8080, store, &MockAlertManager{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/metrics?limit=100", nil)
		w := httptest.NewRecorder()
		server.handleMetrics(w, req)
	}
}

// BenchmarkHandleMetricsByLabel benchmarks label-based metric retrieval
func BenchmarkHandleMetricsByLabel(b *testing.B) {
	// Create storage with 1000 metrics
	metrics := make([]types.Metric, 1000)
	for i := 0; i < 1000; i++ {
		accountID := "0.0.5000"
		if i > 500 {
			accountID = "0.0.5001"
		}

		metrics[i] = types.Metric{
			Name:      "account_balance",
			Value:     float64(i),
			Timestamp: int64(i),
			Labels: map[string]string{
				"account_id": accountID,
			},
		}
	}

	store := &MockStorage{metrics: metrics}
	server := NewServer(8080, store, &MockAlertManager{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/metrics/account?key=account_id&value=0.0.5000", nil)
		w := httptest.NewRecorder()
		server.handleMetricsByLabel(w, req)
	}
}

// TestResponseContentType ensures all responses have correct content type
func TestResponseContentType(t *testing.T) {
	store := &MockStorage{}
	server := NewServer(8080, store, &MockAlertManager{})

	tests := []struct {
		name    string
		path    string
		method  string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{
			name:    "health",
			path:    "/health",
			method:  "GET",
			handler: server.handleHealth,
		},
		{
			name:    "metrics",
			path:    "/api/v1/metrics",
			method:  "GET",
			handler: server.handleMetrics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}
		})
	}
}

// ============================================================================
// Alert Endpoint Tests
// ============================================================================

// TestHandleListAlerts_Success tests listing all alert rules
func TestHandleListAlerts_Success(t *testing.T) {
	// Create MockAlertManager with some test rules
	alertMgr := &MockAlertManager{
		rules: []alerting.AlertRule{
			{
				ID:              "rule-1",
				Name:            "Low Balance Alert",
				Description:     "Alert when account balance is too low",
				MetricName:      "account_balance",
				Condition:       "<",
				Threshold:       1000000,
				Enabled:         true,
				Severity:        "warning",
				CooldownSeconds: 300,
			},
			{
				ID:              "rule-2",
				Name:            "High Transaction Count",
				Description:     "Alert when transactions exceed threshold",
				MetricName:      "transaction_count",
				Condition:       ">",
				Threshold:       1000,
				Enabled:         true,
				Severity:        "info",
				CooldownSeconds: 600,
			},
			{
				ID:              "rule-3",
				Name:            "Network Offline",
				Description:     "Alert when network status is down",
				MetricName:      "network_status",
				Condition:       "==",
				Threshold:       0,
				Enabled:         true,
				Severity:        "critical",
				CooldownSeconds: 60,
			},
		},
	}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("GET", "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Decode response
	var response AlertListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify count matches number of rules
	if response.Count != 3 {
		t.Errorf("expected count 3, got %d", response.Count)
	}

	// Verify all rules are present
	if len(response.Alerts) != 3 {
		t.Errorf("expected 3 alerts in array, got %d", len(response.Alerts))
	}

	// Verify specific rule fields are populated correctly
	if response.Alerts[0].Name != "Low Balance Alert" {
		t.Errorf("expected rule name 'Low Balance Alert', got '%s'", response.Alerts[0].Name)
	}
	if response.Alerts[0].Condition != "<" {
		t.Errorf("expected condition '<', got '%s'", response.Alerts[0].Condition)
	}
	if response.Alerts[0].Severity != "warning" {
		t.Errorf("expected severity 'warning', got '%s'", response.Alerts[0].Severity)
	}
	if response.Alerts[2].Severity != "critical" {
		t.Errorf("expected third rule severity 'critical', got '%s'", response.Alerts[2].Severity)
	}
}

// TestHandleListAlerts_Empty tests listing when no rules exist
func TestHandleListAlerts_Empty(t *testing.T) {
	// Create MockAlertManager with no rules
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Make GET request to /api/v1/alerts
	req := httptest.NewRequest("GET", "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Decode response
	var response AlertListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify count is 0
	if response.Count != 0 {
		t.Errorf("expected count 0, got %d", response.Count)
	}

	// Verify Alerts array is empty
	if len(response.Alerts) != 0 {
		t.Errorf("expected empty alert array, got %d", len(response.Alerts))
	}
}

// TestHandleListAlerts_MethodNotAllowed tests with wrong HTTP method
func TestHandleListAlerts_MethodNotAllowed(t *testing.T) {
	// Create MockAlertManager
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Make PUT request to /api/v1/alerts
	req := httptest.NewRequest("PUT", "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 405 (Method Not Allowed)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}

	// Decode response
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	// Verify error message in response
	if response.Error != "method not allowed" {
		t.Fatalf("expected error 'method not allowed', got '%s'", response.Error)
	}
}

// TestHandleCreateAlert_Success tests creating a new alert rule
func TestHandleCreateAlert_Success(t *testing.T) {
	// Create MockAlertManager
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Create valid CreateAlertRequest payload
	request := &CreateAlertRequest{
		Name:            "Low Balance Alert",
		Description:     "Alert when account balance is too low",
		MetricName:      "account_balance",
		Condition:       "<",
		Threshold:       1000000,
		Severity:        "warning",
		CooldownSeconds: 300,
	}
	jsonBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Make POST request to /api/v1/alerts with JSON body
	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 201 (Created)
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	// Decode response
	var response AlertRuleResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify response contains created rule with matching fields
	if response.Name != request.Name {
		t.Errorf("expected name '%s', got '%s'", request.Name, response.Name)
	}
	if response.Description != request.Description {
		t.Errorf("expected description '%s', got '%s'", request.Description, response.Description)
	}
	if response.MetricName != request.MetricName {
		t.Errorf("expected metric name '%s', got '%s'", request.MetricName, response.MetricName)
	}
	if response.Condition != request.Condition {
		t.Errorf("expected condition '%s', got '%s'", request.Condition, response.Condition)
	}
	if response.Threshold != request.Threshold {
		t.Errorf("expected threshold '%f', got '%f'", request.Threshold, response.Threshold)
	}
	if response.Severity != request.Severity {
		t.Errorf("expected severity '%s', got '%s'", request.Severity, response.Severity)
	}
	if response.CooldownSeconds != request.CooldownSeconds {
		t.Errorf("expected cooldown '%d', got '%d'", request.CooldownSeconds, response.CooldownSeconds)
	}

	// Verify ID was generated
	if response.ID == "" {
		t.Error("expected non-empty id")
	}
	t.Logf("response ID: '%s'", response.ID)

	// Verify AddRule was called
	if alertMgr.addRuleCalls != 1 {
		t.Errorf("expected AddRule to be called once, got %d calls", alertMgr.addRuleCalls)
	}

	// Verify the rule that was added has correct fields
	if alertMgr.lastAddedRule.Name != "Low Balance Alert" {
		t.Errorf("expected rule name 'Low Balance Alert', got '%s'", alertMgr.lastAddedRule.Name)
	}
}

// TestHandleCreateAlert_MissingName tests creating alert with missing name
func TestHandleCreateAlert_MissingName(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	request := &CreateAlertRequest{
		Name:            "",
		MetricName:      "account_balance",
		Condition:       "<",
		Threshold:       1000000,
		Severity:        "warning",
		CooldownSeconds: 300,
	}
	jsonBody, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.addRuleCalls != 0 {
		t.Errorf("expected AddRule not to be called, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleCreateAlert_InvalidCondition tests creating alert with invalid condition
func TestHandleCreateAlert_InvalidCondition(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	request := &CreateAlertRequest{
		Name:            "Test Alert",
		MetricName:      "account_balance",
		Condition:       "foo",
		Threshold:       1000000,
		Severity:        "warning",
		CooldownSeconds: 300,
	}
	jsonBody, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.addRuleCalls != 0 {
		t.Errorf("expected AddRule not to be called, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleCreateAlert_InvalidSeverity tests creating alert with invalid severity
func TestHandleCreateAlert_InvalidSeverity(t *testing.T) {
	// Create MockAlertManager
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Create CreateAlertRequest with invalid severity
	request := CreateAlertRequest{
		Name:            "Test Alert",
		MetricName:      "account_balance",
		Condition:       "<",
		Threshold:       1000000,
		Severity:        "FAKE NEWS",
		CooldownSeconds: 300,
	}
	jsonBody, _ := json.Marshal(request)

	// Make POST request to /api/v1/alerts
	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 400 (Bad Request)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// Verify error message mentions severity
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}
	if !strings.Contains(response.Error, "severity") {
		t.Errorf("expected error to mention 'severity', got: %s", response.Error)
	}

	// Verify AddRule was NOT called
	if alertMgr.addRuleCalls != 0 {
		t.Errorf("expected AddRule not to be called, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleCreateAlert_InvalidJSON tests creating alert with malformed JSON
func TestHandleCreateAlert_InvalidJSON(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.addRuleCalls != 0 {
		t.Errorf("expected AddRule not to be called, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleCreateAlert_ManagerError tests when AddRule fails
func TestHandleCreateAlert_ManagerError(t *testing.T) {
	// Create MockAlertManager with addRuleErr set to an error
	alertMgr := &MockAlertManager{
		addRuleErr: fmt.Errorf("database error"),
	}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Create valid CreateAlertRequest
	req := CreateAlertRequest{
		Name:            "Test Alert",
		Description:     "Test Description",
		MetricName:      "account_balance",
		Condition:       "<",
		Threshold:       1000000,
		Severity:        "warning",
		CooldownSeconds: 300,
	}
	jsonBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Make POST request to /api/v1/alerts
	post := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, post)

	// Verify status 500 (Internal Server Error)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	// Verify error message in response
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.addRuleCalls != 1 {
		t.Errorf("expected AddRule to be called once, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleCreateAlert_NegativeCooldown tests creating alert with negative cooldown
func TestHandleCreateAlert_NegativeCooldown(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	request := &CreateAlertRequest{
		Name:            "Test Alert",
		MetricName:      "account_balance",
		Condition:       "<",
		Threshold:       1000000,
		Severity:        "warning",
		CooldownSeconds: -10,
	}
	jsonBody, _ := json.Marshal(request)

	req := httptest.NewRequest("POST", "/api/v1/alerts", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.addRuleCalls != 0 {
		t.Errorf("expected AddRule not to be called, but it was called %d times", alertMgr.addRuleCalls)
	}
}

// TestHandleDeleteAlert_Success tests deleting an alert rule
func TestHandleDeleteAlert_Success(t *testing.T) {
	testRuleID := "rule-123"
	alertMgr := &MockAlertManager{
		rules: []alerting.AlertRule{
			{
				ID:   testRuleID,
				Name: "Test Rule",
			},
		},
	}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("DELETE", "/api/v1/alerts?id="+testRuleID, nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	if w.Body.Len() != 0 {
		t.Errorf("expected empty response body, got %d bytes", w.Body.Len())
	}

	if alertMgr.removeRuleCalls != 1 {
		t.Errorf("expected RemoveRule to be called once, got %d calls", alertMgr.removeRuleCalls)
	}

	if alertMgr.lastRemovedID != testRuleID {
		t.Errorf("expected RemoveRule to be called with id '%s', got '%s'", testRuleID, alertMgr.lastRemovedID)
	}
}

// TestHandleDeleteAlert_NotFound tests deleting non-existent rule
func TestHandleDeleteAlert_NotFound(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("DELETE", "/api/v1/alerts?id=nonexistent", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.removeRuleCalls != 1 {
		t.Errorf("expected RemoveRule to be called once, got %d calls", alertMgr.removeRuleCalls)
	}
}

// TestHandleDeleteAlert_MissingID tests deleting without id parameter
func TestHandleDeleteAlert_MissingID(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("DELETE", "/api/v1/alerts", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.removeRuleCalls != 0 {
		t.Errorf("expected RemoveRule not to be called, but it was called %d times", alertMgr.removeRuleCalls)
	}
}

// TestHandleDeleteAlert_EmptyID tests deleting with empty id parameter
func TestHandleDeleteAlert_EmptyID(t *testing.T) {
	alertMgr := &MockAlertManager{}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	req := httptest.NewRequest("DELETE", "/api/v1/alerts?id=", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.removeRuleCalls != 0 {
		t.Errorf("expected RemoveRule not to be called, but it was called %d times", alertMgr.removeRuleCalls)
	}
}

// TestHandleDeleteAlert_ManagerError tests when RemoveRule fails (other than not found)
func TestHandleDeleteAlert_ManagerError(t *testing.T) {
	// Create MockAlertManager with removeRuleErr set to an error (not ErrRuleNotFound)
	alertMgr := &MockAlertManager{
		removeRuleErr: alerting.ErrInvalidRule,
	}
	store := &MockStorage{}
	server := NewServer(8080, store, alertMgr)

	// Make DELETE request to /api/v1/alerts?id=test_rule
	req := httptest.NewRequest("DELETE", "/api/v1/alerts?id=test_rule", nil)
	w := httptest.NewRecorder()
	server.handleAlerts(w, req)

	// Verify status 404 (Not Found)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}

	// Verify error message in response
	var response ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Error == "" {
		t.Error("expected error message in response")
	}

	if alertMgr.removeRuleCalls != 1 {
		t.Errorf("expected RemoveRule to be called once, but it was called %d times", alertMgr.removeRuleCalls)
	}
}
