package alerting

import (
	"errors"
	"testing"
)

// TestErrRuleNotFound verifies the ErrRuleNotFound error
func TestErrRuleNotFound(t *testing.T) {
	if ErrRuleNotFound == nil {
		t.Fatal("ErrRuleNotFound is nil")
	}

	if ErrRuleNotFound.Error() != "alert rule not found" {
		t.Errorf("Expected error message 'alert rule not found', got '%s'", ErrRuleNotFound.Error())
	}

	// Verify it can be checked with errors.Is
	if !errors.Is(ErrRuleNotFound, ErrRuleNotFound) {
		t.Error("errors.Is failed for ErrRuleNotFound")
	}
}

// TestErrInvalidRule verifies the ErrInvalidRule error
func TestErrInvalidRule(t *testing.T) {
	if ErrInvalidRule == nil {
		t.Fatal("ErrInvalidRule is nil")
	}

	if ErrInvalidRule.Error() != "invalid alert rule" {
		t.Errorf("Expected error message 'invalid alert rule', got '%s'", ErrInvalidRule.Error())
	}

	if !errors.Is(ErrInvalidRule, ErrInvalidRule) {
		t.Error("errors.Is failed for ErrInvalidRule")
	}
}

// TestErrWebhookFailed verifies the ErrWebhookFailed error
func TestErrWebhookFailed(t *testing.T) {
	if ErrWebhookFailed == nil {
		t.Fatal("ErrWebhookFailed is nil")
	}

	if ErrWebhookFailed.Error() != "webhook notification failed" {
		t.Errorf("Expected error message 'webhook notification failed', got '%s'", ErrWebhookFailed.Error())
	}

	if !errors.Is(ErrWebhookFailed, ErrWebhookFailed) {
		t.Error("errors.Is failed for ErrWebhookFailed")
	}
}

// TestErrorsAreDistinct verifies that all error types are distinct
func TestErrorsAreDistinct(t *testing.T) {
	if ErrRuleNotFound == ErrInvalidRule {
		t.Error("ErrRuleNotFound should not equal ErrInvalidRule")
	}

	if ErrRuleNotFound == ErrWebhookFailed {
		t.Error("ErrRuleNotFound should not equal ErrWebhookFailed")
	}

	if ErrInvalidRule == ErrWebhookFailed {
		t.Error("ErrInvalidRule should not equal ErrWebhookFailed")
	}
}
