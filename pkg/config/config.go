package config

import (
	"fmt"
	"log"

	"github.com/kaldun-tech/hedera-network-monitor/internal/collector"
	"github.com/spf13/viper"
)

// Config represents the complete configuration for the monitor service
type Config struct {
	Network   NetworkConfig
	Accounts  []collector.AccountConfig
	Alerting  AlertingConfig
	API       APIConfig
	Logging   LoggingConfig
}

// NetworkConfig contains Hedera network configuration
type NetworkConfig struct {
	Name        string // "mainnet" or "testnet"
	OperatorID  string // "0.0.3"
	OperatorKey string // Private key for operator account
}

// AlertingConfig contains alert configuration
type AlertingConfig struct {
	Enabled  bool
	Webhooks []string // Webhook URLs for notifications
	Rules    []AlertRule
}

// AlertRule represents an alert configuration
type AlertRule struct {
	ID        string
	Name      string
	MetricName string
	Condition string
	Threshold float64
	Severity  string
}

// APIConfig contains API server configuration
type APIConfig struct {
	Port int    // Port to listen on
	Host string // Host to bind to
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json" or "text"
}

// Load loads configuration from a YAML file
func Load(configFile string) (*Config, error) {
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// Set defaults
	viper.SetDefault("network.name", "testnet")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.host", "localhost")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")
	viper.SetDefault("alerting.enabled", true)

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: Could not read config file %s: %v", configFile, err)
		// Return default config if file doesn't exist
		return getDefaultConfig(), nil
	}

	// Unmarshal configuration
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	log.Printf("Configuration loaded from: %s", viper.ConfigFileUsed())
	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// TODO: Add validation logic
	// - Network name must be valid
	// - Port must be valid (1-65535)
	// - Account IDs must be valid format
	// - At least one account must be configured for monitoring
	// - Webhook URLs must be valid

	if c.Network.Name != "mainnet" && c.Network.Name != "testnet" {
		return fmt.Errorf("invalid network name: %s", c.Network.Name)
	}

	if c.API.Port < 1 || c.API.Port > 65535 {
		return fmt.Errorf("invalid API port: %d", c.API.Port)
	}

	return nil
}

// getDefaultConfig returns a default configuration
func getDefaultConfig() *Config {
	return &Config{
		Network: NetworkConfig{
			Name: "testnet",
		},
		Accounts: make([]collector.AccountConfig, 0),
		Alerting: AlertingConfig{
			Enabled:  true,
			Webhooks: make([]string, 0),
			Rules:    make([]AlertRule, 0),
		},
		API: APIConfig{
			Port: 8080,
			Host: "localhost",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
