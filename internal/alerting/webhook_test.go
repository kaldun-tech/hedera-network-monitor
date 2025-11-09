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
	var unmarshaled WebhookPayload
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal payload: %v", err)
	}

	// Verify all fields match
	if unmarshaled.RuleID != payload.RuleID {
		t.Errorf("RuleID mismatch: %s != %s", unmarshaled.RuleID, payload.RuleID)
	}

	if unmarshaled.RuleName != payload.RuleName {
		t.Errorf("RuleName mismatch: %s != %s", unmarshaled.RuleName, payload.RuleName)
	}

	if unmarshaled.Severity != payload.Severity {
		t.Errorf("Severity mismatch: %s != %s", unmarshaled.Severity, payload.Severity)
	}

	if unmarshaled.Message != payload.Message {
		t.Errorf("Message mismatch: %s != %s", unmarshaled.Message, payload.Message)
	}

	if unmarshaled.Value != payload.Value {
		t.Errorf("Value mismatch: %f != %f", unmarshaled.Value, payload.Value)
	}

	if unmarshaled.Timestamp != payload.Timestamp {
		t.Errorf("Timestamp mismatch: %d != %d", unmarshaled.Timestamp, payload.Timestamp)
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
