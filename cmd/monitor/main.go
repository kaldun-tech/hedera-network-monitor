package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaldun-tech/hedera-network-monitor/internal/alerting"
	"github.com/kaldun-tech/hedera-network-monitor/internal/api"
	"github.com/kaldun-tech/hedera-network-monitor/internal/collector"
	"github.com/kaldun-tech/hedera-network-monitor/internal/storage"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/config"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/hedera"
	"github.com/kaldun-tech/hedera-network-monitor/pkg/logger"
	"golang.org/x/sync/errgroup"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		// Use default logger before config is loaded
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize logger based on configuration
	logLevel := logger.ParseLevel(cfg.Logging.Level)
	if cfg.Logging.Format == "json" {
		logger.InitJSON(logLevel, os.Stdout)
	} else {
		logger.Init(logLevel, os.Stdout)
	}

	logger.Info("Starting Hedera Network Monitor",
		"network", cfg.Network.Name,
		"log_level", cfg.Logging.Level,
		"log_format", cfg.Logging.Format)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize components
	hederaClient, err := hedera.NewClient(cfg.Network.Name, cfg.Network.OperatorID, cfg.Network.OperatorKey)
	if err != nil {
		logger.Error("Failed to create Hedera client", "error", err)
		os.Exit(1)
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
		logger.Info("Starting API server", "port", cfg.API.Port)
		return server.Start(egCtx)
	})

	// Start collectors
	for _, c := range collectors {
		// Capture collector in local variable to avoid closure issue
		coll := c
		eg.Go(func() error {
			logger.Info("Starting collector", "name", coll.Name())
			return coll.Collect(egCtx, store, alertManager)
		})
	}

	// Start alert manager
	eg.Go(func() error {
		logger.Info("Starting alert manager")
		return alertManager.Run(egCtx)
	})

	// Wait for shutdown signal in a separate goroutine
	go func() {
		sig := <-sigChan
		logger.Info("Received signal, initiating graceful shutdown", "signal", sig)
		cancel()
	}()

	// Wait for all services to complete or error
	if err := eg.Wait(); err != nil {
		logger.Error("Service error", "error", err)
		os.Exit(1)
	}

	logger.Info("Service shut down successfully")
}
