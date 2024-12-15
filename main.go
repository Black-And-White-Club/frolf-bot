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
	ctx, cancel := context.WithCancel(context.TODO()) // Use context.TODO()
	defer cancel()

	application, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Start the Watermill router
	if err := application.WatermillRouter.Run(ctx); err != nil {
		log.Fatalf("Failed to start Watermill router: %v", err)
	}

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

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
		// Consider handling the error more explicitly if needed
	}

	// Gracefully close database connections
	if err := application.DB().GetDB().Close(); err != nil {
		log.Println("Error closing database connection:", err)
	}

	fmt.Println("Application shut down gracefully.")
}
