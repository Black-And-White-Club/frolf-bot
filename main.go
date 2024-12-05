package main

import (
	"context"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app" // Import your app package
	"github.com/Black-And-White-Club/tcr-bot/app/handlers"
)

func main() {
	ctx := context.Background()
	app, userService, err := app.NewApp(ctx) // Get userService from NewApp
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	handlers.UserService = userService // Assign userService to handlers package

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
