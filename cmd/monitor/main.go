package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/api"
	"github.com/kaldun-tech/hedera-network-monitor/internal/collector"
	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	hederaClient, err := hedera.NewClient(cfg.Network.Name)
	if err != nil {
		log.Fatalf("Failed to create Hedera client: %v", err)
	}
	store := storage.NewMemoryStorage()
	alertManager := alerting.NewManager(cfg.Alerting)

	// Initialize collectors
	collectors := []collector.Collector{
		collector.NewAccountCollector(hederaClient, cfg.Accounts),
		collector.NewNetworkCollector(hederaClient),
	}

	// Initialize API server
	server := api.NewServer(cfg.API.Port, store, alertManager)

	// Run service in goroutine group with error handling
	eg, egCtx := errgroup.WithContext(ctx)

	// Start API server
	eg.Go(func() error {
		log.Printf("Starting API server on port %d", cfg.API.Port)
		return server.Start(egCtx)
	})

	// Start collectors
	for _, c := range collectors {
		// Capture collector in local variable to avoid closure issue
		coll := c
		eg.Go(func() error {
			log.Printf("Starting collector: %s", coll.Name())
			return coll.Collect(egCtx, store, alertManager)
		})
	}

	// Start alert manager
	eg.Go(func() error {
		log.Println("Starting alert manager")
		return alertManager.Run(egCtx)
	})

	// Wait for shutdown signal in a separate goroutine
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Initiating graceful shutdown...", sig)
		cancel()
	}()

	// Wait for all services to complete or error
	if err := eg.Wait(); err != nil {
		log.Printf("Service error: %v", err)
		os.Exit(1)
	}

	log.Println("Service shut down successfully")
}
