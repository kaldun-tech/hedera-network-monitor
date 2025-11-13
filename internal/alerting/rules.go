package alerting

// AlertRule defines a condition that triggers an alert
type AlertRule struct {
	ID              string
	Name            string
	Description     string
	MetricName      string // The metric this rule applies to
	Condition       string // Condition language defined in EvaluateCondition
	Threshold       float64
	Enabled         bool
	Severity        string // "info", "warning", "critical"
	CooldownSeconds int    // Cooldown period between alerts in seconds (default: 300)
}

// AlertEvent represents a triggered alert
type AlertEvent struct {
	RuleID          string
	RuleName        string
	Severity        string
	Message         string
	Timestamp       int64
	MetricID        string // Reference to the metric that triggered this
	Value           float64
	CooldownSeconds int
}

// EvaluateCondition checks if a metric value satisfies the rule condition
func (r *AlertRule) EvaluateCondition(metricValue float64) bool {
	switch r.Condition {
	case ">":
		return metricValue > r.Threshold
	case "<":
		return metricValue < r.Threshold
	case ">=":
		return metricValue >= r.Threshold
	case "<=":
		return metricValue <= r.Threshold
	case "==":
		return metricValue == r.Threshold
	case "!=":
		return metricValue != r.Threshold
	case "changed":
		// TODO: Implement state tracking - requires previous metric value
		return false
	case "increased":
		// TODO: Implement state tracking - requires previous metric value
		return false
	case "decreased":
		// TODO: Implement state tracking - requires previous metric value
		return false
	default:
		return false
	}
}
