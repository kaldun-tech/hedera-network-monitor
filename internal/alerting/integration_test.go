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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// manager.Run() runs in a loop and returns when context is done
	// 1. Always provide a timeout context - Otherwise it runs forever
	//  2. Expect it to return context.Cancelled or context.DeadlineExceeded - This is normal/expected
	//  3. Don't treat it as a test failure - It's how you clean up the goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- manager.Run(ctx)
	}()

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
	// TODO: Implement this test
	// Similar to above but:
	// 1. Add multiple alert rules (e.g., >, <, ==, changed)
	// 2. Send metric values that trigger multiple rules
	// 3. Verify all webhooks were called with correct payloads
	// 4. Test that each webhook receives separate calls
	t.Skip("Implement multiple rule alert dispatch test")
}

// TestEndToEndAlertWithCooldown tests cooldown prevents duplicate webhook calls
func TestEndToEndAlertWithCooldown(t *testing.T) {
	// TODO: Implement this test
	// 1. Create manager with short cooldown (e.g., 100ms)
	// 2. Add alert rule
	// 3. Send first metric that triggers alert
	// 4. Verify webhook called once
	// 5. Send second metric within cooldown period
	// 6. Verify webhook NOT called (only 1 total call still)
	// 7. Wait for cooldown to expire
	// 8. Send third metric
	// 9. Verify webhook called again (2 total calls now)
	t.Skip("Implement cooldown webhook prevention test")
}

// TestEndToEndWebhookPayloadFormat tests the webhook receives correct JSON structure
func TestEndToEndWebhookPayloadFormat(t *testing.T) {
	// TODO: Implement this test
	// 1. Create server that captures webhook request body
	// 2. Add alert rule with specific values
	// 3. Send metric that triggers alert
	// 4. Verify webhook received payload with:
	//    - Correct RuleID
	//    - Correct RuleName
	//    - Correct Severity
	//    - Correct Value
	//    - Valid Timestamp
	//    - Correct Message
	t.Skip("Implement webhook payload format test")
}

// TestEndToEndAlertWithLabels tests alerts work with labeled metrics
func TestEndToEndAlertWithLabels(t *testing.T) {
	// TODO: Implement this test
	// 1. Add alert rule for a metric
	// 2. Send metric with labels (e.g., account_id: "0.0.5000")
	// 3. Verify alert.MetricID includes the label (e.g., "metric_name[0.0.5000]")
	// 4. Verify webhook receives MetricID in payload
	t.Skip("Implement alert with labels test")
}

// TestEndToEndMultipleWebhooks tests alert is sent to multiple webhook endpoints
func TestEndToEndMultipleWebhooks(t *testing.T) {
	// TODO: Implement this test
	// 1. Create two separate httptest.Server endpoints
	// 2. Create manager with both webhook URLs
	// 3. Add alert rule
	// 4. Send metric that triggers alert
	// 5. Verify BOTH webhooks received the alert payload
	// 6. Verify payloads are identical
	t.Skip("Implement multiple webhooks test")
}

// TestEndToEndAlertDisabledRuleNoWebhook tests disabled rules don't trigger webhooks
func TestEndToEndAlertDisabledRuleNoWebhook(t *testing.T) {
	// TODO: Implement this test
	// 1. Create mock webhook server
	// 2. Add alert rule but set Enabled: false
	// 3. Send metric that would normally trigger the rule
	// 4. Verify webhook was NOT called
	t.Skip("Implement disabled rule webhook test")
}

// TestEndToEndContextCancellation tests graceful shutdown via context
func TestEndToEndContextCancellation(t *testing.T) {
	// TODO: Implement this test
	// 1. Create manager with alert rule
	// 2. Start manager.Run() with cancellable context
	// 3. Send a few metrics
	// 4. Cancel the context
	// 5. Verify manager.Run() returns without panicking
	// 6. Verify context.Cancelled error is returned
	t.Skip("Implement context cancellation test")
}

// TestEndToEndWebhookFailureHandling tests webhook retry on server errors
func TestEndToEndWebhookFailureHandling(t *testing.T) {
	// TODO: Implement this test
	// 1. Create server that fails first 2 requests, succeeds on 3rd
	// 2. Add alert rule
	// 3. Send metric that triggers alert
	// 4. Verify webhook eventually succeeds (checks for retry logic)
	// 5. Verify alert was logged/handled correctly
	t.Skip("Implement webhook failure handling test")
}

// TestEndToEndAlertQueueOverflow tests behavior when alert queue is full
func TestEndToEndAlertQueueOverflow(t *testing.T) {
	// TODO: Implement this test
	// 1. Create manager with very small queue buffer (e.g., 1)
	// 2. Add multiple alert rules
	// 3. Send many metrics rapidly to overflow queue
	// 4. Verify alert is dropped with log message
	// 5. Verify no panic occurs
	t.Skip("Implement alert queue overflow test")
}

// TestEndToEndStateTrackingWithWebhook tests state-tracking conditions (changed/increased/decreased)
// work correctly and trigger webhooks appropriately
func TestEndToEndStateTrackingWithWebhook(t *testing.T) {
	// TODO: Implement this test
	// 1. Create "changed" condition alert rule
	// 2. Send first metric: 100 (initializes state)
	// 3. Verify NO webhook called (first value, no previous to compare)
	// 4. Send second metric: 150 (different from previous)
	// 5. Verify webhook IS called
	// 6. Send third metric: 150 (same as previous)
	// 7. Verify webhook NOT called
	t.Skip("Implement state tracking with webhook test")
}

func defaultAlertManagerConfig(t *testing.T) config.AlertingConfig {
	return config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{"https://example.com/webhook"},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
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
		// 1. Verify request method is POST
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 2. Verify Content-Type is application/json
		contentTypeHeader := r.Header.Get("Content-Type")
		if contentTypeHeader != "application/json" {
			t.Errorf("Expected application/json, got %s", contentTypeHeader)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 3. Decode request body into WebhookPayload
		var payload WebhookPayload
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			t.Errorf("Error decoding JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 4. Lock mutex, append to calls slice, unlock
		mu.Lock()
		calls = append(calls, payload)
		mu.Unlock()

		// 5. Write 200 OK response
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
func waitForWebhookCall(callsPtr *[]WebhookPayload, expectedCount int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// TODO: Implement wait logic
		// 1. Lock mutex
		// 2. Check if len(calls) >= expectedCount
		// 3. Unlock mutex
		// 4. Return true if condition met
		// 5. Sleep 10ms and retry
		// 6. Return false if deadline exceeded

		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// contextWithTimeout creates a context with timeout for test cleanup
func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
