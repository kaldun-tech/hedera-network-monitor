package config

import (
	"os"
	"testing"

	"github.com/kaldun-tech/hedera-network-monitor/internal/collector"
)

func TestValidate_ValidConfig(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{
			Name: "testnet",
		},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test Account"},
		},
		Alerting: AlertingConfig{
			Enabled: false,
		},
		API: APIConfig{
			Port: 8080,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error for valid config, got: %v", err)
	}
}

func TestValidate_InvalidNetwork(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "invalid"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
	}
	err := config.Validate()
	if err == nil {
		t.Error("expected error for invalid network name")
	}
}

func TestValidate_InvalidNetworkMainnet(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "mainnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		API: APIConfig{
			Port: 8080,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error for mainnet, got: %v", err)
	}
}

func TestValidate_NoAccounts(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{},
	}
	err := config.Validate()
	if err == nil {
		t.Error("expected error when no accounts configured")
	}
}

func TestValidate_AlertingEnabledNoRules(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		Alerting: AlertingConfig{
			Enabled: true,
			Rules:   []AlertRule{},
		},
	}
	err := config.Validate()
	if err == nil {
		t.Error("expected error when alerting enabled but no rules configured")
	}
}

func TestValidate_AlertingDisabledNoRules(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		Alerting: AlertingConfig{
			Enabled: false,
			Rules:   []AlertRule{},
		},
		API: APIConfig{
			Port: 8080,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error when alerting disabled, got: %v", err)
	}
}

func TestValidate_InvalidAPIPort_TooLow(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		API: APIConfig{
			Port: 0,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Error("expected error for port 0")
	}
}

func TestValidate_InvalidAPIPort_TooHigh(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		API: APIConfig{
			Port: 65536,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err == nil {
		t.Error("expected error for port 65536")
	}
}

func TestValidate_ValidAPIPort_Max(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		API: APIConfig{
			Port: 65535,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error for port 65535, got: %v", err)
	}
}

func TestValidate_ValidAPIPort_Min(t *testing.T) {
	config := &Config{
		Network: NetworkConfig{Name: "testnet"},
		Accounts: []collector.AccountConfig{
			{ID: "0.0.5000", Label: "Test"},
		},
		API: APIConfig{
			Port: 1,
			Host: "localhost",
		},
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("expected no error for port 1, got: %v", err)
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := getDefaultConfig()

	if config.Network.Name != "testnet" {
		t.Errorf("expected default network to be testnet, got: %s", config.Network.Name)
	}

	if config.API.Port != 8080 {
		t.Errorf("expected default API port to be 8080, got: %d", config.API.Port)
	}

	if config.API.Host != "localhost" {
		t.Errorf("expected default API host to be localhost, got: %s", config.API.Host)
	}

	if config.Logging.Level != "info" {
		t.Errorf("expected default log level to be info, got: %s", config.Logging.Level)
	}

	if config.Logging.Format != "text" {
		t.Errorf("expected default log format to be text, got: %s", config.Logging.Format)
	}

	if !config.Alerting.Enabled {
		t.Error("expected default alerting to be enabled")
	}

	if len(config.Accounts) != 0 {
		t.Errorf("expected default accounts to be empty, got: %d", len(config.Accounts))
	}
}

func TestLoad_MissingFile_ReturnsDefaults(t *testing.T) {
	config, err := Load("/nonexistent/path/config.yaml")

	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}

	if config == nil {
		t.Error("expected default config to be returned")
	}

	if config.Network.Name != "testnet" {
		t.Errorf("expected default network, got: %s", config.Network.Name)
	}
}

func TestLoad_ValidYAMLFile(t *testing.T) {
	// Create a temporary config file
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid config
	content := `
network:
  name: testnet
  operator_id: "0.0.3"
  operator_key: "key123"
accounts:
  - id: "0.0.5000"
    label: "Main Account"
api:
  port: 8080
  host: "0.0.0.0"
logging:
  level: debug
  format: json
alerting:
  enabled: true
  webhooks:
    - "https://example.com/hook"
  rules:
    - id: "balance_low"
      name: "Low Balance Alert"
      metric_name: "account_balance"
      condition: "<"
      threshold: 1000000000
      severity: "warning"
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := Load(tmpFile.Name())

	if err != nil {
		t.Errorf("expected no error loading valid config, got: %v", err)
	}

	if config.Network.Name != "testnet" {
		t.Errorf("expected network testnet, got: %s", config.Network.Name)
	}

	if config.API.Port != 8080 {
		t.Errorf("expected port 8080, got: %d", config.API.Port)
	}

	if config.Logging.Level != "debug" {
		t.Errorf("expected log level debug, got: %s", config.Logging.Level)
	}

	if !config.Alerting.Enabled {
		t.Error("expected alerting to be enabled")
	}

	if len(config.Alerting.Rules) != 1 {
		t.Errorf("expected 1 alert rule, got: %d", len(config.Alerting.Rules))
	}
}

func TestLoad_InvalidYAMLFile(t *testing.T) {
	// Create a temporary config file with invalid YAML
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid YAML
	content := `
network:
  name: testnet
  invalid: [
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := Load(tmpFile.Name())

	// The current implementation returns defaults on parse failure with a warning
	// This test verifies that behavior
	if err != nil {
		// Error is acceptable but not required
	}

	if config == nil {
		t.Error("expected config to not be nil (should return defaults on parse failure)")
	}

	if config.Network.Name != "testnet" {
		t.Errorf("expected default network testnet, got: %s", config.Network.Name)
	}
}

func TestLoad_ValidFileInvalidConfig(t *testing.T) {
	// Create a temporary config file with valid YAML but invalid values
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write valid YAML but invalid config (no accounts)
	content := `
network:
  name: testnet
api:
  port: 8080
  host: "localhost"
alerting:
  enabled: false
`

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	tmpFile.Close()

	config, err := Load(tmpFile.Name())

	if err == nil {
		t.Error("expected error for invalid config (no accounts)")
	}

	if config != nil {
		t.Error("expected nil config for invalid configuration")
	}
}
