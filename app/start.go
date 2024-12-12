package app

import (
	"context"
	"fmt"
)

// Start starts the Watermill router.
func (app *App) Start(ctx context.Context) error {
	// Start the Watermill router
	if err := app.WatermillRouter.Run(context.Background()); err != nil {
		return fmt.Errorf("failed to start Watermill router: %w", err)
	}

	// Wait for shutdown signal
	app.WaitForShutdown(ctx) // Assuming WaitForShutdown is updated to not take *http.Server

	return nil
}
