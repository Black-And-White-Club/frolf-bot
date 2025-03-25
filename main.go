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

	// Initialize observability with expanded configuration
	obsConfig := observability.Config{
		LokiURL:         cfg.Observability.LokiURL,
		LokiTenantID:    cfg.Observability.LokiTenantID,
		ServiceName:     "frolf-bot",
		ServiceVersion:  "1.0.0", // Get this from build info or config
		Environment:     cfg.Observability.Environment,
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	}

	// Initialize observability service
	obs, err := observability.NewObservability(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	// Get logger from observability for cleaner error reporting
	logger := obs.GetLogger()

	// Initialize application with observability
	// Initialize application with observability
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, obs); err != nil {
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}

	// Register health checkers from the EventBus
	if application.EventBus != nil {
		healthCheckers := application.EventBus.GetHealthCheckers()
		if len(healthCheckers) > 0 {
			logger.Info("Registering EventBus health checkers", attr.Int("checker_count", len(healthCheckers)))

			for _, checker := range healthCheckers {
				obs.RegisterHealthChecker(checker)
				logger.Info("Registered health checker", attr.String("checker", checker.Name()))
			}
		} else {
			logger.Warn("No EventBus health checkers available")
		}
	} else {
		logger.Warn("EventBus not initialized, skipping health checker registration")
	}

	// Start observability components
	if err := obs.Start(ctx); err != nil {
		logger.Error("Failed to start observability", attr.Error(err))
		os.Exit(1)
	}

	// Improved shutdown with timeout context and better error handling
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := obs.Stop(shutdownCtx); err != nil {
			logger.Error("Error shutting down observability", attr.Error(err))
		} else {
			logger.Info("Observability shutdown successful")
		}
	}()

	// Handle graceful shutdown with proper logging
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Health check ticker
	healthCheckTicker := time.NewTicker(2 * time.Minute)
	defer healthCheckTicker.Stop()

	// Clean shutdown channel
	cleanShutdown := make(chan struct{})

	go func() {
		select {
		case sig := <-interrupt:
			logger.Info("Received signal", attr.String("signal", sig.String()))
			cancel() // Cancel the context to signal shutdown
		case <-ctx.Done():
			logger.Info("Application context cancelled")
		}

		logger.Info("Initiating graceful shutdown...")

		// Set a timeout for shutdown
		shutdownTimer := time.NewTimer(15 * time.Second)
		defer shutdownTimer.Stop()

		// Shutdown application first
		cleanup := func() {
			if err := application.Close(); err != nil {
				logger.Error("Error during application shutdown", attr.Error(err))
			} else {
				logger.Info("Application closed successfully")
			}
			close(cleanShutdown)
		}

		// Handle cleanup with timeout
		go cleanup()

		select {
		case <-shutdownTimer.C:
			logger.Error("Graceful shutdown timeout reached, forcing exit")
			os.Exit(1)
		case <-cleanShutdown:
			logger.Info("Graceful shutdown completed successfully")
		}
	}()

	// Health check goroutine
	go func() {
		for {
			select {
			case <-healthCheckTicker.C:
				if err := obs.HealthCheck(ctx); err != nil {
					logger.Warn("Health check failed", attr.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Run the application
	go func() {
		logger.Info("Starting application")
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Application run failed", attr.Error(err))
			cancel() // Signal shutdown on error
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	// Wait for context cancellation
	<-ctx.Done()

	// Wait for clean shutdown to complete
	select {
	case <-cleanShutdown:
		// Already logged in the shutdown goroutine
	case <-time.After(20 * time.Second):
		logger.Error("Final shutdown timeout reached, exiting")
	}
}
