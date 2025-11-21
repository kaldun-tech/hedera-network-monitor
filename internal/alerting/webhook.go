package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"
)

// WebhookPayload represents the JSON payload sent to webhooks
type WebhookPayload struct {
	RuleID    string  `json:"rule_id"`
	RuleName  string  `json:"rule_name"`
	Severity  string  `json:"severity"`
	Message   string  `json:"message"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	MetricID  string  `json:"metric_id"`
}

// WebhookConfig holds configuration for webhook sending
type WebhookConfig struct {
	Timeout        time.Duration
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultWebhookConfig returns sensible defaults for webhook sending
func DefaultWebhookConfig() WebhookConfig {
	return WebhookConfig{
		Timeout:        10 * time.Second,
		MaxRetries:     5,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     32 * time.Second,
	}
}

// SendWebhookRequest sends a webhook request with retry logic
// Uses exponential backoff for retries
// Returns error if all retries fail
func SendWebhookRequest(webhookURL string, payload WebhookPayload, config WebhookConfig) error {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		req, err := http.NewRequest("POST", webhookURL, bytes.NewReader(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "hedera-network-monitor/1.0")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < config.MaxRetries {
				backoff := calculateBackoff(attempt, config.InitialBackoff, config.MaxBackoff)
				log.Printf("[AlertManager] Webhook request failed (attempt %d/%d): %v. Retrying in %v",
					attempt+1, config.MaxRetries+1, err, backoff)
				time.Sleep(backoff)
				continue
			}
			return fmt.Errorf("webhook request failed after %d retries: %w", config.MaxRetries+1, lastErr)
		}

		// Check if response status is success (2xx)
		if 200 <= resp.StatusCode && resp.StatusCode < 300 {
			// Read and discard response body to close connection
			_, _ = io.ReadAll(resp.Body)
			log.Printf("[AlertManager] Webhook sent successfully to %s (status: %d)", webhookURL, resp.StatusCode)
			_ = resp.Body.Close()
			return nil
		}

		// Non-2xx response
		bodyBytes, _ := io.ReadAll(resp.Body)
		err = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
		}

		if attempt < config.MaxRetries {
			// Retry on non-2xx responses
			backoff := calculateBackoff(attempt, config.InitialBackoff, config.MaxBackoff)
			log.Printf("[AlertManager] Webhook returned %d (attempt %d/%d). Retrying in %v",
				resp.StatusCode, attempt+1, config.MaxRetries+1, backoff)
			time.Sleep(backoff)
			continue
		}
	}

	return fmt.Errorf("webhook failed after %d retries: %w", config.MaxRetries+1, lastErr)
}

// calculateBackoff returns exponential backoff duration with jitter
func calculateBackoff(attempt int, initialBackoff, maxBackoff time.Duration) time.Duration {
	// Exponential backoff: initialBackoff * 2^attempt
	backoff := time.Duration(math.Min(
		float64(initialBackoff)*math.Pow(2, float64(attempt)),
		float64(maxBackoff),
	))
	return backoff
}
