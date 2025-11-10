package config

import (
	"fmt"
	"log"

	"github.com/kaldun-tech/hedera-network-monitor/internal/collector"
	"github.com/spf13/viper"
)

// Config represents the complete configuration for the monitor service
type Config struct {
	Network  NetworkConfig
	Accounts []collector.AccountConfig
	Alerting AlertingConfig
	API      APIConfig
	Logging  LoggingConfig
}

// NetworkConfig contains Hedera network configuration
type NetworkConfig struct {
	Name        string // "mainnet" or "testnet"
	OperatorID  string // "0.0.3"
	OperatorKey string // Private key for operator account
}

// AlertingConfig contains alert configuration
type AlertingConfig struct {
	Enabled         bool
	Webhooks        []string // Webhook URLs for notifications
	Rules           []AlertRule
	CooldownSeconds int // Default cooldown for all rules (seconds)
}

// AlertRule represents an alert configuration
type AlertRule struct {
	ID              string
	Name            string
	MetricName      string
	Condition       string
	Threshold       float64
	Severity        string
	CooldownSeconds int // Optional: override default cooldown (0 = use AlertingConfig default)
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
	viper.SetDefault("alerting.cooldown_seconds", 300)

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
	// Network name must be valid
	if c.Network.Name != "mainnet" && c.Network.Name != "testnet" {
		return fmt.Errorf("invalid network name: %s", c.Network.Name)
	}

	// Account IDs must be valid format. At least one account must be configured for monitoring
	if c.Accounts == nil || len(c.Accounts) == 0 {
		return fmt.Errorf("no accounts configured")
	}

	// Webhook URLs must be valid
	if c.Alerting.Enabled && len(c.Alerting.Rules) == 0 {
		return fmt.Errorf("no alerting rules configured")
	}

	if c.Alerting.Enabled && c.Alerting.CooldownSeconds < 0 {
		return fmt.Errorf("invalid cooldown seconds: %d", c.Alerting.CooldownSeconds)
	}

	// Port must be in range [1: 65535]
	if c.API.Port < 1 || 65535 < c.API.Port {
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
			Enabled:         true,
			Webhooks:        make([]string, 0),
			Rules:           make([]AlertRule, 0),
			CooldownSeconds: 300,
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
