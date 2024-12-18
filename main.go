package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Black-And-White-Club/tcr-bot/app"
)

func main() {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize the application
	application := &app.App{}
	if err := application.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer application.Close()

	// Start the application
	if err := application.Run(ctx); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	fmt.Println("Waiting for shutdown signal...")
	select {
	case <-interrupt:
		fmt.Println("Shutting down application...")
	case <-ctx.Done():
		fmt.Println("Application context canceled")
	}

	// Graceful shutdown is handled by the deferred application.Close()
	log.Println("Graceful shutdown complete.")
}
