package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// WaitForShutdown waits for a shutdown signal and gracefully stops the application.
func (app *App) WaitForShutdown(ctx context.Context) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	fmt.Println("Waiting for shutdown signal...")
	<-interrupt

	fmt.Println("Shutting down application...")

	// Close the Watermill PubSub
	if err := app.WatermillPubSub.Close(); err != nil {
		// Handle the error, e.g., log it
	}
}
