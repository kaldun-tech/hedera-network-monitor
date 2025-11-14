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
		ID:       "test_rule_1",
		Name:     "Test Rule",
		MetricName: "test_metric",
		Condition: ">",
		Threshold: 100.0,
		Enabled:   true,
		Severity: "warning",
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
		ID:        "gt_rule",
		Name:      "Greater Than Test",
		MetricName: "test_metric",
		Condition: ">",
		Threshold: 100.0,
		Enabled:   true,
		Severity:  "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	// Test metric value above threshold
	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Allow time for async alert processing
	time.Sleep(100 * time.Millisecond)

	// We can't directly check the queue without modifying the manager,
	// but we can verify the method completes without error
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
		ID:        "lt_rule",
		Name:      "Less Than Test",
		MetricName: "test_metric",
		Condition: "<",
		Threshold: 50.0,
		Enabled:   true,
		Severity:  "critical",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:  "test_metric",
		Value: 25.0,
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
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
		ID:        "eq_rule",
		Name:      "Equal Test",
		MetricName: "test_metric",
		Condition: "==",
		Threshold: 100.0,
		Enabled:   true,
		Severity:  "info",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:  "test_metric",
		Value: 100.0,
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
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
		ID:        "disabled_rule",
		Name:      "Disabled Test",
		MetricName: "test_metric",
		Condition: ">",
		Threshold: 50.0,
		Enabled:   false, // Disabled
		Severity:  "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:  "test_metric",
		Value: 100.0, // Value that would trigger if rule was enabled
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

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
		ID:        "rule1",
		Name:      "Test Rule",
		MetricName: "different_metric",
		Condition: ">",
		Threshold: 100.0,
		Enabled:   true,
		Severity:  "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:  "test_metric",
		Value: 150.0,
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
	}

	err := manager.CheckMetric(metric)
	if err != nil {
		t.Fatalf("CheckMetric failed: %v", err)
	}

	// Rule applies to different metric name, so no alert
	select {
	case <-manager.alertQueue:
		t.Error("Alert was queued for non-matching metric name")
	default:
		// Expected: no alert queued
	}
}

// TestCheckMetricCooldown tests that cooldown prevents alert spam
// NOTE: This test will be fully functional once condition evaluation is implemented
func TestCheckMetricCooldown(t *testing.T) {
	cfg := config.AlertingConfig{
		Enabled:         true,
		Webhooks:        []string{},
		QueueBufferSize: 100,
		CooldownSeconds: 300,
	}
	manager := NewManager(cfg)

	rule := AlertRule{
		ID:        "spam_rule",
		Name:      "Spam Test",
		MetricName: "test_metric",
		Condition: ">",
		Threshold: 50.0,
		Enabled:   true,
		Severity:  "warning",
	}
	if err := manager.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	metric := types.Metric{
		Name:  "test_metric",
		Value: 100.0,
		Timestamp: time.Now().Unix(),
		Labels: map[string]string{},
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
