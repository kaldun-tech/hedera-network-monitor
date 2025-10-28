package alerting

import "errors"

var (
	// ErrRuleNotFound is returned when an alert rule is not found
	ErrRuleNotFound = errors.New("alert rule not found")

	// ErrInvalidRule is returned when an alert rule is invalid
	ErrInvalidRule = errors.New("invalid alert rule")

	// ErrWebhookFailed is returned when a webhook notification fails
	ErrWebhookFailed = errors.New("webhook notification failed")
)
