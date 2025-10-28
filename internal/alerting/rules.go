package alerting

// AlertRule defines a condition that triggers an alert
type AlertRule struct {
	ID          string
	Name        string
	Description string
	MetricName  string // The metric this rule applies to
	Condition   string // TODO: Define condition language (e.g., ">1000", "<0.1", "changed")
	Threshold   float64
	Enabled     bool
	Severity    string // "info", "warning", "critical"
}

// AlertEvent represents a triggered alert
type AlertEvent struct {
	RuleID    string
	RuleName  string
	Severity  string
	Message   string
	Timestamp int64
	MetricID  string // Reference to the metric that triggered this
	Value     float64
}
