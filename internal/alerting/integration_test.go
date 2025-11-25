package alerting

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
)

// TestEndToEndAlertDispatchSingleRule tests the complete flow:
// metric collection -> rule evaluation -> webhook dispatch
func TestEndToEndAlertDispatchSingleRule(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "test_rule",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, _, errChan := startAlertProcessor(manager, 2*time.Second)

	// Send metric via CheckMetric() that triggers the rule alert
	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
	}
	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Wait for webhook to be called
	time.Sleep(100 * time.Millisecond)

	// Wait for Run() to return
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}

	// Verify webhook was called
	if len(*webhookCalls) != 1 {
		t.Fatalf("Expected 1 webhook call, got %d", len(*webhookCalls))
	}

	// Verify payload content
	call := (*webhookCalls)[0]
	if call.RuleID != "test_rule" {
		t.Errorf("Expected rule_id test_rule, got %s", call.RuleID)
	}
	if call.Value != 150.0 {
		t.Errorf("Expected value 150.0, got %f", call.Value)
	}
}

// TestEndToEndAlertDispatchMultipleRules tests alert dispatch with multiple rules
func TestEndToEndAlertDispatchMultipleRules(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	// Add multiple alert rules (e.g., >, <, ==, changed)
	manager := NewManager(cfg)
	// Add >
	err := manager.AddRule(AlertRule{
		ID:         "gt_rule",
		Name:       "High Value",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}
	// Add <
	err = manager.AddRule(AlertRule{
		ID:         "lt_rule",
		Name:       "Low Value",
		MetricName: "test_metric",
		Condition:  "<",
		Threshold:  50,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}
	// Add changed
	err = manager.AddRule(AlertRule{
		ID:         "changed_rule",
		Name:       "Changed",
		MetricName: "test_metric",
		Condition:  "changed",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, _, errChan := startAlertProcessor(manager, 2*time.Second)

	// Send metric values that trigger only greater than > and changed rules
	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
	}
	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Wait for webhook to be called
	time.Sleep(100 * time.Millisecond)

	// Wait for Run() to return
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}

	// Verify webhooks 1 and 3 were called with correct payloads
	if len(*webhookCalls) != 2 {
		t.Fatalf("Expected 2 webhook call, got %d", len(*webhookCalls))
	}

	ruleIDs := map[string]bool{}
	for _, call := range *webhookCalls {
		ruleIDs[call.RuleID] = true
	}

	if !ruleIDs["rule1"] {
		t.Error("Expected rule1 (>) to trigger")
	}
	if !ruleIDs["rule3"] {
		t.Error("Expected rule3 (changed) to trigger")
	}
	if ruleIDs["rule2"] {
		t.Error("Expected rule2 (<) NOT to trigger")
	}
}

// TestEndToEndAlertWithCooldown tests cooldown prevents duplicate webhook calls
func TestEndToEndAlertWithCooldown(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL and a short cooldown
	cfg := config.AlertingConfig{
		CooldownSeconds: 1,
		Webhooks:        []string{server.URL},
	}
	manager := NewManager(cfg)
	// Add alert rule
	err := manager.AddRule(AlertRule{
		ID:         "test_rule",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 5*time.Second)
	defer cancel()

	// Send first metric that triggers alert
	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond, 1, nil, webhookCalls)

	// Send second metric within cooldown period (no additional alert)
	sendMetricAndVerifyWebhooks(t, manager, 200.0, 0, 1, nil, webhookCalls)

	// Wait for cooldown to expire (1.1 seconds to be safe)
	time.Sleep(1100 * time.Millisecond)

	// Send third metric after cooldown expires (triggers alert again)
	sendMetricAndVerifyWebhooks(t, manager, 250.0, 100*time.Millisecond, 2, nil, webhookCalls)

	// Verify both payloads are correct
	if (*webhookCalls)[0].Value != 150.0 {
		t.Errorf("First call: expected value 150.0, got %f", (*webhookCalls)[0].Value)
	}
	if (*webhookCalls)[1].Value != 250.0 {
		t.Errorf("Second call: expected value 250.0, got %f", (*webhookCalls)[1].Value)
	}

	// Clean up: wait for processor to finish
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndWebhookPayloadFormat tests the webhook receives correct JSON structure
func TestEndToEndWebhookPayloadFormat(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:          "format_test_rule",
		Name:        "Format Test Alert",
		Description: "Testing webhook payload format",
		MetricName:  "test_metric",
		Condition:   ">",
		Threshold:   100,
		Enabled:     true,
		Severity:    "critical",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 2*time.Second)
	defer cancel()

	// Capture the time before sending metric
	beforeTime := time.Now().Unix()

	// Send metric that triggers alert
	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond,
		1, nil, webhookCalls)

	// Capture the time after webhook was received
	afterTime := time.Now().Unix()

	// Verify webhook received payload with:
	//    - Correct RuleID: "format_test_rule"
	//    - Correct RuleName: "Format Test Alert"
	//    - Correct Severity: "critical"
	//    - Correct Value: match the metric value
	//    - Valid Timestamp: is set and reasonable
	//    - Correct Message: "Testing webhook payload format"
	payload := (*webhookCalls)[0]
	if payload.RuleID != "format_test_rule" {
		t.Errorf("Expected RuleID format_test_rule, got %s", payload.RuleID)
	}
	if payload.RuleName != "Format Test Alert" {
		t.Errorf("Expected RuleName Format Test Alert, got %s", payload.RuleName)
	}
	if payload.Severity != "critical" {
		t.Errorf("Expected Severity critical, got %s", payload.Severity)
	}
	if payload.Value != 150.0 {
		t.Errorf("Expected value 150.0, got %f", payload.Value)
	}
	if payload.Timestamp < beforeTime {
		t.Errorf("Timestamp %d is before alert was sent (%d)", payload.Timestamp, beforeTime)
	}
	if payload.Timestamp > afterTime {
		t.Errorf("Timestamp %d is in the future (now is %d)", payload.Timestamp, afterTime)
	}
	if payload.Message != "Testing webhook payload format" {
		t.Errorf("Expected Message Testing webhook payload format, got %s", payload.Message)
	}

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndAlertWithLabels tests alerts work with labeled metrics
func TestEndToEndAlertWithLabels(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "labeled_rule",
		Name:       "Labeled Rule Test",
		MetricName: "account_balance",
		Condition:  ">",
		Threshold:  1000000,
		Enabled:    true,
		Severity:   "info",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 2*time.Second)
	defer cancel()

	// Send metric with labels (e.g., account_id: "0.0.5000")
	labels := map[string]string{
		"account_id": "0.0.5000",
	}
	sendMetricAndVerifyWebhooks(t, manager, 1500000.0, 100*time.Millisecond,
		1, labels, webhookCalls)

	// Verify alert.MetricID includes the label (e.g., "account_balance[0.0.5000]")
	payload := (*webhookCalls)[0]

	// Verify webhook receives the MetricID in the payload
	if payload.MetricID != "account_balance[0.0.5000]" {
		t.Errorf("Expected MetricID account_balance[0.0.5000], got %s", payload.MetricID)
	}

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndMultipleWebhooks tests alert is sent to multiple webhook endpoints
func TestEndToEndMultipleWebhooks(t *testing.T) {
	// Create two mock servers that capture webhook calls
	server1, webhookCalls1 := captureWebhookCalls(t)
	defer server1.Close()
	server2, webhookCalls2 := captureWebhookCalls(t)
	defer server2.Close()

	// Create AlertManager with both webhook URLs
	cfg := config.AlertingConfig{
		Webhooks: []string{server1.URL, server2.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "multi_webhook_rule",
		Name:       "Multiple Webhooks Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
		Severity:   "warning",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 2*time.Second)
	defer cancel()

	// Send metric that triggers alert
	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond,
		1, nil, webhookCalls1)
	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond,
		1, nil, webhookCalls2)

	// Verify BOTH webhookCalls1 and webhookCalls2 have the alert
	payload1 := (*webhookCalls1)[0]
	payload2 := (*webhookCalls2)[0]

	// Verify payloads are identical (same RuleID, Value, Timestamp, etc.)
	if payload1.RuleID != "multi_webhook_rule" || payload2.RuleID != "multi_webhook_rule" {
		t.Errorf("Expected RuleID multi_webhook_rule, got %s and %s",
			payload1.RuleID, payload2.RuleID)
	}
	if payload1.RuleName != "Multiple Webhooks Test" || payload2.RuleName != "Multiple Webhooks Test" {
		t.Errorf("Expected RuleName Multiple Webhooks Test, got %s and %s",
			payload1.RuleName, payload2.RuleName)
	}
	if payload1.Severity != "warning" || payload2.Severity != "warning" {
		t.Errorf("Expected Severity warning, got %s and %s",
			payload1.Severity, payload2.Severity)
	}
	if payload1.Value != 150.0 || payload2.Value != 150.0 {
		t.Errorf("Expected Value 150.0, got %f and %f", payload1.Value, payload2.Value)
	}
	if payload1.Timestamp != payload2.Timestamp {
		t.Errorf("Expected identical timestamps, got %d vs %d",
			payload1.Timestamp, payload2.Timestamp)
	}

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndAlertDisabledRuleNoWebhook tests disabled rules don't trigger webhooks
func TestEndToEndAlertDisabledRuleNoWebhook(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "disabled_rule",
		Name:       "Disabled Rule Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    false, // KEY: Rule is disabled
		Severity:   "warning",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 2*time.Second)
	defer cancel()

	// Send metric that would normally trigger the rule (e.g., value > 100)
	// Verify webhook was NOT called (webhookCalls should be empty)
	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond,
		0, nil, webhookCalls)

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndContextCancellation tests graceful shutdown via context
func TestEndToEndContextCancellation(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "cancel_test_rule",
		Name:       "Context Cancellation Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
		Severity:   "info",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 10*time.Second) // long timeout
	_ = webhookCalls                                                   // silence unused warning if not used

	// Send a metric
	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
	}
	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Wait briefly for alert processing to start
	time.Sleep(50 * time.Millisecond)

	// Now cancel the context to trigger shutdown
	cancel()

	// Wait for Run() to return
	err = <-errChan
	// Verify error is context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}

	// Clean up
	cancel()
	err = <-errChan
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
}

// TestEndToEndWebhookFailureHandling tests webhook retry on server errors
func TestEndToEndWebhookFailureHandling(t *testing.T) {
	// Create a server that fails first 2 requests, then succeeds
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount < 3 {
			// First 2 attempts fail
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// 3rd attempt succeeds
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Capture webhook calls with the failing server. Track each attempt
	var requestAttempts []struct {
		AttemptNumber int
		StatusCode    int
		Timestamp     time.Time
	}
	var mu sync.Mutex
	var calls []WebhookPayload

	failingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		calls = append(calls, payload)
		attemptNum := len(calls) // Track which attempt this is
		mu.Unlock()

		// Record this attempt
		mu.Lock()
		var statusCode int
		if attemptNum < 3 {
			statusCode = http.StatusServiceUnavailable // Fail first 2
		} else {
			statusCode = http.StatusOK // Succeed on 3rd
		}

		requestAttempts = append(requestAttempts, struct {
			AttemptNumber int
			StatusCode    int
			Timestamp     time.Time
		}{
			AttemptNumber: attemptNum,
			StatusCode:    statusCode,
			Timestamp:     time.Now(),
		})
		mu.Unlock()

		w.WriteHeader(statusCode)
	}))
	defer failingServer.Close()

	// Create AlertManager with the failing webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{failingServer.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "retry_test_rule",
		Name:       "Retry Test Rule",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100,
		Enabled:    true,
		Severity:   "critical",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor (need longer timeout for retries)
	_, cancel, errChan := startAlertProcessor(manager, 5*time.Second)
	defer cancel()

	// Send metric that triggers alert
	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
	}
	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Wait for retries to complete (exponential backoff: 1s, 2s, then success)
	// Max retries is 5, initial backoff 1s, so max wait >7 seconds for this test
	time.Sleep(10 * time.Second)

	// Verify attempts 1 & 2 failed, attempt 3 succeeded
	mu.Lock()
	defer mu.Unlock()

	if len(requestAttempts) < 3 {
		t.Fatalf("Expected at least 3 attempts, got %d", len(requestAttempts))
	}

	// Check first two attempts failed
	if requestAttempts[0].StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Attempt 1: expected %d, got %d", http.StatusServiceUnavailable, requestAttempts[0].StatusCode)
	}
	if requestAttempts[1].StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Attempt 2: expected %d, got %d", http.StatusServiceUnavailable, requestAttempts[1].StatusCode)
	}
	// Check third attempt succeeded
	if requestAttempts[2].StatusCode != http.StatusOK {
		t.Errorf("Attempt 3: expected %d, got %d", http.StatusOK, requestAttempts[2].StatusCode)
	}

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndAlertQueueOverflow tests behavior when alert queue is full
func TestEndToEndAlertQueueOverflow(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with very small queue buffer
	cfg := config.AlertingConfig{
		Webhooks:        []string{server.URL},
		QueueBufferSize: 1, // Only room for 1 alert in queue
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "overflow_rule",
		Name:       "Overflow Test Rule",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  50,
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 5*time.Second)
	defer cancel()

	// Send many metrics rapidly to trigger overflow
	// With QueueBufferSize=1, only 1 alert can be queued at a time
	// Additional alerts will be dropped
	for i := 0; i < 10; i++ {
		metric := types.Metric{
			Name:  "test_metric",
			Value: float64(60 + i), // All > 50 threshold, triggers rule
		}
		err := manager.CheckMetric(metric)
		if err != nil {
			t.Fatalf("CheckMetric panicked or failed: %v", err)
		}
	}

	// Wait for webhook calls to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify that some webhooks were called (but not all 10, due to queue overflow)
	// We expect at least 1 and at most a few (not 10)
	callCount := len(*webhookCalls)
	if callCount == 0 {
		t.Errorf("Expected at least 1 webhook call, got %d", callCount)
	}
	if callCount >= 10 {
		t.Errorf("Expected some alerts to be dropped due to overflow, but all 10 were processed")
	}

	// Verify webhook payload is correct (for the ones that did get through)
	if callCount > 0 {
		firstCall := (*webhookCalls)[0]
		if firstCall.RuleID != "overflow_rule" {
			t.Errorf("Expected rule_id overflow_rule, got %s", firstCall.RuleID)
		}
	}

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

// TestEndToEndStateTrackingWithWebhook tests state-tracking conditions (changed/increased/decreased)
// work correctly and trigger webhooks appropriately
func TestEndToEndStateTrackingWithWebhook(t *testing.T) {
	// Create mock server that captures webhook calls
	server, webhookCalls := captureWebhookCalls(t)
	defer server.Close()

	// Create AlertManager with mock webhook URL
	cfg := config.AlertingConfig{
		Webhooks: []string{server.URL},
	}
	manager := NewManager(cfg)
	err := manager.AddRule(AlertRule{
		ID:         "state_tracking_rule",
		Name:       "State Tracking Test",
		MetricName: "test_metric",
		Condition:  "changed",
		Enabled:    true,
		Severity:   "info",
	})
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Start alert processor
	_, cancel, errChan := startAlertProcessor(manager, 3*time.Second)
	defer cancel()

	// Send first metric: 100 (initializes state)
	// Wait briefly, verify webhook NOT called (first value has no previous to compare)
	waitTime := 50 * time.Millisecond
	sendMetricAndVerifyWebhooks(t, manager, 100.0, waitTime,
		0, nil, webhookCalls)

	// Send second metric: 150 (different from previous)
	// Wait, verify webhook IS called (1 total call)
	sendMetricAndVerifyWebhooks(t, manager, 150.0, waitTime,
		1, nil, webhookCalls)

	// Send third metric: 150 (same as previous)
	// Wait, verify webhook NOT called (still 1 total call)
	sendMetricAndVerifyWebhooks(t, manager, 150.0, waitTime,
		1, nil, webhookCalls)

	// Send fourth metric: 200 (different again)
	// Wait, verify webhook called again (2 total calls)
	sendMetricAndVerifyWebhooks(t, manager, 200.0, waitTime,
		2, nil, webhookCalls)

	// Clean up
	err = <-errChan
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("Unexpected error from Run(): %v", err)
	}
}

//nolint:unused
func defaultAlertManagerConfig(t *testing.T) config.AlertingConfig {
	return config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{"https://example.com/webhook"},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
}

// Helper: sendMetricAndVerifyWebhooks sends a metric and verifies webhook was called expected number of times
// Reduces duplication when testing multiple metric scenarios
// Example usage:
//
//	sendMetricAndVerifyWebhooks(t, manager, 150.0, 100*time.Millisecond, 1, webhookCalls)
func sendMetricAndVerifyWebhooks(t *testing.T, manager *Manager, value float64, waitTime time.Duration,
	expectedCalls int, labels map[string]string, webhookCalls *[]WebhookPayload) {
	metric := types.Metric{
		Name:   "test_metric",
		Value:  value,
		Labels: labels,
	}
	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	if 0 < waitTime {
		time.Sleep(waitTime)
	}

	if len(*webhookCalls) != expectedCalls {
		t.Fatalf("Expected %d webhook calls, got %d", expectedCalls, len(*webhookCalls))
	}
}

// Helper: captureWebhookCalls wraps an httptest.Server to capture webhook payloads
// Returns the server and a slice to track calls
// Example usage:
//
//	server, webhookCalls := captureWebhookCalls()
//	defer server.Close()
//	manager.webhooks = []string{server.URL}
//	 ...send metrics...
//	 webhookCalls[0] now contains first WebhookPayload
func captureWebhookCalls(t *testing.T) (*httptest.Server, *[]WebhookPayload) {
	var mu sync.Mutex
	var calls []WebhookPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method is POST
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Verify Content-Type is application/json
		contentTypeHeader := r.Header.Get("Content-Type")
		if contentTypeHeader != "application/json" {
			t.Errorf("Expected application/json, got %s", contentTypeHeader)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Decode request body into WebhookPayload
		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			t.Errorf("Error decoding JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// ock mutex, append to calls slice, unlock
		mu.Lock()
		calls = append(calls, payload)
		mu.Unlock()

		// Write 200 OK response
		w.WriteHeader(http.StatusOK)
	}))

	return server, &calls
}

// Helper: waitForWebhookCall waits up to timeout for at least n webhook calls
// Useful for synchronizing tests with async webhook dispatch
// Example usage:
//
//	if !waitForWebhookCall(webhookCalls, 1, 2*time.Second) {
//	    t.Fatal("Expected 1 webhook call within 2 seconds")
//	}
//
//nolint:unused
func waitForWebhookCall(callsPtr *[]WebhookPayload, expectedCount int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check if len(calls) >= expectedCount
		callCount := len(*callsPtr)

		if callCount >= expectedCount {
			return true
		}

		// Sleep 10ms and retry
		time.Sleep(10 * time.Millisecond)
	}

	// Return false if deadline exceeded
	return false
}

// Helper: contextWithTimeout creates a context with timeout for test cleanup
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// Helper: startAlertProcessor starts the alert manager's Run loop in a goroutine
// Returns the context, cancel func, and error channel
// Useful for managing processor lifecycle in tests
// Example usage:
//
//	ctx, cancel, errChan := startAlertProcessor(manager, 2*time.Second)
//	defer cancel()
//	// ...send metrics...
//	err := <-errChan  // waits for processor to stop
func startAlertProcessor(manager *Manager, timeout time.Duration) (context.Context, context.CancelFunc, <-chan error) {
	// manager.Run() runs in a loop and returns when context is done
	// 1. Always provide a timeout context - Otherwise it runs forever
	// 2. Expect it to return context.Cancelled or context.DeadlineExceeded - This is normal/expected
	// 3. Don't treat it as a test failure - It's how you clean up the goroutine
	ctx, cancel := contextWithTimeout(timeout)
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Run(ctx)
	}()
	return ctx, cancel, errChan
}
