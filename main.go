package main

import (
	"context"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app"
	"github.com/Black-And-White-Club/tcr-bot/app/handlers"
)

func main() {
	ctx := context.Background()
	app, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	// Access services through the App struct
	handlers.UserService = app.UserService
	handlers.LeaderboardService = app.LeaderboardService
	// ... assign other services to the handlers package if needed ...

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
