package alerting

// WebhookPayload represents the JSON payload sent to webhooks
type WebhookPayload struct {
	RuleID    string  `json:"rule_id"`
	RuleName  string  `json:"rule_name"`
	Severity  string  `json:"severity"`
	Message   string  `json:"message"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

// TODO: Implement webhook sender with:
// - HTTP client with timeouts
// - Retry logic with exponential backoff
// - Error logging and monitoring
// - Support for different webhook types (Slack, Discord, generic HTTP, etc.)
// - Webhook signature verification (HMAC)
// - Rate limiting to avoid hammering the webhook endpoint
