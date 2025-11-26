package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// UNIT TESTS FOR ALERTS LIST COMMAND
// ============================================================================

// TestAlertListCommand_NoAlerts tests alerts list when no rules exist
func TestAlertListCommand_NoAlerts(t *testing.T) {
	// Mock API returning empty alerts
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := AlertListResponse{Alerts: []AlertRuleResponse{}, Count: 0}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	// Sets API URL to target the mock server
	setGlobalFlags(server.URL, "info")

	// Verify output contains "No alert rules configured"
	output := captureCommandOutput(t, func() error {
		return handleAlertsList()
	})

	if !strings.Contains(output, "No alert rules configured") {
		t.Errorf("Expected 'No alert rules configured', got: %s", output)
	}
}

// TestAlertListCommand_WithAlerts tests alerts list displays rules correctly
func TestAlertListCommand_WithAlerts(t *testing.T) {
	// Mock API with 3 rules
	rules := []AlertRuleResponse{
		{ID: "1", Name: "Rule 1", MetricName: "metric1", Condition: ">", Threshold: 100, Severity: "warning"},
		{ID: "2", Name: "Rule 2", MetricName: "metric2", Condition: "<", Threshold: 50, Severity: "critical"},
		{ID: "3", Name: "Rule 3", MetricName: "metric3", Condition: "==", Threshold: 10, Severity: "info"},
	}

	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := AlertListResponse{Alerts: rules, Count: len(rules)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	output := captureCommandOutput(t, func() error {
		return handleAlertsList()
	})

	// Verify it contains the length of 3
	if !strings.Contains(output, "3") {
		t.Errorf("Expected output to contain rule count '3', got: %s", output)
	}

	// Verify output contains rule count, names, and conditions
	for _, rule := range rules {
		if !strings.Contains(output, rule.ID) {
			t.Errorf("Expected alert rule with ID '%s', got: %s", rule.ID, output)
		}
		if !strings.Contains(output, rule.Name) {
			t.Errorf("Expected alert rule with Name '%s', got: %s", rule.Name, output)
		}
		if !strings.Contains(output, rule.Condition) {
			t.Errorf("Expected alert rule with Condition '%s', got: %s", rule.Condition, output)
		}
	}
}

// TestAlertListCommand_APIError tests handling of API errors
func TestAlertListCommand_APIError(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal Server Error"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	err := handleAlertsList()
	if err == nil {
		t.Errorf("Expected error on 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected error to contain '500', got: %v", err)
	}
}

// TestAlertListCommand_InvalidJSON tests handling of malformed API response
func TestAlertListCommand_InvalidJSON(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{invalid json}"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	err := handleAlertsList()
	if err == nil {
		t.Errorf("Expected error on invalid JSON, got nil")
	} else if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("Expected decode error, got: %v", err)
	}
}

// TestAlertListCommand_ConnectionRefused tests handling when API is unavailable
func TestAlertListCommand_ConnectionRefused(t *testing.T) {
	setGlobalFlags("http://localhost:1", "info")
	err := handleAlertsList()
	if err == nil {
		t.Errorf("Expected error on connection refused, got nil")
	}
}

// ============================================================================
// UNIT TESTS FOR ALERTS ADD COMMAND
// ============================================================================

// TestAlertAddCommand_ValidRule tests adding a valid alert rule
func TestAlertAddCommand_ValidRule(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		rule := AlertRuleResponse{
			ID:         "rule1",
			Name:       "Test Rule",
			MetricName: "account_balance",
			Condition:  ">",
			Threshold:  1000000000,
			Severity:   "warning",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rule)
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	err := handleAlertAdd(createValidRuleJSON())
	if err != nil {
		t.Errorf("Unexpected error for valid json: %v", err)
	}
}

// TestAlertAddCommand_InvalidJSON tests adding rule with malformed JSON
func TestAlertAddCommand_InvalidJSON(t *testing.T) {
	setGlobalFlags("http://localhost:8080", "info")
	err := handleAlertAdd(createInvalidRuleJSON())
	if err == nil {
		t.Errorf("Expected error on invalid JSON, got nil")
	} else if !strings.Contains(err.Error(), "parse") && !strings.Contains(err.Error(), "format") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

// TestAlertAddCommand_MissingRequiredField tests validation of required fields
func TestAlertAddCommand_MissingRequiredField(t *testing.T) {
	missingFieldTests := []struct {
		name     string
		ruleJSON string
	}{
		{
			name:     "missing_name",
			ruleJSON: `{"metric_name":"test","condition":">","threshold":100,"severity":"warning"}`,
		},
		{
			name:     "missing_metric_name",
			ruleJSON: `{"name":"test","condition":">","threshold":100,"severity":"warning"}`,
		},
		{
			name:     "missing_threshold",
			ruleJSON: `{"name":"test","metric_name":"test","condition":">","severity":"warning"}`,
		},
	}

	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Missing required field"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")

	for _, tt := range missingFieldTests {
		t.Run(tt.name, func(t *testing.T) {
			err := handleAlertAdd(tt.ruleJSON)
			if err == nil {
				t.Errorf("Expected error for %s", tt.name)
			}
		})
	}
}

// TestAlertAddCommand_InvalidCondition tests validation of condition operator
func TestAlertAddCommand_InvalidCondition(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid condition"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	ruleJSON := `{"name":"test","metric_name":"test","condition":">>","threshold":100,"severity":"warning"}`
	err := handleAlertAdd(ruleJSON)
	if err == nil {
		t.Errorf("Expected error on invalid condition, got nil")
	}
}

// TestAlertAddCommand_InvalidSeverity tests validation of severity level
func TestAlertAddCommand_InvalidSeverity(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Invalid severity"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	ruleJSON := `{"name":"test","metric_name":"test","condition":">","threshold":100,"severity":"urgent"}`
	err := handleAlertAdd(ruleJSON)
	if err == nil {
		t.Errorf("Expected error on invalid severity, got nil")
	}
}

// TestAlertAddCommand_APIError tests handling of API errors during creation
func TestAlertAddCommand_APIError(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Database error"))
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	err := handleAlertAdd(createValidRuleJSON())
	if err == nil {
		t.Errorf("Expected error on API failure, got nil")
	} else if !strings.Contains(err.Error(), "500") {
		t.Errorf("Expected 500 error, got: %v", err)
	}
}

// TestAlertAddCommand_WithOptionalFields tests adding rule with optional fields
func TestAlertAddCommand_WithOptionalFields(t *testing.T) {
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		rule := AlertRuleResponse{
			ID:              "rule1",
			Name:            "Test",
			Description:     "Test Description",
			MetricName:      "test_metric",
			Condition:       ">",
			Threshold:       100,
			Severity:        "warning",
			CooldownSeconds: 600,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(rule)
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	ruleJSON := `{"name":"Test","metric_name":"test_metric","condition":">","threshold":100,"severity":"warning","description":"Test Description","cooldown_seconds":600}`
	err := handleAlertAdd(ruleJSON)
	if err != nil {
		t.Errorf("Expected success with optional fields, got error: %v", err)
	}
}

// ============================================================================
// INTEGRATION TESTS FOR ALERTS COMMANDS
// ============================================================================

// TestAlertsIntegration_ListThenAdd tests full workflow: list then add rule
func TestAlertsIntegration_ListThenAdd(t *testing.T) {
	var rules []AlertRuleResponse

	// Create stateful mock API
	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// List endpoint
			response := AlertListResponse{Alerts: rules, Count: len(rules)}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
		} else if r.Method == http.MethodPost {
			// Add endpoint
			var newRule AlertRuleResponse
			_ = json.NewDecoder(r.Body).Decode(&newRule)
			newRule.ID = "rule1"
			rules = append(rules, newRule)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(newRule)
		}
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")

	// List empty rules
	err := handleAlertsList()
	if err != nil {
		t.Errorf("First list failed: %v", err)
	}

	// Add a rule
	err = handleAlertAdd(createValidRuleJSON())
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	// List again - should have the rule
	err = handleAlertsList()
	if err != nil {
		t.Errorf("Second list failed: %v", err)
	}
}

// TestAlertsIntegration_MultipleRules tests managing multiple alert rules
func TestAlertsIntegration_MultipleRules(t *testing.T) {
	rules := []AlertRuleResponse{
		{ID: "1", Name: "Rule 1", MetricName: "metric1", Condition: ">", Threshold: 100, Severity: "warning"},
		{ID: "2", Name: "Rule 2", MetricName: "metric2", Condition: "<", Threshold: 50, Severity: "critical"},
		{ID: "3", Name: "Rule 3", MetricName: "metric3", Condition: "==", Threshold: 10, Severity: "info"},
	}

	server := createMockAPIServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := AlertListResponse{Alerts: rules, Count: len(rules)}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	setGlobalFlags(server.URL, "info")
	err := handleAlertsList()
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

// ============================================================================
// UNIT TESTS FOR ACCOUNT COMMANDS
// ============================================================================

// TestAccountBalanceCommand_ValidAccount tests balance query for valid account
func TestAccountBalanceCommand_ValidAccount(t *testing.T) {
	// Mock hedera.Client, set credentials, verify balance output
	t.Skip("Implement account balance valid account test")
}

// TestAccountBalanceCommand_InvalidAccount tests balance query with invalid account
func TestAccountBalanceCommand_InvalidAccount(t *testing.T) {
	// Account balance command requires Hedera SDK mock
	// Simpler approach: just verify the command validates input format
	// Invalid account format should fail at parsing, before SDK call
	t.Skip("Implement account balance invalid account test")
}

// TestAccountBalanceCommand_NetworkError tests handling of network failures
func TestAccountBalanceCommand_NetworkError(t *testing.T) {
	// Account balance command requires Hedera SDK mock
	// To test network errors, you'd need to mock hedera.NewClient to return error
	t.Skip("Implement account balance network error test")
}

// TestAccountBalanceCommand_MissingAccountID tests argument validation
func TestAccountBalanceCommand_MissingAccountID(t *testing.T) {
	// This is a cobra argument validation test - can verify that command rejects missing args
	// The command has Args: cobra.ExactArgs(1), so no account ID should error
	// However, testing cobra commands requires calling Execute()
	t.Skip("Implement account balance missing ID test")
}

// TestAccountTransactionsCommand_ValidAccount tests transaction query
func TestAccountTransactionsCommand_ValidAccount(t *testing.T) {
	// Account transactions require Hedera SDK mocking
	// Would need to mock hedera.Client.GetAccountRecords()
	t.Skip("Implement account transactions valid account test")
}

// TestAccountTransactionsCommand_NoTransactions tests account with no transactions
func TestAccountTransactionsCommand_NoTransactions(t *testing.T) {
	// Account transactions require Hedera SDK mocking
	// Would return empty slice from GetAccountRecords()
	t.Skip("Implement account transactions no transactions test")
}

// TestAccountTransactionsCommand_InvalidAccount tests with invalid account
func TestAccountTransactionsCommand_InvalidAccount(t *testing.T) {
	// Account transactions require Hedera SDK mocking
	// Would mock NewClient() or GetAccountRecords() to return error
	t.Skip("Implement account transactions invalid account test")
}

// ============================================================================
// HELPER FUNCTIONS FOR TESTING
// ============================================================================

// createMockAPIServer creates an httptest.Server for testing CLI commands
func createMockAPIServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// captureCommandOutput captures stdout from running a cobra command
func captureCommandOutput(t *testing.T, cmdFunc func() error) string {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	os.Stdout = w

	go func() {
		_ = cmdFunc()
		w.Close()
	}()

	output, err := io.ReadAll(r)
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	return string(output)
}

// setGlobalFlags sets CLI global flags for testing
func setGlobalFlags(apiURLValue string, loglevelValue string) {
	apiURL = apiURLValue
	loglevel = loglevelValue
}

// createValidRuleJSON creates a valid alert rule JSON for testing
func createValidRuleJSON() string {
	return `{"name":"Test Rule","metric_name":"account_balance","condition":">","threshold":1000000000,"severity":"warning"}`
}

// createInvalidRuleJSON creates invalid alert rule JSON for testing
func createInvalidRuleJSON() string {
	return `{not valid json}`
}

// waitForServer waits for a server to be ready
//
//nolint:unused
func waitForServer(t *testing.T, url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
