package alerting

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
)

// Manager handles alert rules and sending notifications
type Manager struct {
	rules           []AlertRule
	webhooks        []string // Webhook URLs for notifications
	alertQueue      chan AlertEvent
	ruleMutex       sync.RWMutex
	lastAlerts      map[string]time.Time // Track when we last alerted on each rule to avoid spam
	lastMetrics     map[string]float64   // Maps rule ID to previously observed metric value
	metricMutex     sync.Mutex
	alertMutex      sync.Mutex
	webhookConfig   WebhookConfig
	defaultCooldown int
}

// NewManager creates a new alert manager
func NewManager(config config.AlertingConfig) *Manager {
	// Convert config rules to alerting rules
	rules := make([]AlertRule, len(config.Rules))
	for i, cfgRule := range config.Rules {
		rules[i] = AlertRule{
			// Use ID from config if available, will be set below if empty
			ID:              cfgRule.ID,
			Name:            cfgRule.Name,
			MetricName:      cfgRule.MetricName,
			Condition:       cfgRule.Condition,
			Threshold:       cfgRule.Threshold,
			Severity:        cfgRule.Severity,
			Enabled:         true, // Rules are enabled by default
			CooldownSeconds: cfgRule.CooldownSeconds,
		}
		// Generate ID if not provided in config
		if rules[i].ID == "" {
			rules[i].ID = uuid.New().String()
		}
	}

	return &Manager{
		rules:           rules,
		webhooks:        config.Webhooks,
		alertQueue:      make(chan AlertEvent, config.QueueBufferSize),
		lastAlerts:      make(map[string]time.Time),
		lastMetrics:     make(map[string]float64),
		webhookConfig:   DefaultWebhookConfig(),
		defaultCooldown: config.CooldownSeconds,
	}
}

// AddRule adds a new alert rule
func (m *Manager) AddRule(rule AlertRule) error {
	m.ruleMutex.Lock()
	defer m.ruleMutex.Unlock()

	m.rules = append(m.rules, rule)
	return nil
}

// RemoveRule removes an alert rule by ID
func (m *Manager) RemoveRule(ruleID string) error {
	m.ruleMutex.Lock()
	defer m.ruleMutex.Unlock()

	for i, rule := range m.rules {
		if rule.ID == ruleID {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return nil
		}
	}

	return ErrRuleNotFound
}

// GetRules returns all current alert rules
func (m *Manager) GetRules() []AlertRule {
	m.ruleMutex.RLock()
	defer m.ruleMutex.RUnlock()

	// Return a copy to prevent external modification
	rules := make([]AlertRule, len(m.rules))
	copy(rules, m.rules)
	return rules
}

// formatMetricId Create a metric ID by concatenating the labels
func formatMetricId(alert *AlertEvent, metric types.Metric) {
	// Simple approach: name + account_id (if present)
	accountID := metric.Labels["account_id"]
	if accountID != "" {
		alert.MetricID = fmt.Sprintf("%s[%s]", metric.Name, accountID)
	} else {
		alert.MetricID = metric.Name
	}
	// Output: "account_balance[0.0.5000]"
}

// queueAlert creates and queues the alert
func (m *Manager) queueAlert(rule AlertRule, metric types.Metric) {
	// Create and queue the alert
	alert := AlertEvent{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Severity:  rule.Severity,
		Message:   rule.Description,
		Timestamp: time.Now().Unix(),
		Value:     metric.Value,
	}
	formatMetricId(&alert, metric)

	select {
	case m.alertQueue <- alert:
		m.alertMutex.Lock()
		m.lastAlerts[rule.ID] = time.Now()
		m.alertMutex.Unlock()
	default:
		log.Printf("[AlertManager] Alert queue full, dropping alert for rule %s", rule.ID)
	}
}

// CheckMetric evaluates a metric against all active rules
// If a rule condition is met, an alert is queued for sending
func (m *Manager) CheckMetric(metric types.Metric) error {
	m.ruleMutex.RLock()
	rules := m.GetRules()
	m.ruleMutex.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// Skip rules that don't apply to this metric
		if rule.MetricName != metric.Name {
			continue
		}

		log.Printf("[AlertManager] Evaluating metric against rule: %s", rule.ID)

		// Extract and compare to actual metric value
		m.metricMutex.Lock()
		previousValue := m.lastMetrics[rule.ID]
		m.metricMutex.Unlock()

		shouldAlert := rule.EvaluateCondition(metric.Value, previousValue)

		if shouldAlert {
			// Check if we recently alerted on this rule to avoid spam
			m.alertMutex.Lock()
			lastAlert, exists := m.lastAlerts[rule.ID]
			m.alertMutex.Unlock()

			cooldownSeconds := rule.CooldownSeconds
			if cooldownSeconds == 0 {
				cooldownSeconds = m.defaultCooldown
			}
			cooldown := time.Duration(cooldownSeconds) * time.Second
			if exists && time.Since(lastAlert) < cooldown {
				log.Printf("[AlertManager] Skipping alert for rule %s (cooldown period)", rule.ID)
				continue
			}

			m.queueAlert(rule, metric)
		}

		// Update previousValue
		m.metricMutex.Lock()
		m.lastMetrics[rule.ID] = metric.Value
		m.metricMutex.Unlock()
	}

	return nil
}

// Run starts the alert manager's main loop
// It processes queued alerts and sends notifications via webhooks
func (m *Manager) Run(ctx context.Context) error {
	log.Println("[AlertManager] Starting alert processor")

	for {
		select {
		case <-ctx.Done():
			log.Println("[AlertManager] Stopping alert processor")
			return ctx.Err()
		case alert := <-m.alertQueue:
			log.Printf("[AlertManager] Alert triggered: %s (severity: %s, value: %.2f)",
				alert.RuleName, alert.Severity, alert.Value)

			// Send to webhooks in parallel using goroutines
			for _, webhook := range m.webhooks {
				go m.sendWebhook(webhook, alert)
			}
		}
	}
}

// sendWebhook sends an alert to a webhook URL
// Uses HTTP POST with retry logic and exponential backoff
func (m *Manager) sendWebhook(webhookURL string, alert AlertEvent) {
	payload := WebhookPayload{
		RuleID:    alert.RuleID,
		RuleName:  alert.RuleName,
		Severity:  alert.Severity,
		Message:   alert.Message,
		Value:     alert.Value,
		Timestamp: alert.Timestamp,
	}

	err := SendWebhookRequest(webhookURL, payload, m.webhookConfig)
	if err != nil {
		log.Printf("[AlertManager] Failed to send webhook to %s for alert %s: %v",
			webhookURL, alert.RuleID, err)
	}
}
