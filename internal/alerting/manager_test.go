package alerting

import (
	"errors"
	"testing"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
)

// TestNewManager tests manager initialization
func TestNewManager(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{"https://example.com/webhook"},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if len(manager.webhooks) != 1 {
		t.Errorf("Expected 1 webhook, got %d", len(manager.webhooks))
	}

	if manager.webhooks[0] != "https://example.com/webhook" {
		t.Errorf("Expected webhook https://example.com/webhook, got %s", manager.webhooks[0])
	}
}

// TestAddRule tests adding alert rules
func TestAddRule(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "test_rule_1",
		Name:       "Test Rule",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100.0,
		Enabled:    true,
		Severity:   "warning",
	}

	err := manager.AddRule(rule)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	if rules[0].ID != rule.ID {
		t.Errorf("Expected rule ID %s, got %s", rule.ID, rules[0].ID)
	}
}

// TestRemoveRule tests removing alert rules
func TestRemoveRule(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule1 := AlertRule{ID: "rule1", Name: "Rule 1", Enabled: true}
	rule2 := AlertRule{ID: "rule2", Name: "Rule 2", Enabled: true}

	err := manager.AddRule(rule1)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	err = manager.AddRule(rule2)
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	rules := manager.GetRules()
	if len(rules) != 2 {
		t.Fatalf("Expected 2 rules, got %d", len(rules))
	}

	// Remove first rule
	err = manager.RemoveRule("rule1")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	rules = manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule after removal, got %d", len(rules))
	}

	if rules[0].ID != "rule2" {
		t.Errorf("Expected remaining rule ID rule2, got %s", rules[0].ID)
	}
}

// TestRemoveRuleNotFound tests removing a non-existent rule
func TestRemoveRuleNotFound(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	err := manager.RemoveRule("nonexistent")
	if !errors.Is(err, ErrRuleNotFound) {
		t.Errorf("Expected ErrRuleNotFound, got %v", err)
	}
}

// TestGetRules tests getting all rules
func TestGetRules(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule1 := AlertRule{ID: "rule1", Name: "Rule 1", Enabled: true}
	rule2 := AlertRule{ID: "rule2", Name: "Rule 2", Enabled: false}

	if err := manager.AddRule(rule1); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
	if err := manager.AddRule(rule2); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	rules := manager.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	// Verify it's a copy, not a reference
	rules[0].Name = "Modified"
	rules = manager.GetRules()
	if rules[0].Name != "Rule 1" {
		t.Error("GetRules should return a copy, not a reference")
	}
}

// TestCheckMetricThresholdGreaterThan tests condition evaluation with >
func TestCheckMetricThresholdGreaterThan(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "gt_rule",
		Name:       "Greater Than Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  100.0,
		Enabled:    true,
		Severity:   "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// Test metric value above threshold
	metric := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)
	// Verify alert was queued
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "gt_rule" {
			t.Errorf("Expected alert for greater than, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}
}

// TestCheckMetricThresholdLessThan tests condition evaluation with <
func TestCheckMetricThresholdLessThan(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "lt_rule",
		Name:       "Less Than Test",
		MetricName: "test_metric",
		Condition:  "<",
		Threshold:  50.0,
		Enabled:    true,
		Severity:   "critical",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:      "test_metric",
		Value:     25.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)
	// Verify alert was queued
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "lt_rule" {
			t.Errorf("Expected alert for less than, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}
}

// TestCheckMetricThresholdGreaterThanEqual tests condition evaluation with >=
func TestCheckMetricThresholdGreaterThanEqual(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 0,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:              "gte_rule",
		Name:            "Greater Than Equal Test",
		Description:     "Alert when metric >= 100",
		MetricName:      "test_metric",
		Condition:       ">=",
		Threshold:       100.0,
		Enabled:         true,
		Severity:        "warning",
		CooldownSeconds: 0,
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// Test metric below threshold, should not alert
	metric := types.Metric{
		Name:      "test_metric",
		Value:     75.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		t.Errorf("Unexpected alert with ID %s", alert.RuleID)
	default:
		t.Log("No alert was queued as expected")
	}

	// Test metric value equal to threshold - should trigger alert
	metric.Value = 100.0
	metric.Timestamp = time.Now().Unix()
	beforeTime := time.Now().Unix()

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "gte_rule" {
			t.Errorf("Expected RuleID gte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "warning" {
			t.Errorf("Expected severity warning, got %s", alert.Severity)
		}
		if alert.Value != 100.0 {
			t.Errorf("Expected value 100.0, got %f", alert.Value)
		}
		if alert.Timestamp < beforeTime || alert.Timestamp > time.Now().Unix() {
			t.Errorf("Alert timestamp %d out of expected range", alert.Timestamp)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Test metric value greater than threshold - should trigger alert
	metric.Value = 150.0
	metric.Timestamp = time.Now().Unix()
	beforeTime = time.Now().Unix()

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "gte_rule" {
			t.Errorf("Expected RuleID gte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "warning" {
			t.Errorf("Expected severity warning, got %s", alert.Severity)
		}
		if alert.Value != 150.0 {
			t.Errorf("Expected value 150.0, got %f", alert.Value)
		}
		if alert.Timestamp < beforeTime || alert.Timestamp > time.Now().Unix() {
			t.Errorf("Alert timestamp %d out of expected range", alert.Timestamp)
		}
	default:
		t.Fatal("Expected second alert to be queued")
	}

	// Test with account_id label - verify MetricID is formatted correctly
	metric.Value = 125.0
	metric.Timestamp = time.Now().Unix()
	metric.Labels = map[string]string{"account_id": "0.0.5000"}

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "gte_rule" {
			t.Errorf("Expected RuleID gte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "warning" {
			t.Errorf("Expected severity warning, got %s", alert.Severity)
		}
		if alert.Value != 125.0 {
			t.Errorf("Expected value 125.0, got %f", alert.Value)
		}
		if alert.MetricID != "test_metric[0.0.5000]" {
			t.Errorf("Expected MetricID test_metric[0.0.5000], got %s", alert.MetricID)
		}
	default:
		t.Fatal("Expected third alert to be queued")
	}
}

// TestCheckMetricThresholdLessThanEqualTo tests condition evaluation with <=
func TestCheckMetricThresholdLessThanEqualTo(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 0,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:              "lte_rule",
		Name:            "Less Than Equal Test",
		Description:     "Alert when metric <= 50",
		MetricName:      "test_metric",
		Condition:       "<=",
		Threshold:       50.0,
		Enabled:         true,
		Severity:        "critical",
		CooldownSeconds: 0,
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// Test metric less than threshold - should trigger alert
	metric := types.Metric{
		Name:      "test_metric",
		Value:     25.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "lte_rule" {
			t.Errorf("Expected RuleID lte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "critical" {
			t.Errorf("Expected severity critical, got %s", alert.Severity)
		}
		if alert.Value != 25.0 {
			t.Errorf("Expected value 25.0, got %f", alert.Value)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Test metric value equal to threshold - should trigger alert
	metric.Value = 50.0
	metric.Timestamp = time.Now().Unix()
	beforeTime := time.Now().Unix()

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "lte_rule" {
			t.Errorf("Expected RuleID lte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "critical" {
			t.Errorf("Expected severity critical, got %s", alert.Severity)
		}
		if alert.Value != 50.0 {
			t.Errorf("Expected value 50.0, got %f", alert.Value)
		}
		if alert.Timestamp < beforeTime || alert.Timestamp > time.Now().Unix() {
			t.Errorf("Alert timestamp %d out of expected range", alert.Timestamp)
		}
	default:
		t.Fatal("Expected second alert to be queued")
	}

	// Test metric value greater than threshold - should NOT trigger alert
	metric.Value = 100.0
	metric.Timestamp = time.Now().Unix()

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		t.Errorf("Unexpected alert with ID %s (metric value exceeded threshold)", alert.RuleID)
	default:
		t.Log("No alert was queued as expected")
	}

	// Test with account_id label - verify MetricID is formatted correctly
	metric.Value = 30.0
	metric.Timestamp = time.Now().Unix()
	metric.Labels = map[string]string{"account_id": "0.0.7000"}

	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "lte_rule" {
			t.Errorf("Expected RuleID lte_rule, got %s", alert.RuleID)
		}
		if alert.Severity != "critical" {
			t.Errorf("Expected severity critical, got %s", alert.Severity)
		}
		if alert.Value != 30.0 {
			t.Errorf("Expected value 30.0, got %f", alert.Value)
		}
		if alert.MetricID != "test_metric[0.0.7000]" {
			t.Errorf("Expected MetricID test_metric[0.0.7000], got %s", alert.MetricID)
		}
	default:
		t.Fatal("Expected fourth alert to be queued")
	}
}

// TestCheckMetricThresholdEqual tests condition evaluation with ==
func TestCheckMetricThresholdEqual(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "eq_rule",
		Name:       "Equal Test",
		MetricName: "test_metric",
		Condition:  "==",
		Threshold:  100.0,
		Enabled:    true,
		Severity:   "info",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:      "test_metric",
		Value:     100.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)
	// Verify alert was queued
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "eq_rule" {
			t.Errorf("Expected alert for equals, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}
}

// TestCheckMetricDisabledRule tests that disabled rules are skipped
func TestCheckMetricDisabledRule(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "disabled_rule",
		Name:       "Disabled Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  50.0,
		Enabled:    false, // Disabled
		Severity:   "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:      "test_metric",
		Value:     100.0, // Value that would trigger if rule was enabled
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)

	// Verify no alert was queued (queue should still be empty)
	select {
	case <-manager.alertQueue:
		t.Error("Alert was queued for disabled rule")
	default:
		// Expected: no alert queued
	}
}

// TestCheckMetricNoMatch tests when metric doesn't match any rule
func TestCheckMetricNoMatch(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "rule1",
		Name:       "Test Rule",
		MetricName: "different_metric",
		Condition:  ">",
		Threshold:  100.0,
		Enabled:    true,
		Severity:   "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)

	// Rule applies to different metric name, so no alert
	select {
	case <-manager.alertQueue:
		t.Error("Alert was queued for non-matching metric name")
	default:
		// Expected: no alert queued
	}
}

// TestCheckMetricCooldown tests that cooldown prevents alert spam
func TestCheckMetricCooldown(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "spam_rule",
		Name:       "Spam Test",
		MetricName: "test_metric",
		Condition:  ">",
		Threshold:  50.0,
		Enabled:    true,
		Severity:   "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:      "test_metric",
		Value:     100.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}

	// First call should complete without error and queue an alert
	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("First CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(50 * time.Millisecond)

	// Verify first alert was queued
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "spam_rule" {
			t.Errorf("Expected alert for spam_rule, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Second call should also complete without error
	err = manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("Second CheckMetric failed: %v", err)
	}

	// Verify second alert was NOT queued due to cooldown
	select {
	case <-manager.alertQueue:
		t.Error("Second alert should not be queued due to cooldown period")
	default:
		// Expected: no alert queued due to cooldown
	}
}

// TestAlertEventCreation verifies AlertEvent structure
func TestAlertEventCreation(t *testing.T) {
	event := AlertEvent{
		RuleID:    "test_rule",
		RuleName:  "Test Rule",
		Severity:  "critical",
		Message:   "Test message",
		Timestamp: time.Now().Unix(),
		MetricID:  "test_metric",
		Value:     42.5,
	}

	if event.RuleID != "test_rule" {
		t.Errorf("Expected RuleID test_rule, got %s", event.RuleID)
	}

	if event.Value != 42.5 {
		t.Errorf("Expected Value 42.5, got %f", event.Value)
	}

	if event.Severity != "critical" {
		t.Errorf("Expected Severity critical, got %s", event.Severity)
	}
}

// TestCheckMetricStateChanged tests the "changed" state-tracking condition
func TestCheckMetricStateChanged(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "changed_rule",
		Name:       "Changed State Test",
		MetricName: "test_metric",
		Condition:  "changed",
		Enabled:    true,
		Severity:   "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// First metric: 100.0 (initializes state)
	metric1 := types.Metric{
		Name:      "test_metric",
		Value:     100.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err := manager.CheckMetric(metric1)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Second metric: 150.0 (different from previous - should trigger alert)
	metric2 := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric2)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Verify first alert was queued
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "changed_rule" {
			t.Errorf("Expected alert for changed, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Third metric: 150.0 (same as previous - should NOT trigger alert)
	metric3 := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric3)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}
	// If we got here without panic, state tracking worked correctly

	// Check second alert was not queued
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		t.Fatalf("Unexpected second alert with ID %s", alert.RuleID)
	default:
		t.Log("Second alert was correctly not queued")
	}
}

// TestCheckMetricStateIncreased tests the "increased" state-tracking condition
func TestCheckMetricStateIncreased(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "increased_rule",
		Name:       "Increased State Test",
		MetricName: "test_metric",
		Condition:  "increased",
		Enabled:    true,
		Severity:   "info",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// First metric: 100.0 (initializes state)
	metric1 := types.Metric{
		Name:      "test_metric",
		Value:     100.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err := manager.CheckMetric(metric1)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Second metric: 150.0 (increased from previous - should trigger alert)
	metric2 := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric2)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Verify first alert was queued
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "increased_rule" {
			t.Errorf("Expected alert for increased, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Third metric: 120.0 (decreased from previous - should NOT trigger alert)
	metric3 := types.Metric{
		Name:      "test_metric",
		Value:     120.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric3)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// If we got here without panic, state tracking worked correctly
	// Verify second alert was not triggered
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		t.Errorf("Unexpected second alert with ID %s", alert.RuleID)
	default:
		t.Log("Second alert was correctly not queued")
	}
}

// TestCheckMetricStateDecreased tests the "decreased" state-tracking condition
func TestCheckMetricStateDecreased(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:         "decreased_rule",
		Name:       "Decreased State Test",
		MetricName: "test_metric",
		Condition:  "decreased",
		Enabled:    true,
		Severity:   "critical",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// First metric: 150.0 (initializes state)
	metric1 := types.Metric{
		Name:      "test_metric",
		Value:     150.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err := manager.CheckMetric(metric1)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Second metric: 100.0 (decreased from previous - should trigger alert)
	metric2 := types.Metric{
		Name:      "test_metric",
		Value:     100.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric2)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Verify first alert was queued
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		if alert.RuleID != "decreased_rule" {
			t.Errorf("Expected alert for decreased, got %s", alert.RuleID)
		}
	default:
		t.Fatal("Expected first alert to be queued")
	}

	// Third metric: 120.0 (increased from previous - should NOT trigger alert)
	metric3 := types.Metric{
		Name:      "test_metric",
		Value:     120.0,
		Timestamp: time.Now().Unix(),
		Labels:    map[string]string{},
	}
	err = manager.CheckMetric(metric3)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// If we got here without panic, state tracking worked correctly
	// Verify second alert was not triggered
	time.Sleep(100 * time.Millisecond)
	select {
	case alert := <-manager.alertQueue:
		t.Errorf("Unexpected second alert with ID %s", alert.RuleID)
	default:
		t.Log("Second alert was correctly not queued")
	}
}
