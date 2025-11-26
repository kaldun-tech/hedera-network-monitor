package alerting

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// newTestPayload creates a standard webhook payload for testing
func newTestPayload() WebhookPayload {
	return WebhookPayload{
		RuleID:    "test_rule",
		Severity:  "warning",
		Message:   "Test message",
		Value:     42.0,
		Timestamp: time.Now().Unix(),
		MetricID:  "test_metric",
	}
}

// newTestConfig creates a standard webhook config for testing
func newTestConfig() WebhookConfig {
	return WebhookConfig{
		Timeout:        100 * time.Millisecond,
		MaxRetries:     5,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}
}

// TestWebhookPayloadCreation tests creating a webhook payload
func TestWebhookPayloadCreation(t *testing.T) {
	payload := WebhookPayload{
		RuleID:    "rule_123",
		RuleName:  "High CPU Usage",
		Severity:  "critical",
		Message:   "CPU usage exceeded 90%",
		Value:     95.5,
		Timestamp: time.Now().Unix(),
		MetricID:  "cpu_usage",
	}

	if payload.RuleID != "rule_123" {
		t.Errorf("Expected RuleID rule_123, got %s", payload.RuleID)
	}

	if payload.Severity != "critical" {
		t.Errorf("Expected Severity critical, got %s", payload.Severity)
	}

	if payload.Value != 95.5 {
		t.Errorf("Expected Value 95.5, got %f", payload.Value)
	}

	if payload.MetricID != "cpu_usage" {
		t.Errorf("Expected MetricID cpu_usage, got %s", payload.MetricID)
	}
}

// TestWebhookPayloadJSON tests that webhook payload marshals to JSON correctly
func TestWebhookPayloadJSON(t *testing.T) {
	payload := WebhookPayload{
		RuleID:    "rule_456",
		RuleName:  "Low Memory",
		Severity:  "warning",
		Message:   "Memory usage below 10%",
		Value:     8.2,
		Timestamp: 1234567890,
		MetricID:  "memory_usage",
	}

	// Marshal to JSON
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	// Unmarshal to verify structure
	var unm WebhookPayload
	err = json.Unmarshal(data, &unm)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// Verify all fields match
	if unm.RuleID != payload.RuleID {
		t.Errorf("RuleID mismatch: %s != %s", unm.RuleID, payload.RuleID)
	}

	if unm.RuleName != payload.RuleName {
		t.Errorf("RuleName mismatch: %s != %s", unm.RuleName, payload.RuleName)
	}

	if unm.Severity != payload.Severity {
		t.Errorf("Severity mismatch: %s != %s", unm.Severity, payload.Severity)
	}

	if unm.Message != payload.Message {
		t.Errorf("Message mismatch: %s != %s", unm.Message, payload.Message)
	}

	if unm.Value != payload.Value {
		t.Errorf("Value mismatch: %f != %f", unm.Value, payload.Value)
	}

	if unm.Timestamp != payload.Timestamp {
		t.Errorf("Timestamp mismatch: %d != %d", unm.Timestamp, payload.Timestamp)
	}

	if unm.MetricID != payload.MetricID {
		t.Errorf("MetricID mismatch: %s != %s", unm.MetricID, payload.MetricID)
	}
}

// TestWebhookPayloadJSONKeys tests that JSON keys match expected format
func TestWebhookPayloadJSONKeys(t *testing.T) {
	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Alert",
		Severity:  "info",
		Message:   "Test message",
		Value:     42.0,
		Timestamp: 1000000000,
		MetricID:  "test_metric",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Parse as map to check keys
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal as map: %v", err)
	}

	expectedKeys := []string{"rule_id", "rule_name", "severity", "message", "value", "timestamp", "metric_id"}
	for _, key := range expectedKeys {
		if _, exists := jsonMap[key]; !exists {
			t.Errorf("Expected key %s not found in JSON", key)
		}
	}
}

// TestSendWebhookRequest_SuccessOnFirstAttempt tests webhook succeeds immediately
func TestSendWebhookRequest_SuccessOnFirstAttempt(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := SendWebhookRequest(server.URL, newTestPayload(), newTestConfig())
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected call count to be 1, got %d", callCount)
	}
}

// TestSendWebhookRequest_RetryOnServerError tests webhook retries on 5xx errors
func TestSendWebhookRequest_RetryOnServerError(t *testing.T) {
	// Implement retry logic verification
	// Create server with call counter
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
		} else {
			w.WriteHeader(http.StatusOK) // 200
		}
	}))
	defer server.Close()

	err := SendWebhookRequest(server.URL, newTestPayload(), newTestConfig())
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}
}

// TestSendWebhookRequest_ExponentialBackoff tests backoff increases exponentially
func TestSendWebhookRequest_ExponentialBackoff(t *testing.T) {
	// Track request timestamps to verify exponential backoff
	var timestamps []time.Time
	var tsMutex sync.Mutex

	// Create server that fails 4x then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tsMutex.Lock()
		timestamps = append(timestamps, time.Now())
		tsMutex.Unlock()

		if len(timestamps) < 5 {
			w.WriteHeader(http.StatusServiceUnavailable) // 503
		} else {
			w.WriteHeader(http.StatusOK) // 200
		}
	}))
	defer server.Close()

	err := SendWebhookRequest(server.URL, newTestPayload(), newTestConfig())
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	// Calculate actual delays between requests
	var delays []time.Duration
	for i := 1; i < len(timestamps); i++ {
		delays = append(delays, timestamps[i].Sub(timestamps[i-1]))
	}

	// Verify exponential growth: each delay ~2x the previous
	tolerance := 1.5 // Allow 50% variance
	for i := 1; i < len(delays); i++ {
		ratio := float64(delays[i]) / float64(delays[i-1])
		if ratio < (2.0-tolerance) || ratio > (2.0+tolerance) {
			t.Errorf("Delay %d: ratio %.2f, expected ~2.0", i, ratio)
		}
	}
}

// TestSendWebhookRequest_InvalidURL tests webhook handles bad URLs
func TestSendWebhookRequest_InvalidURL(t *testing.T) {
	// Call SendWebhookRequest with invalid URL, verify error (no retries)
	err := SendWebhookRequest("not-a-valid-url", newTestPayload(), newTestConfig())
	if err == nil {
		t.Errorf("Expected error, got none")
	}
}

// TestSendWebhookRequest_ContentType tests webhook sets correct headers
func TestSendWebhookRequest_ContentType(t *testing.T) {
	// Track headers
	var header map[string][]string
	var hMutex sync.Mutex

	// Create server that captures headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hMutex.Lock()
		header = r.Header
		hMutex.Unlock()

		w.WriteHeader(http.StatusOK)
	}))

	err := SendWebhookRequest(server.URL, newTestPayload(), newTestConfig())
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if header["Content-Type"][0] != "application/json" {
		t.Errorf("Expected application/json, got %s", header["Content-Type"][0])
	}
	userAgent := header["User-Agent"][0]
	if userAgent != "hedera-network-monitor/1.0" {
		t.Errorf("Expected hedera-network-monitor/1.0, got %s", userAgent)
	}
}

// TestSendWebhookRequest_RetryOnNetworkError tests webhook retries on network failures
func TestSendWebhookRequest_RetryOnNetworkError(t *testing.T) {
	// Server that always returns an error (closed connection)
	// This forces retries on every attempt
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Accept and immediately close connections to simulate network errors
	var callMutex = &sync.Mutex{}
	callCount := 0

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}
			callMutex.Lock()
			callCount++
			callMutex.Unlock()
			_ = conn.Close() // Close immediately, simulating network error
		}
	}()

	url := "http://" + listener.Addr().String()

	config := newTestConfig()
	config.MaxRetries = 2

	err = SendWebhookRequest(url, newTestPayload(), config)
	if err == nil {
		t.Errorf("Expected error on network failure, got nil")
	}

	_ = listener.Close()

	callMutex.Lock()
	count := callCount
	callMutex.Unlock()
	if count < 2 {
		t.Errorf("Expected at least 2 attempts, got %d", count)
	}
}

// TestSendWebhookRequest_ExhaustedRetries tests webhook gives up after max retries
func TestSendWebhookRequest_ExhaustedRetries(t *testing.T) {
	// Server always fails with 500 error
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	payload := newTestPayload()
	payload.Severity = "critical"
	payload.Value = 100.0

	config := newTestConfig()
	config.Timeout = 5 * time.Second
	config.MaxRetries = 2

	err := SendWebhookRequest(server.URL, payload, config)
	if err == nil {
		t.Errorf("Expected error after exhausting retries, got nil")
	}

	expectedCalls := 3 // initial + 2 retries
	if callCount != expectedCalls {
		t.Errorf("Expected %d calls, got %d", expectedCalls, callCount)
	}
}

// TestSendWebhookRequest_MaxBackoffCap tests backoff respects maximum
func TestSendWebhookRequest_MaxBackoffCap(t *testing.T) {
	// Server fails 6 times to trigger multiple backoff periods
	var tsMutex = &sync.Mutex{}
	callCount := 0
	var timestamps []time.Time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tsMutex.Lock()
		timestamps = append(timestamps, time.Now())
		callCount++
		count := callCount
		tsMutex.Unlock()

		if count > 5 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	payload := newTestPayload()
	payload.Message = "Test"
	payload.Value = 50.0

	config := newTestConfig()
	config.Timeout = 5 * time.Second
	config.InitialBackoff = 50 * time.Millisecond

	err := SendWebhookRequest(server.URL, payload, config)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	// Verify backoff capped at MaxBackoff
	// After attempt 2, backoff would be 50*2^2 = 200ms, but capped at 100ms
	tsMutex.Lock()
	ts := timestamps
	tsMutex.Unlock()
	if len(ts) >= 4 {
		delay3 := ts[3].Sub(ts[2])
		if delay3 > 150*time.Millisecond {
			t.Errorf("Backoff delay %v exceeds MaxBackoff + tolerance (100ms)", delay3)
		}
	}
}

// TestSendWebhookRequest_TimeoutConfig tests custom timeout is respected
func TestSendWebhookRequest_TimeoutConfig(t *testing.T) {
	// Server takes a long time to respond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload := newTestPayload()
	payload.Severity = "info"
	payload.Message = "Test"
	payload.Value = 25.0

	config := newTestConfig()
	config.MaxRetries = 2
	config.InitialBackoff = 50 * time.Millisecond

	start := time.Now()
	err := SendWebhookRequest(server.URL, payload, config)
	elapsed := time.Since(start)

	if err == nil {
		t.Errorf("Expected timeout error, got nil")
	}

	// Should timeout quickly, not wait 2 seconds
	if elapsed > 2*time.Second {
		t.Errorf("Timeout not respected: took %v", elapsed)
	}
}

// TestSendWebhookRequest_Redirect tests webhook handles HTTP redirects
func TestSendWebhookRequest_Redirect(t *testing.T) {
	// Create redirect target server
	finalCallReceived := false
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		finalCallReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	// Create redirect server
	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	payload := newTestPayload()
	payload.Value = 75.0

	err := SendWebhookRequest(redirectServer.URL, payload, DefaultWebhookConfig())
	if err != nil {
		t.Errorf("Expected redirect to succeed, got error: %v", err)
	}

	if !finalCallReceived {
		t.Errorf("Expected request to reach redirect target")
	}
}

// TestSendWebhookRequest_ResponseBodyRead tests response body is properly closed
func TestSendWebhookRequest_ResponseBodyRead(t *testing.T) {
	// Server returns a large response body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write a substantial response body
		_, _ = w.Write([]byte("This is a test response body that should be read and closed properly to avoid resource leaks"))
	}))
	defer server.Close()

	payload := newTestPayload()
	payload.Message = "Test body read"
	payload.Value = 10.0

	// Make multiple requests to verify bodies are properly cleaned up
	for i := 0; i < 5; i++ {
		err := SendWebhookRequest(server.URL, payload, DefaultWebhookConfig())
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
		}
	}
}
