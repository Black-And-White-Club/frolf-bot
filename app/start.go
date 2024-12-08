// app/start.go

package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
)

func (app *App) Start(ctx context.Context) error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	fmt.Println("Starting server on port", port)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: app.Router(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Wait for shutdown signal
	app.WaitForShutdown(ctx, srv)

	return nil
}
