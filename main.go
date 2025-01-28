package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot/app"
)

func main() {
	// Initialize the application
	// os.Setenv("WATERMILL_DEBUG", "1") // Enable Watermill debug logs
	application := &app.App{}

	// Initialize the logger before calling application.Initialize
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := application.Initialize(ctx); err != nil {
		logger.Error("Failed to initialize application", slog.Any("error", err))
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
		cancel()
		time.Sleep(5 * time.Second)
		logger.Info("Graceful shutdown timeout reached. Exiting.")
		os.Exit(1)
	}()

	go func() {
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Application run failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	fmt.Println("Application is running. Press Ctrl+C to gracefully shut down.")

	<-ctx.Done()

	application.Close()
	logger.Info("Graceful shutdown complete.")
}
