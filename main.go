package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Assuming app and config are in the correct paths relative to this main package
	"github.com/Black-And-White-Club/frolf-bot/app"
	"github.com/Black-And-White-Club/frolf-bot/config"

	// Import the updated observability package
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr" // Keep using attr for structured logging if desired
)

func main() {
	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Configuration Loading ---
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		// Use standard fmt here as logger isn't initialized yet
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// --- Observability Initialization ---
	// Map your application config to the new observability config struct
	obsConfig := observability.Config{
		ServiceName:     "frolf-bot", // Or get from cfg if available
		Environment:     cfg.Observability.Environment,
		Version:         "1.0.0",
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	// Initialize the new observability stack (Provider + Registry)
	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		// Use standard fmt as the logger might have failed during init
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	// Get logger from the observability provider
	logger := obs.Provider.Logger // Access logger via Provider

	logger.Info("Observability initialized successfully")

	// --- Application Initialization ---
	// Initialize application, passing the new observability struct
	// Ensure app.Initialize signature matches (accepts *observability.Observability)
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, *obs); err != nil { // Pass the new obs struct
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}
	logger.Info("Application initialized successfully")

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{}) // Channel to signal clean shutdown completion

	// Goroutine to handle signals and initiate shutdown
	go func() {
		select {
		case sig := <-interrupt:
			logger.Info("Received signal", attr.String("signal", sig.String()))
		case <-ctx.Done():
			logger.Info("Application context cancelled")
		}

		logger.Info("Initiating graceful shutdown...")
		cancel() // Cancel the main context to signal all components

		// Create a timeout for the entire shutdown process
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second) // Increased timeout
		defer shutdownCancel()

		// Shutdown application first
		go func() {
			defer close(cleanShutdown) // Signal that cleanup is done
			logger.Info("Closing application...")
			if err := application.Close(); err != nil {
				logger.Error("Error during application shutdown", attr.Error(err))
			} else {
				logger.Info("Application closed successfully")
			}

			// Shutdown observability components (provider handles internal shutdowns)
			logger.Info("Shutting down observability components...")
			obsShutdownCtx, obsShutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
			defer obsShutdownCancel()
			if err := obs.Provider.Shutdown(obsShutdownCtx); err != nil { // Use Provider.Shutdown
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
				os.Exit(1) // Force exit if shutdown takes too long
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
			cancel() // Signal shutdown on unexpected error
		} else {
			logger.Info("Application run loop finished")
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	// --- Wait for Shutdown Signal ---
	<-ctx.Done() // Wait until context is cancelled (either by signal or error)

	// --- Final Wait for Cleanup ---
	// Wait a bit longer to ensure the shutdown goroutine finishes logging etc.
	select {
	case <-cleanShutdown:
		// Shutdown completed cleanly, already logged.
	case <-time.After(5 * time.Second): // Short extra wait after cleanShutdown should have closed
		logger.Warn("Did not receive clean shutdown signal within final wait period.")
	}

	logger.Info("Exiting main.")
	os.Exit(0) // Optional: Explicitly exit with code 0
}
