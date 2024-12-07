// main.go

package main

import (
	"context"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app"
)

func main() {
	ctx := context.Background()
	app, err := app.NewApp(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize app: %v", err)
	}

	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
