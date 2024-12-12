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
	ctx := context.Background()
	app, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Start the Watermill router
	if err := app.WatermillRouter.Run(context.Background()); err != nil {
		log.Fatalf("Failed to start Watermill router: %v", err)
	}

	// Graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Waiting for shutdown signal...")
	<-interrupt

	fmt.Println("Shutting down application...")

	// Close the Watermill PubSub in the UserModule
	if err := app.Modules.UserModule.PubSub.Close(); err != nil {
		log.Printf("Failed to close Watermill PubSub in UserModule: %v", err)
	}

	// ... add similar checks for other modules as needed ...
}
