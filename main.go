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

	// --- Configuration Loading ---
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// --- Observability Initialization ---
	obsConfig := config.ToObsConfig(cfg)
	obsConfig.Version = "1.0.0"

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	logger := obs.Provider.Logger
	logger.Info("Observability initialized successfully")

	// --- Application Initialization ---
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, *obs); err != nil {
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}
	logger.Info("Application initialized successfully")

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{})

	// Goroutine to handle signals and initiate shutdown
	go func() {
		select {
		case sig := <-interrupt:
			logger.Info("Received signal", attr.String("signal", sig.String()))
		case <-ctx.Done():
			logger.Info("Application context cancelled")
		}

		logger.Info("Initiating graceful shutdown...")
		cancel()

		// Create a timeout for the entire shutdown process
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer shutdownCancel()

		// Shutdown application first
		go func() {
			defer close(cleanShutdown)
			logger.Info("Closing application...")
			if err := application.Close(); err != nil {
				logger.Error("Error during application shutdown", attr.Error(err))
			} else {
				logger.Info("Application closed successfully")
			}

			// Shutdown observability components
			logger.Info("Shutting down observability components...")
			obsShutdownCtx, obsShutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
			defer obsShutdownCancel()
			if err := obs.Provider.Shutdown(obsShutdownCtx); err != nil {
				logger.Error("Error shutting down observability", attr.Error(err))
			} else {
				logger.Info("Observability shutdown successful")
			}
		}()

		// Wait for cleanup or timeout
		select {
		case <-shutdownCtx.Done():
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Error("Graceful shutdown timeout reached, forcing exit")
				os.Exit(1)
			}
		case <-cleanShutdown:
			logger.Info("Graceful shutdown completed successfully")
		}
	}()

	// --- Run Application ---
	go func() {
		logger.Info("Starting application run loop")
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Application run failed", attr.Error(err))
			cancel()
		} else {
			logger.Info("Application run loop finished")
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	// --- Wait for Shutdown Signal ---
	<-ctx.Done()

	// --- Final Wait for Cleanup ---
	select {
	case <-cleanShutdown:
		// Shutdown completed cleanly
	case <-time.After(5 * time.Second):
		logger.Warn("Did not receive clean shutdown signal within final wait period.")
	}

	logger.Info("Exiting main.")
	os.Exit(0)
}
