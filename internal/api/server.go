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
	// IMPLEMENTATION:
	// 1. Check if request method is GET (done below)
	// 2. Create HealthResponse struct
	// 3. Call s.writeJSON() with 200 status and response
	// 4. That's it! No storage access needed.

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	s.writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "healthy",
		Version: "0.1.0",
	})
}

// handleMetrics returns metrics based on query parameters
// GET /api/v1/metrics
// Query parameters:
//   - name: metric name filter (optional, empty string = all)
//   - limit: maximum number of results (optional, default 100, max 10000)
// Returns: MetricsResponse with metrics slice and count
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// IMPLEMENTATION STEPS:
	// 1. Check method is GET
	// 2. Parse query parameters:
	//    - Get "name" param with r.URL.Query().Get("name")
	//    - Get "limit" param with r.URL.Query().Get("limit")
	// 3. Parse limit as integer:
	//    - If empty or invalid, use default of 100
	//    - If > 10000, cap at 10000 (prevent abuse)
	// 4. Call s.store.GetMetrics(name, limit)
	// 5. Handle error case: log and return 500 with error message
	// 6. Handle nil metrics: convert to empty slice for JSON
	// 7. Create MetricsResponse with metrics slice and len(metrics)
	// 8. Call s.writeJSON() with 200 status and response

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Parse query parameters
	name := r.URL.Query().Get("name")
	limitStr := r.URL.Query().Get("limit")

	// Parse limit with sensible defaults
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Cap limit to prevent abuse
	if limit > 10000 {
		limit = 10000
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
// Returns: MetricsResponse with filtered metrics
func (s *Server) handleMetricsByLabel(w http.ResponseWriter, r *http.Request) {
	// IMPLEMENTATION STEPS:
	// 1. Check method is GET
	// 2. Extract key and value from query params
	// 3. Validate both are present (return 400 if not)
	// 4. Call s.store.GetMetricsByLabel(key, value)
	// 5. Handle error case: log and return 500
	// 6. Handle nil metrics: convert to empty slice
	// 7. Create MetricsResponse and call s.writeJSON()

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

	// Handle nil case
	if metrics == nil {
		metrics = []types.Metric{}
	}

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
	// IMPLEMENTATION STEPS:
	// 1. Check method is GET
	// 2. Type assert s.store to check if it has Stats() method
	//    Example: statsProvider, ok := s.store.(interface { Stats() (map[string]interface{}, error) })
	// 3. If not supported, return 501 NotImplemented
	// 4. Call statsProvider.Stats()
	// 5. Handle error: log and return 500
	// 6. Create StatsResponse from returned map (may need type conversion)
	// 7. Call s.writeJSON() with response

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "only GET allowed")
		return
	}

	// Type assert storage supports Stats
	statsProvider, ok := s.store.(interface {
		Stats() (map[string]interface{}, error)
	})

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

	s.writeJSON(w, http.StatusOK, stats)
}
