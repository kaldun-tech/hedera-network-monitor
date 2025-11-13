package alerting

import (
	"encoding/json"
	"testing"
	"time"
)

// TestWebhookPayloadCreation tests creating a webhook payload
func TestWebhookPayloadCreation(t *testing.T) {
	payload := WebhookPayload{
		RuleID:    "rule_123",
		RuleName:  "High CPU Usage",
		Severity:  "critical",
		Message:   "CPU usage exceeded 90%",
		Value:     95.5,
		Timestamp: time.Now().Unix(),
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

	expectedKeys := []string{"rule_id", "rule_name", "severity", "message", "value", "timestamp"}
	for _, key := range expectedKeys {
		if _, exists := jsonMap[key]; !exists {
			t.Errorf("Expected key %s not found in JSON", key)
		}
	}
}
