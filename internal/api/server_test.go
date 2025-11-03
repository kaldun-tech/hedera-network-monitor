package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// MockStorage is a mock implementation of the Storage interface for testing
type MockStorage struct {
	metrics         []types.Metric
	getMetricsErr   error
	getByLabelErr   error
	storeMetricErr  error
	deleteOldErr    error
	closeErr        error
	statsErr        error
	supportsStats   bool
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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

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
	server := NewServer(8080, store)

	tests := []struct {
		name   string
		path   string
		method string
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
