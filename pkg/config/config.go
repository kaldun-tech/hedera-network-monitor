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
	Name        string `mapstructure:"name"` // "mainnet" or "testnet"
	OperatorID  string `mapstructure:"operator_id"` // "0.0.3"
	OperatorKey string `mapstructure:"operator_key"` // Private key for operator account
}

// AlertingConfig contains alert configuration
type AlertingConfig struct {
	Enabled         bool `mapstructure:"enabled"`
	Webhooks        []string `mapstructure:"webhooks"` // Webhook URLs for notifications
	Rules           []AlertRule `mapstructure:"rules"`
	CooldownSeconds int `mapstructure:"cooldown_seconds"` // Default cooldown for all rules (seconds)
	QueueBufferSize int `mapstructure:"queue_buffer_size"` // Alert queue buffer size (default: 100)
}

// AlertRule represents an alert configuration
type AlertRule struct {
	ID              string `mapstructure:"id"`
	Name            string `mapstructure:"name"`
	MetricName      string `mapstructure:"metric_name"`
	Condition       string `mapstructure:"condition"`
	Threshold       float64 `mapstructure:"threshold"`
	Severity        string `mapstructure:"severity"`
	CooldownSeconds int `mapstructure:"cooldown_seconds"` // Optional: override default cooldown (0 = use AlertingConfig default)
}

// APIConfig contains API server configuration
type APIConfig struct {
	Port int `mapstructure:"port"` // Port to listen on
	Host string `mapstructure:"host"` // Host to bind to
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"` // "debug", "info", "warn", "error"
	Format string `mapstructure:"format"` // "json" or "text"
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
	viper.SetDefault("alerting.queue_buffer_size", 100)

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

// Validate checks if the alert rule is valid
func (r *AlertRule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}
	if r.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	if r.MetricName == "" {
		return fmt.Errorf("rule metric name cannot be empty")
	}

	// Validate condition is supported
	validConditions := []string{">", "<", ">=", "<=", "==", "!=", "changed", "increased", "decreased"}
	isValid := false
	for _, vc := range validConditions {
		if r.Condition == vc {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid condition: %s", r.Condition)
	}

	// Validate severity
	validSeverities := []string{"info", "warning", "critical"}
	isSevere := false
	for _, vs := range validSeverities {
		if r.Severity == vs {
			isSevere = true
			break
		}
	}
	if !isSevere {
		return fmt.Errorf("invalid severity: %s", r.Severity)
	}

	if r.CooldownSeconds < 0 {
		return fmt.Errorf("cooldown seconds cannot be negative: %d", r.CooldownSeconds)
	}

	return nil
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

	// Validate all rules
	for i, rule := range c.Alerting.Rules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("invalid rule at index %d: %w", i, err)
		}
	}
	// Alerting cooldown seconds must be positive
	if c.Alerting.Enabled && c.Alerting.CooldownSeconds <= 0 {
		return fmt.Errorf("invalid cooldown seconds: %d", c.Alerting.CooldownSeconds)
	}

	// Alerting queue buffer size must be positive
	if c.Alerting.Enabled && c.Alerting.QueueBufferSize <= 0 {
		return fmt.Errorf("invalid alert queue buffer size: %d", c.Alerting.QueueBufferSize)
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
			QueueBufferSize: 100,
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
