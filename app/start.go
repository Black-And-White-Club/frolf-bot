// app/start.go

package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Black-And-White-Club/tcr-bot/round"
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

	// Create event handlers and pass the publisher
	roundEventHandler := round.NewRoundEventHandler(app.RoundService, app.Publisher())
	// leaderboardEventHandler := leaderboard.NewLeaderboardEventHandler(app.LeaderboardService, app.Publisher())
	// userEventHandler := user.NewUserEventHandler(app.UserService, app.Publisher())
	// scoreEventHandler := score.NewScoreEventHandler(app.ScoreService, app.Publisher())

	// Subscribe to events
	if err := app.RoundService.StartNATSSubscribers(ctx, roundEventHandler); err != nil {
		return fmt.Errorf("failed to start NATS subscribers for RoundService: %w", err)
	}
	// if err := app.LeaderboardService.StartNATSSubscribers(ctx, leaderboardEventHandler); err != nil {
	// 	return fmt.Errorf("failed to start NATS subscribers for LeaderboardService: %w", err)
	// }
	// if err := app.UserService.StartNATSSubscribers(ctx, userEventHandler); err != nil {
	// 	return fmt.Errorf("failed to start NATS subscribers for UserService: %w", err)
	// }
	// if err := app.ScoreService.StartNATSSubscribers(ctx, scoreEventHandler); err != nil {
	// 	return fmt.Errorf("failed to start NATS subscribers for ScoreService: %w", err)
	// }

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Wait for shutdown signal
	app.WaitForShutdown(ctx, srv)

	return nil
}
