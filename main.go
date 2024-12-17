package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app"
)

func main() {
	// Use context.Background() for the root context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add a timeout to the root context (e.g., 1 hour)
	ctx, cancel = context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()

	log.Println("Initializing application...") // More informative log message

	application, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %+v", err) // Include error details in log
	}

	log.Println("Application initialized successfully.") // More informative log message

	// Start the Watermill router
	log.Println("Starting Watermill router...") // More informative log message
	if err := application.WatermillRouter.Run(ctx); err != nil {
		log.Fatalf("Failed to start Watermill router: %+v", err) // Include error details in log
	}

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT) // Add SIGINT

	fmt.Println("Waiting for shutdown signal...")
	select {
	case <-interrupt:
		fmt.Println("Shutting down application...")
	case <-ctx.Done():
		fmt.Println("Application context canceled")
	}

	// Close the Watermill PubSub in the UserModule
	if err := application.Modules.UserModule.PubSub.Close(); err != nil {
		log.Printf("Failed to close Watermill PubSub in UserModule: %v", err)
		// TODO: Handle the error more explicitly if needed
	}

	// Gracefully close database connections
	if err := application.DB().GetDB().Close(); err != nil {
		log.Println("Error closing database connection:", err)
		// TODO: Handle the error more explicitly if needed
	}

	fmt.Println("Application shut down gracefully.")
}
