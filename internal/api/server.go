package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/internal/types"
)

// Response types for standardized API responses

// MetricsResponse wraps metric results with count and error info
type MetricsResponse struct {
	Metrics []types.Metric `json:"metrics"`
	Count   int            `json:"count"`
	Error   string         `json:"error,omitempty"`
}

// HealthResponse represents the service health status
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// StatsResponse represents storage statistics
type StatsResponse struct {
	MetricCount int    `json:"metric_count"`
	MaxSize     int    `json:"max_size"`
	Utilization string `json:"utilization"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// AlertRuleResponse represents an alert rule in API responses
type AlertRuleResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	MetricName      string  `json:"metric_name"`
	Condition       string  `json:"condition"`
	Threshold       float64 `json:"threshold"`
	Severity        string  `json:"severity"`
	Enabled         bool    `json:"enabled"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

// AlertListResponse wraps a list of alert rules
type AlertListResponse struct {
	Alerts []AlertRuleResponse `json:"alerts"`
	Count  int                 `json:"count"`
}

// CreateAlertRequest represents the payload for creating an alert rule
type CreateAlertRequest struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	MetricName      string  `json:"metric_name"`
	Condition       string  `json:"condition"`
	Threshold       float64 `json:"threshold"`
	Severity        string  `json:"severity"`
	CooldownSeconds int     `json:"cooldown_seconds"`
}

// AlertingManager interface defines the contract for alert management
// This allows the Server to work with both the real AlertManager and test mocks
type AlertingManager interface {
	GetRules() []alerting.AlertRule
	AddRule(rule alerting.AlertRule) error
	RemoveRule(ruleID string) error
}

// Server represents the HTTP API server
type Server struct {
	port         int
	store        storage.Storage
	alertManager AlertingManager
	server       *http.Server
}

// NewServer creates a new API server
func NewServer(port int, store storage.Storage, alertManager AlertingManager) *Server {
	return &Server{
		port:         port,
		store:        store,
		alertManager: alertManager,
	}
}

// Helper functions for JSON response handling

// writeJSON encodes data to JSON and writes it to the response
// Uses json.NewEncoder which properly handles errors
// Code: HTTP status code (200, 400, 500, etc.)
// data: struct to marshal to JSON
func (s *Server) writeJSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError writes an error response to the client
// code: HTTP status code
// message: error message to send
func (s *Server) writeError(w http.ResponseWriter, code int, message string) {
	s.writeJSON(w, code, ErrorResponse{Error: message})
}

// Start starts the HTTP server and blocks until context is cancelled
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/metrics/account", s.handleMetricsByLabel)
	mux.HandleFunc("/api/v1/storage/stats", s.handleStorageStats)
	mux.HandleFunc("/api/v1/alerts", s.handleAlerts)
	// TODO: Add more handlers:
	// - WebSocket endpoint for real-time metrics

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("HTTP API server listening on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down API server")
	return s.server.Shutdown(shutdownCtx)
}

// handleHealth returns service health status
// GET /health
// No query parameters required
// Returns: HealthResponse with status and version
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check if request method is GET
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Create HealthResponse struct and call s.writeJSON() with 200 status and response
	s.writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "0.1.0",
	})
}

const DefaultLimit = 100
const MaxLimit = 10000

// handleMetrics returns metrics based on query parameters
// GET /api/v1/metrics
// Query parameters:
//   - name: metric name filter (optional, empty string = all)
//   - limit: maximum number of results (optional, default 100, max 10000)
//
// Returns: MetricsResponse with metrics slice and count
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Check method is GET
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Parse query parameters:
	name := r.URL.Query().Get("name")
	limitStr := r.URL.Query().Get("limit")

	// Parse limit as integer
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		log.Printf("Invalid limit, using default %d", DefaultLimit)
		limit = DefaultLimit
	} else if MaxLimit < limit {
		log.Printf("Limit too high, using max %d", MaxLimit)
		limit = MaxLimit
	}

	// Query storage
	metrics, err := s.store.GetMetrics(name, limit)
	if err != nil {
		log.Printf("Error retrieving metrics: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve metrics")
		return
	}

	// Handle nil case (convert to empty slice for JSON)
	if metrics == nil {
		metrics = []types.Metric{}
	}

	// Create MetricsResponse with 200 status, metrics slice and len(metrics)
	s.writeJSON(w, http.StatusOK, MetricsResponse{
		Metrics: metrics,
		Count:   len(metrics),
	})
}

// handleMetricsByLabel returns metrics filtered by label
// GET /api/v1/metrics/account
// Query parameters:
//   - key: label key (required, e.g. "account_id")
//   - value: label value (required, e.g. "0.0.5000")
//
// Returns: MetricsResponse with filtered metrics
func (s *Server) handleMetricsByLabel(w http.ResponseWriter, r *http.Request) {
	// Check method is GET
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Extract label key and value
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")

	// Validate required parameters
	if key == "" || value == "" {
		s.writeError(w, http.StatusBadRequest, "key and value query parameters required")
		return
	}

	// Query storage
	metrics, err := s.store.GetMetricsByLabel(key, value)
	if err != nil {
		log.Printf("Error retrieving metrics by label: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve metrics")
		return
	}

	// Handle nil case by converting to empty slice
	if metrics == nil {
		metrics = []types.Metric{}
	}

	// Create MetricsResponse and write JSON
	s.writeJSON(w, http.StatusOK, MetricsResponse{
		Metrics: metrics,
		Count:   len(metrics),
	})
}

// handleStorageStats returns storage statistics
// GET /api/v1/storage/stats
// No query parameters
// Returns: StatsResponse with metric count, max size, and utilization percentage
func (s *Server) handleStorageStats(w http.ResponseWriter, r *http.Request) {
	// Check method is GET
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Type assert storage supports Stats
	statsProvider, ok := s.store.(interface {
		Stats() (map[string]interface{}, error)
	})

	// If not supported, return 501 NotImplemented
	if !ok {
		s.writeError(w, http.StatusNotImplemented, "storage backend does not support stats")
		return
	}

	// Get stats from storage
	stats, err := statsProvider.Stats()
	if err != nil {
		log.Printf("Error retrieving storage stats: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve stats")
		return
	}

	// Create StatsResponse from returned map
	metricCount := stats["metric_count"].(int)
	maxSize := stats["max_size"].(int)
	utilization := stats["utilization"].(string)

	response := StatsResponse{
		MetricCount: metricCount,
		MaxSize:     maxSize,
		Utilization: utilization,
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleAlerts handles alert rule management endpoints
// Supports:
//   - GET /api/v1/alerts - List all alert rules
//   - POST /api/v1/alerts - Create a new alert rule
//   - DELETE /api/v1/alerts/{id} - Delete an alert rule
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListAlerts(w, r)
	case http.MethodPost:
		s.handleCreateAlert(w, r)
	case http.MethodDelete:
		s.handleDeleteAlert(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleListAlerts returns all configured alert rules
// GET /api/v1/alerts
// Returns: AlertListResponse with all alert rules
func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	// Get all rules from alertManager using GetRules()
	log.Printf("[API] GET /api/v1/alerts")
	allRules := s.alertManager.GetRules()

	// Convert alerting.AlertRule to AlertRuleResponse
	alertResponseList := make([]AlertRuleResponse, len(allRules))
	for i, rule := range allRules {
		ruleResponse := AlertRuleResponse{
			ID:              rule.ID,
			Name:            rule.Name,
			Description:     rule.Description,
			MetricName:      rule.MetricName,
			Condition:       rule.Condition,
			Threshold:       rule.Threshold,
			Severity:        rule.Severity,
			Enabled:         rule.Enabled,
			CooldownSeconds: rule.CooldownSeconds,
		}
		alertResponseList[i] = ruleResponse
	}

	// Return AlertListResponse with count
	s.writeJSON(w, http.StatusOK, AlertListResponse{
		Alerts: alertResponseList,
		Count:  len(alertResponseList),
	})
}

// Validate validates a CreateAlertRequest
// Returns: error if invalid
func (r *CreateAlertRequest) Validate() error {
	// Name, MetricName, Condition, Severity validation
	// Similar to config.AlertRule.Validate
	if r.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	if r.MetricName == "" {
		return fmt.Errorf("rule metric name cannot be empty")
	}
	if r.Condition == "" {
		return fmt.Errorf("rule condition name cannot be empty")
	}
	if r.Severity == "" {
		return fmt.Errorf("rule severity name cannot be empty")
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

// handleCreateAlert creates a new alert rule
// POST /api/v1/alerts
// Request body: CreateAlertRequest
// Returns: AlertRuleResponse with created rule
func (s *Server) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] POST /api/v1/alerts")
	// Parse JSON body into CreateAlertRequest
	createRequest := CreateAlertRequest{}
	err := json.NewDecoder(r.Body).Decode(&createRequest)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate required fields (MetricName, Condition, Threshold, Severity)
	err = createRequest.Validate()
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Generate unique ID (UUID or sequential)
	newUUID := uuid.New().String()
	fmt.Printf("Generated UUID for new rule: %s\n", newUUID)

	// Convert to alerting.AlertRule
	rule := alerting.AlertRule{
		ID:              newUUID,
		Name:            createRequest.Name,
		Description:     createRequest.Description,
		MetricName:      createRequest.MetricName,
		Condition:       createRequest.Condition,
		Threshold:       createRequest.Threshold,
		Enabled:         true,
		Severity:        createRequest.Severity,
		CooldownSeconds: createRequest.CooldownSeconds,
	}

	err = s.alertManager.AddRule(rule)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}

	// Return created rule as AlertRuleResponse with 201 status
	response := AlertRuleResponse{
		ID:              rule.ID,
		Name:            rule.Name,
		Description:     rule.Description,
		MetricName:      rule.MetricName,
		Condition:       rule.Condition,
		Threshold:       rule.Threshold,
		Severity:        rule.Severity,
		Enabled:         rule.Enabled,
		CooldownSeconds: rule.CooldownSeconds,
	}
	s.writeJSON(w, http.StatusCreated, response)
}

// handleDeleteAlert deletes an alert rule
// DELETE /api/v1/alerts/{id}
// Query parameter: id - the alert rule ID to delete
// Returns: 204 No Content on success
func (s *Server) handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] DELETE /api/v1/alerts")
	// Extract rule ID from URL query parameter
	ruleID := r.URL.Query().Get("id")

	// Validate ID is not empty
	if ruleID == "" {
		s.writeError(w, http.StatusBadRequest, "rule ID query parameter is required")
		return
	}

	// Remove rule, handle ErrRuleNotFound (404)
	err := s.alertManager.RemoveRule(ruleID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}
