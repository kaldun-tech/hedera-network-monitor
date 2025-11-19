package alerting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSendWebhookRequestSuccess tests successful webhook delivery
func TestSendWebhookRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse payload
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Failed to decode payload: %v", err)
		}

		// Verify payload data
		if payload.RuleID != "test_rule" {
			t.Errorf("Expected RuleID test_rule, got %s", payload.RuleID)
		}

		// Return success
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status": "ok"}`))
		if err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := WebhookConfig{
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	err := SendWebhookRequest(server.URL, payload, config)
	if err != nil {
		t.Fatalf("SendWebhookRequest failed: %v", err)
	}
}

// TestSendWebhookRequestBadURL tests error handling for invalid URL
func TestSendWebhookRequestBadURL(t *testing.T) {
	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := WebhookConfig{
		Timeout:        5 * time.Second,
		MaxRetries:     0,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	err := SendWebhookRequest("http://invalid-domain-that-does-not-exist.com", payload, config)
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

// TestSendWebhookRequestTimeout tests timeout handling
func TestSendWebhookRequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := WebhookConfig{
		Timeout:        100 * time.Millisecond, // Very short timeout
		MaxRetries:     0,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	err := SendWebhookRequest(server.URL, payload, config)
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

// TestSendWebhookRequestNonSuccessStatus tests retry on non-2xx status
func TestSendWebhookRequestNonSuccessStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts fail with 500
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Server error"))
		} else {
			// Third attempt succeeds
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := WebhookConfig{
		Timeout:        5 * time.Second,
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	err := SendWebhookRequest(server.URL, payload, config)
	if err != nil {
		t.Fatalf("SendWebhookRequest failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestSendWebhookRequestExhaustedRetries tests error after exhausting retries
func TestSendWebhookRequestExhaustedRetries(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		// Always return error
		w.WriteHeader(http.StatusServiceUnavailable)
		_, err := w.Write([]byte("Service unavailable"))
		if err != nil {
			t.Logf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := WebhookConfig{
		Timeout:        5 * time.Second,
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	err := SendWebhookRequest(server.URL, payload, config)
	if err == nil {
		t.Fatal("Expected error after exhausting retries")
	}

	// Should have attempted 3 times (initial + 2 retries)
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestSendWebhookRequestSuccess201 tests success with 201 status code
func TestSendWebhookRequestSuccess201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // 201
	}))
	defer server.Close()

	payload := WebhookPayload{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "warning",
		Message:   "Test alert",
		Value:     42.5,
		Timestamp: 1234567890,
	}

	config := DefaultWebhookConfig()

	err := SendWebhookRequest(server.URL, payload, config)
	if err != nil {
		t.Fatalf("SendWebhookRequest should succeed with 201 status: %v", err)
	}
}

// TestCalculateBackoff tests exponential backoff calculation
func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		name     string
		attempt  int
		initial  time.Duration
		max      time.Duration
		expected time.Duration
	}{
		{
			name:     "First attempt",
			attempt:  0,
			initial:  1 * time.Second,
			max:      30 * time.Second,
			expected: 1 * time.Second,
		},
		{
			name:     "Second attempt",
			attempt:  1,
			initial:  1 * time.Second,
			max:      30 * time.Second,
			expected: 2 * time.Second,
		},
		{
			name:     "Third attempt",
			attempt:  2,
			initial:  1 * time.Second,
			max:      30 * time.Second,
			expected: 4 * time.Second,
		},
		{
			name:     "Capped at max",
			attempt:  10,
			initial:  1 * time.Second,
			max:      30 * time.Second,
			expected: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.attempt, tt.initial, tt.max)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestDefaultWebhookConfig tests default configuration values
func TestDefaultWebhookConfig(t *testing.T) {
	config := DefaultWebhookConfig()

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", config.Timeout)
	}

	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", config.MaxRetries)
	}

	if config.InitialBackoff != 1*time.Second {
		t.Errorf("Expected InitialBackoff 1s, got %v", config.InitialBackoff)
	}

	if config.MaxBackoff != 32*time.Second {
		t.Errorf("Expected MaxBackoff 32s, got %v", config.MaxBackoff)
	}
}
