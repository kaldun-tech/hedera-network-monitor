package metrics

import (
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// MetricAggregator provides functions for aggregating metrics
type MetricAggregator struct {
	metrics []types.Metric
}

// NewMetricAggregator creates a new metric aggregator
func NewMetricAggregator() *MetricAggregator {
	return &MetricAggregator{
		metrics: make([]types.Metric, 0),
	}
}

// AddMetric adds a metric to the aggregator
func (ma *MetricAggregator) AddMetric(metric types.Metric) {
	ma.metrics = append(ma.metrics, metric)
}

// Average calculates the average value of metrics
func (ma *MetricAggregator) Average() float64 {
	if len(ma.metrics) == 0 {
		return 0
	}

	sum := 0.0
	for _, m := range ma.metrics {
		sum += m.Value
	}

	return sum / float64(len(ma.metrics))
}

// Min returns the minimum value
func (ma *MetricAggregator) Min() float64 {
	if len(ma.metrics) == 0 {
		return 0
	}

	min := ma.metrics[0].Value
	for _, m := range ma.metrics[1:] {
		if m.Value < min {
			min = m.Value
		}
	}

	return min
}

// Max returns the maximum value
func (ma *MetricAggregator) Max() float64 {
	if len(ma.metrics) == 0 {
		return 0
	}

	max := ma.metrics[0].Value
	for _, m := range ma.metrics[1:] {
		if m.Value > max {
			max = m.Value
		}
	}

	return max
}

// Count returns the number of metrics
func (ma *MetricAggregator) Count() int {
	return len(ma.metrics)
}

// TODO: Add more aggregation functions:
// - Percentile calculations
// - Rate of change
// - Moving average
// - Standard deviation
// - Time-series analysis
// - Anomaly detection
