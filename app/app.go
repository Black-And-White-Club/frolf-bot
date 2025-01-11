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

	events "github.com/Black-And-White-Club/tcr-bot/app/eventbus"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// App holds the application components.
type App struct {
	Config     *config.Config
	Logger     *slog.Logger
	Router     *message.Router
	UserModule *user.Module
	DB         *bundb.DBService
	EventBus   shared.EventBus
}

// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context) error {
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	fmt.Printf("Loaded Config: %+v\n", cfg)
	app.Config = cfg

	// Use slog for logging with Debug level
	app.Logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Set log level to Debug
	}))

	app.Logger.Info("App Initialize started") // Log initialization start

	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize EventBus
	app.EventBus, err = events.NewEventBus(ctx, cfg.NATS.URL, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	// CREATE STREAMS HERE - ONLY ONCE
	streams := map[string]string{
		userstream.UserSignupRequestStreamName:      "user.signup.request",
		userstream.UserSignupResponseStreamName:     "user.signup.response",
		userstream.UserRoleUpdateRequestStreamName:  "user.role.update.request",
		userstream.UserRoleUpdateResponseStreamName: "user.role.update.response",
		userstream.LeaderboardStreamName:            "leaderboard.>",
	}

	for streamName, subject := range streams {
		if err := app.EventBus.CreateStream(ctx, streamName, subject); err != nil {
			return fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
		app.Logger.Info("Stream created", "stream_name", streamName, "subject", subject)
	}

	app.Logger.Info("All streams created")

	// Set Watermill's logger to use slog
	watermillLogger := watermill.NewSlogLogger(app.Logger)

	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}

	app.Router = router

	// Initialize User Module
	userModule, err := user.NewUserModule(ctx, cfg, app.Logger, app.DB.UserDB, app.EventBus)
	if err != nil {
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule

	// Wait for user module subscribers to be ready
	<-userModule.SubscribersReady

	app.Logger.Info("User module initialized in App Initialize")

	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// Signal handling
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		app.Logger.Info("Interrupt signal received, shutting down...")
		cancel() // Cancel the main context
	}()

	// START ROUTER (AND SUBSCRIBERS) *AFTER* INITIALIZATION
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.Router.Run(ctx); err != nil {
			app.Logger.Error("Error running Watermill router", slog.Any("error", err))
			cancel()
		}
	}()

	// Initialization timeout
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 10*time.Second)
		defer timeoutCancel()

		for {
			select {
			case <-timeoutCtx.Done():
				app.Logger.Error("Timeout waiting for subscribers to initialize")
				cancel()
				return
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				if app.UserModule != nil && app.UserModule.IsInitialized() {
					app.Logger.Info("User module initialized")
					return
				}
			}
		}
	}()

	wg.Wait()

	app.Close()
	app.Logger.Info("Graceful shutdown complete.")

	return nil
}

// Close gracefully shuts down the application.
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
