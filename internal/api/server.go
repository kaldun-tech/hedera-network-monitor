package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

// Server represents the HTTP API server
type Server struct {
	port   int
	store  storage.Storage
	server *http.Server
}

// NewServer creates a new API server
func NewServer(port int, store storage.Storage) *Server {
	return &Server{
		port:  port,
		store: store,
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
	// TODO: Add more handlers:
	// - POST /api/v1/alerts - create alert rule
	// - GET /api/v1/alerts - list alert rules
	// - DELETE /api/v1/alerts/{id} - delete alert rule
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
