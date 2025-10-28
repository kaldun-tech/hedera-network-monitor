package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
)

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

// Start starts the HTTP server and blocks until context is cancelled
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/api/v1/metrics/account", s.handleMetricsByLabel)
	// TODO: Add more handlers:
	// - POST /api/v1/alerts - create alert rule
	// - GET /api/v1/alerts - list alert rules
	// - DELETE /api/v1/alerts/{id} - delete alert rule
	// - GET /api/v1/storage/stats - storage statistics
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
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","version":"0.1.0"}`)
}

// handleMetrics returns metrics based on query parameters
// Query parameters:
//   - name: metric name filter
//   - limit: maximum number of results
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement metrics retrieval
	// - Parse query parameters
	// - Call storage.GetMetrics()
	// - Return JSON response
	// - Handle errors properly
	// - Add pagination support

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"metrics":[],"count":0}`)
}

// handleMetricsByLabel returns metrics filtered by label
// Query parameters:
//   - key: label key
//   - value: label value
func (s *Server) handleMetricsByLabel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement metrics by label retrieval
	// - Extract key and value from query parameters
	// - Call storage.GetMetricsByLabel()
	// - Return JSON response
	// - Handle errors properly

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"metrics":[],"count":0}`)
}
