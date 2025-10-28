package types

// Metric represents a single metric data point
type Metric struct {
	Name      string            // Name of the metric (e.g., "account_balance")
	Timestamp int64             // Unix timestamp
	Value     float64           // Numeric value
	Labels    map[string]string // Additional labels for classification
}
