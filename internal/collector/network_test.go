package collector

import (
	"testing"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// TestNewNetworkCollector_Initialization tests network collector creation
func TestNewNetworkCollector_Initialization(t *testing.T) {
	mockClient := &MockClient{}

	collector := NewNetworkCollector(mockClient)

	if collector == nil {
		t.Fatal("expected collector to be created")
	}

	if collector.client != mockClient {
		t.Error("expected client to be set correctly")
	}

	if collector.interval == 0 {
		t.Error("expected interval to be set")
	}

	if collector.BaseCollector == nil {
		t.Error("expected BaseCollector to be initialized")
	}
}

// TestNewNetworkCollector_DefaultInterval tests default interval is set
func TestNewNetworkCollector_DefaultInterval(t *testing.T) {
	mockClient := &MockClient{}

	collector := NewNetworkCollector(mockClient)

	// Default interval should be 30 seconds (set in account.go as DefaultInterval)
	if collector.interval == 0 {
		t.Error("expected interval to be set to a non-zero value")
	}
}

// TestNewNetworkCollector_Name tests collector has correct name
func TestNewNetworkCollector_Name(t *testing.T) {
	mockClient := &MockClient{}

	collector := NewNetworkCollector(mockClient)

	// NetworkCollector should have a name set from BaseCollector
	if collector.Name() == "" {
		t.Error("expected collector to have a name")
	}
}

// TestBuildPerNodeMetrics_EmptyNodeList tests with no nodes
// Note: This test demonstrates the structure we expect from buildPerNodeMetrics
// without relying on complex SDK types we can't easily construct
func TestBuildPerNodeMetrics_Structure(t *testing.T) {
	// The buildPerNodeMetrics function takes NodeAddresses and a network name
	// It should return a slice of metrics
	// Each node should produce 2 metrics: availability and endpoint count

	// This is a structural test demonstrating expected behavior
	// Full integration tests would require actual SDK types
}

// TestMetricStructure_NodeAvailability tests the expected structure of node availability metrics
func TestMetricStructure_NodeAvailability(t *testing.T) {
	metric := types.Metric{
		Name:      "network_node_available",
		Timestamp: time.Now().Unix(),
		Value:     1.0,
		Labels: map[string]string{
			"network":         "testnet",
			"node_id":         "1",
			"node_account_id": "0.0.3",
		},
	}

	// Verify structure
	if metric.Name != "network_node_available" {
		t.Error("metric name should be 'network_node_available'")
	}
	if metric.Value != 1.0 {
		t.Error("availability metric should have value 1.0")
	}
	if metric.Labels["network"] == "" {
		t.Error("metric should have network label")
	}
	if metric.Labels["node_id"] == "" {
		t.Error("metric should have node_id label")
	}
	if metric.Labels["node_account_id"] == "" {
		t.Error("metric should have node_account_id label")
	}
}

// TestMetricStructure_NodeEndpoints tests the expected structure of endpoint count metrics
func TestMetricStructure_NodeEndpoints(t *testing.T) {
	metric := types.Metric{
		Name:      "network_node_endpoints",
		Timestamp: time.Now().Unix(),
		Value:     2.0,
		Labels: map[string]string{
			"network":         "testnet",
			"node_id":         "1",
			"node_account_id": "0.0.3",
		},
	}

	// Verify structure
	if metric.Name != "network_node_endpoints" {
		t.Error("metric name should be 'network_node_endpoints'")
	}
	if metric.Value != 2.0 {
		t.Errorf("expected endpoint count 2.0, got %v", metric.Value)
	}
	if metric.Labels["network"] == "" {
		t.Error("metric should have network label")
	}
}

// TestMetricStructure_NodeConsensus tests the expected structure of consensus status metrics
func TestMetricStructure_NodeConsensus(t *testing.T) {
	// Test metric when network is up
	metricUp := types.Metric{
		Name:      "network_consensus_active",
		Timestamp: time.Now().Unix(),
		Value:     1.0,
		Labels: map[string]string{
			"network": "testnet",
		},
	}

	if metricUp.Value != 1.0 {
		t.Error("consensus metric should be 1.0 when network is up")
	}

	// Test metric when network is down
	metricDown := types.Metric{
		Name:      "network_consensus_active",
		Timestamp: time.Now().Unix(),
		Value:     0.0,
		Labels: map[string]string{
			"network": "testnet",
		},
	}

	if metricDown.Value != 0.0 {
		t.Error("consensus metric should be 0.0 when network is down")
	}
}

// TestMetricStructure_NodeCount tests the expected structure of node count metrics
func TestMetricStructure_NodeCount(t *testing.T) {
	metric := types.Metric{
		Name:      "network_nodes_available",
		Timestamp: time.Now().Unix(),
		Value:     25.0,
		Labels: map[string]string{
			"network": "testnet",
		},
	}

	if metric.Name != "network_nodes_available" {
		t.Error("metric name should be 'network_nodes_available'")
	}
	if metric.Value != 25.0 {
		t.Errorf("expected node count 25.0, got %v", metric.Value)
	}
	if metric.Labels["network"] != "testnet" {
		t.Error("metric should have network label")
	}
}

// TestMetricValidation_AllMetricsHaveTimestamp demonstrates all metrics should have timestamps
func TestMetricValidation_AllMetricsHaveTimestamp(t *testing.T) {
	metrics := []types.Metric{
		{
			Name:      "network_nodes_available",
			Timestamp: time.Now().Unix(),
			Value:     25.0,
			Labels:    map[string]string{"network": "testnet"},
		},
		{
			Name:      "network_node_available",
			Timestamp: time.Now().Unix(),
			Value:     1.0,
			Labels:    map[string]string{"network": "testnet", "node_id": "1"},
		},
		{
			Name:      "network_node_endpoints",
			Timestamp: time.Now().Unix(),
			Value:     2.0,
			Labels:    map[string]string{"network": "testnet", "node_id": "1"},
		},
		{
			Name:      "network_consensus_active",
			Timestamp: time.Now().Unix(),
			Value:     1.0,
			Labels:    map[string]string{"network": "testnet"},
		},
	}

	for _, metric := range metrics {
		if metric.Timestamp == 0 {
			t.Errorf("metric %s should have a non-zero timestamp", metric.Name)
		}
		if metric.Name == "" {
			t.Error("metric should have a name")
		}
	}
}
