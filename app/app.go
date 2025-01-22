package app

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/eventbus"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/score"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// App holds the application components.
type App struct {
	Config            *config.Config
	Logger            *slog.Logger
	Router            *message.Router
	UserModule        *user.Module
	LeaderboardModule *leaderboard.Module
	RoundModule       *round.Module
	ScoreModule       *score.Module
	DB                *bundb.DBService
	EventBus          shared.EventBus
	js                jetstream.JetStream // New JetStream context
}

// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context) error {
	// Parse config file
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	app.Config = cfg

	// Initialize logger
	app.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	app.Logger.Info("App Initialize started")

	// Initialize database
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Connect to NATS
	nc, err := nats.Connect(app.Config.NATS.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	defer nc.Close()

	// Get JetStream context using the new JetStream API
	app.js, err = jetstream.Connect(nc)
	if err != nil {
		return fmt.Errorf("failed to connect to JetStream: %w", err)
	}

	// Create streams (using new JetStream API)
	if err := app.createStreams(ctx); err != nil {
		return fmt.Errorf("failed to create streams: %w", err)
	}

	// Initialize EventBus
	app.EventBus, err = eventbus.NewEventBus(ctx, cfg.Nats.URL, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	// Initialize User Module
	userModule, err := user.NewUserModule(ctx, cfg, app.Logger, app.DB.UserDB, app.EventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule

	// Initialize Leaderboard Module
	leaderboardModule, err := leaderboard.NewLeaderboardModule(ctx, cfg, app.Logger, app.DB.LeaderboardDB, app.EventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	app.LeaderboardModule = leaderboardModule

	// Initialize Round Module
	roundModule, err := round.NewRoundModule(ctx, cfg, app.Logger, app.DB.RoundDB, app.EventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	app.RoundModule = roundModule

	// Initialize Score Module
	scoreModule, err := score.NewScoreModule(ctx, cfg, app.Logger, app.DB.ScoreDB, app.EventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
	app.ScoreModule = scoreModule

	// Wait for subscribers to be ready
	<-userModule.SubscribersReady
	<-leaderboardModule.SubscribersReady
	<-roundModule.SubscribersReady
	<-scoreModule.SubscribersReady

	// Add a delay here
	time.Sleep(100 * time.Millisecond)

	app.Logger.Info("All modules initialized successfully")

	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Handle OS signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		app.Logger.Info("Interrupt signal received, shutting down...")
		cancel()
	}()

	// Start the router
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.Router.Run(ctx); err != nil {
			app.Logger.Error("Error running Watermill router", slog.Any("error", err))
			cancel()
		}
	}()

	// Wait for graceful shutdown
	wg.Wait()

	app.Close()
	app.Logger.Info("Graceful shutdown complete.")

	return nil
}

// Close shuts down all resources.
func (app *App) Close() {
	if app.Router != nil {
		if err := app.Router.Close(); err != nil {
			app.Logger.Error("Error closing Watermill router", slog.Any("error", err))
		}
	}

	if app.EventBus != nil {
		if err := app.EventBus.Close(); err != nil {
			app.Logger.Error("Error closing event bus", slog.Any("error", err))
		}
	}
}
