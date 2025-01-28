package app

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/eventbus"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/round"
	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// App holds the application components.
type App struct {
	Config            *config.Config
	Logger            *slog.Logger
	Router            *message.Router
	UserModule        *user.Module
	LeaderboardModule *leaderboard.Module // Add Leaderboard module
	RoundModule       *round.Module       // Add Round module
	RouterReady       chan struct{}       // Channel to signal when the main router is ready
	DB                *bundb.DBService
	EventBus          shared.EventBus
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

	/// Initialize database
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create Watermill logger
	watermillLogger := watermill.NewSlogLogger(app.Logger)

	// Create a message deduplicator
	deduplicator := middleware.Deduplicator(middleware.Deduplicator{})

	// Create Router
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}
	app.Router = router

	// Add Middleware
	app.Router.AddMiddleware(
		middleware.CorrelationID, // Generate or propagate correlation ID
		deduplicator.Middleware,  // Deduplicate messages
		middleware.Recoverer,     // Recover from panics
	)

	// Initialize EventBus
	eventBus, err := eventbus.NewEventBus(ctx, app.Config.NATS.URL, app.Logger)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}
	app.EventBus = eventBus

	// Initialize modules and register handlers
	if err := app.initializeModules(ctx, cfg, app.Logger, app.DB, app.EventBus, app.Router); err != nil {
		return fmt.Errorf("failed to initialize modules: %w", err)
	}

	app.Logger.Info("All modules initialized successfully")

	return nil
}

func (app *App) initializeModules(ctx context.Context, cfg *config.Config, logger *slog.Logger, db *bundb.DBService, eventBus shared.EventBus, router *message.Router) error {
	logger.Info("Entering initializeModules")

	// Initialize User Module
	userModule, err := user.NewUserModule(ctx, cfg, logger, db.UserDB, eventBus, router)
	if err != nil {
		logger.Error("Failed to initialize user module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule
	logger.Info("User module initialized successfully")

	// Initialize Leaderboard Module
	leaderboardModule, err := leaderboard.NewLeaderboardModule(ctx, cfg, logger, db.LeaderboardDB, eventBus, router)
	if err != nil {
		logger.Error("Failed to initialize leaderboard module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	app.LeaderboardModule = leaderboardModule
	logger.Info("Leaderboard module initialized successfully")

	// Initialize Round Module
	roundModule, err := round.NewRoundModule(ctx, cfg, logger, db.RoundDB, eventBus, router)
	if err != nil {
		logger.Error("Failed to initialize round module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	app.RoundModule = roundModule
	logger.Info("Round module initialized successfully")

	logger.Info("Exiting initializeModules")
	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle OS signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-interrupt
		app.Logger.Info("Interrupt signal received, shutting down...")
		cancel()
	}()

	// Start the Watermill router in a separate goroutine
	go func() {
		app.Logger.Info("Starting main Watermill router in goroutine")
		if err := app.Router.Run(ctx); err != nil {
			app.Logger.Error("Error running Watermill router", slog.Any("error", err))
			cancel() // Signal other goroutines to stop
		}
		app.Logger.Info("Main Watermill router stopped")
	}()

	// Wait for the main router to start running
	app.Logger.Info("Waiting for main router to start running")
	select {
	case <-app.Router.Running():
		app.Logger.Info("Main router started and running")
	case <-time.After(time.Second * 5): // Increased timeout
		app.Logger.Error("Timeout waiting for main router to start")
		cancel()
		return fmt.Errorf("timeout waiting for main router to start")
	}

	// Start modules
	app.UserModule.Run(ctx, nil)
	app.LeaderboardModule.Run(ctx, nil)
	app.RoundModule.Run(ctx, nil)

	// Keep the main goroutine alive until the context is canceled.
	// This could be due to an interrupt signal or an error in the router.
	<-ctx.Done()

	// Perform a graceful shutdown
	app.Logger.Info("Shutting down...")
	if err := app.Close(); err != nil {
		return err
	}

	app.Logger.Info("Graceful shutdown complete.")
	return nil
}

func (app *App) Close() error {
	app.Logger.Info("Starting app.Close()")

	// Close modules first
	app.Logger.Info("Closing user module")
	app.UserModule.Close()

	app.Logger.Info("Closing leaderboard module")
	app.LeaderboardModule.Close()

	app.Logger.Info("Closing round module")
	app.RoundModule.Close()

	// Then close the Watermill router
	if app.Router != nil {
		app.Logger.Info("Closing Watermill router")
		if err := app.Router.Close(); err != nil {
			app.Logger.Error("Error closing Watermill router", slog.Any("error", err))
		}
	}

	// Finally, close the event bus
	if app.EventBus != nil {
		app.Logger.Info("Closing event bus")
		if err := app.EventBus.Close(); err != nil {
			app.Logger.Error("Error closing event bus", slog.Any("error", err))
		}
	}

	app.Logger.Info("Finished app.Close()")
	return nil
}
