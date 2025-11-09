package alerting

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// Manager handles alert rules and sending notifications
type Manager struct {
	rules          []AlertRule
	webhooks       []string // Webhook URLs for notifications
	alertQueue     chan AlertEvent
	ruleMutex      sync.RWMutex
	lastAlerts     map[string]time.Time // Track when we last alerted on each rule to avoid spam
	alertMutex     sync.Mutex
	webhookConfig  WebhookConfig
}

// NewManager creates a new alert manager
func NewManager(webhooks []string) *Manager {
	return &Manager{
		rules:          make([]AlertRule, 0),
		webhooks:       webhooks,
		alertQueue:     make(chan AlertEvent, 100), // TODO: Make buffer size configurable
		lastAlerts:     make(map[string]time.Time),
		webhookConfig:  DefaultWebhookConfig(),
	}
}

// AddRule adds a new alert rule
func (m *Manager) AddRule(rule AlertRule) error {
	m.ruleMutex.Lock()
	defer m.ruleMutex.Unlock()

	// TODO: Validate rule
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

		// TODO: Implement condition evaluation with actual metric
		// This is a placeholder - once import cycle is resolved, use collector.Metric
		// For now, just log that we would evaluate
		log.Printf("[AlertManager] Would evaluate metric against rule: %s", rule.ID)

		// TODO: Implement actual metric value extraction and comparison
		// Example: simple threshold check
		shouldAlert := false
		// switch rule.Condition {
		// case ">":
		//     shouldAlert = metric.Value > rule.Threshold
		// case "<":
		//     shouldAlert = metric.Value < rule.Threshold
		// case "==":
		//     shouldAlert = metric.Value == rule.Threshold
		// }

		if shouldAlert {
			// Check if we recently alerted on this rule to avoid spam
			m.alertMutex.Lock()
			lastAlert, exists := m.lastAlerts[rule.ID]
			m.alertMutex.Unlock()

			// TODO: Make cooldown period configurable
			cooldown := 5 * time.Minute
			if exists && time.Since(lastAlert) < cooldown {
				log.Printf("[AlertManager] Skipping alert for rule %s (cooldown period)", rule.ID)
				continue
			}

			// Create and queue the alert
			alert := AlertEvent{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Severity:  rule.Severity,
				Message:   rule.Description,
				Timestamp: time.Now().Unix(),
				Value:     0.0, // TODO: Extract from metric
			}

			select {
			case m.alertQueue <- alert:
				m.alertMutex.Lock()
				m.lastAlerts[rule.ID] = time.Now()
				m.alertMutex.Unlock()
			default:
				log.Printf("[AlertManager] Alert queue full, dropping alert for rule %s", rule.ID)
			}
		}
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
			// TODO: Send alert to webhooks
			log.Printf("[AlertManager] Alert triggered: %s (severity: %s, value: %.2f)",
				alert.RuleName, alert.Severity, alert.Value)

			// Send to webhooks (parallel)
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
