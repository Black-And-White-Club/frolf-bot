package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot/app"
	"github.com/Black-And-White-Club/frolf-bot/config"
)

func main() {
	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse config first
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize observability
	obsConfig := observability.Config{
		LokiURL:         cfg.Observability.LokiURL,
		LokiTenantID:    cfg.Observability.LokiTenantID,
		ServiceName:     "frolf-bot",
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		ServiceVersion:  "1.0.0", // Consider adding this to your build process
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	obs, err := observability.NewObservability(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	// Start observability components
	if err := obs.Start(ctx); err != nil {
		fmt.Printf("Failed to start observability: %v\n", err)
		os.Exit(1)
	}

	// Improved shutdown with timeout context
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := obs.Stop(shutdownCtx); err != nil {
			fmt.Printf("Error shutting down observability: %v\n", err)
		}
	}()

	// Get logger from observability
	logger := obs.GetLogger()

	// Initialize application with observability
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, obs); err != nil {
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}

	// Handle graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-interrupt:
			logger.Info("Received interrupt signal. Shutting down...")
		case <-ctx.Done():
			logger.Info("Application context cancelled. Shutting down...")
		}

		// Start graceful shutdown
		cancel()

		// Set a timeout for shutdown
		shutdownTimer := time.NewTimer(10 * time.Second)
		defer shutdownTimer.Stop()
		select {
		case <-shutdownTimer.C:
			logger.Info("Graceful shutdown timeout reached. Exiting.")
			os.Exit(1)
		case <-ctx.Done():
			logger.Info("Graceful shutdown complete.")
		}
	}()

	go func() {
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Application run failed", attr.Error(err))
			os.Exit(1)
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	<-ctx.Done()

	application.Close()
	logger.Info("Application closed successfully.")
}
