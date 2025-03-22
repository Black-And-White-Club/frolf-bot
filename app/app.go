package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/score"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/user"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// App holds the application components.
type App struct {
	Config            *config.Config
	Observability     observability.Observability
	Logger            observability.Logger
	Router            *message.Router
	UserModule        *user.Module
	LeaderboardModule *leaderboard.Module
	RoundModule       *round.Module
	ScoreModule       *score.Module
	RouterReady       chan struct{}
	DB                *bundb.DBService
	EventBus          eventbus.EventBus
	ErrorReporter     errors.ErrorReporterInterface
	Helpers           utils.Helpers
}

// Initialize initializes the application.
// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context, cfg *config.Config, obs observability.Observability) error {
	app.Config = cfg
	app.Observability = obs
	app.Logger = obs.GetLogger() // Use logger from observability

	logger := app.Logger
	logger.Info("App Initialize started")

	// Initialize database with metrics and tracer
	var err error
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres, obs.GetMetrics(), obs.GetTracer())
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create Watermill logger adapter
	watermillLogger := observability.ToWatermillAdapter(logger)

	// Create Router
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}
	app.Router = router

	// Add Middleware
	app.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		observability.LokiLoggingMiddleware(watermillLogger), // Add our logging middleware
	)

	// Initialize EventBus with observability
	eventBus, err := eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend")
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}
	app.EventBus = eventBus

	// Initialize modules and register handlers
	if err := app.initializeModules(ctx, cfg, app.Observability, app.DB, app.EventBus, app.Router, app.Helpers); err != nil {
		return fmt.Errorf("failed to initialize modules: %w", err)
	}

	logger.Info("All modules initialized successfully")

	return nil
}

func (app *App) initializeModules(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	db *bundb.DBService,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
) error {
	logger := obs.GetLogger()
	logger.Info("Entering initializeModules")

	// Initialize User Module
	userModule, err := user.NewUserModule(ctx, cfg, logger, db.UserDB, eventBus, router, helpers)
	if err != nil {
		logger.Error("Failed to initialize user module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	app.UserModule = userModule
	logger.Info("User  module initialized successfully")

	// Initialize Leaderboard Module
	leaderboardModule, err := leaderboard.NewLeaderboardModule(ctx, cfg, logger, db.LeaderboardDB, eventBus, router, helpers)
	if err != nil {
		logger.Error("Failed to initialize leaderboard module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	app.LeaderboardModule = leaderboardModule
	logger.Info("Leaderboard module initialized successfully")

	// Initialize Round Module
	roundModule, err := round.NewRoundModule(ctx, cfg, obs, db.RoundDB, eventBus, router, helpers)
	if err != nil {
		logger.Error("❌ Failed to initialize round module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	app.RoundModule = roundModule
	logger.Info("✅ Round module initialized successfully")

	// Initialize Score Module
	scoreModule, err := score.NewScoreModule(ctx, cfg, logger, db.ScoreDB, eventBus, router)
	if err != nil {
		logger.Error("Failed to initialize score module", slog.Any("error", err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
	app.ScoreModule = scoreModule
	logger.Info("Score module initialized successfully")

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
	app.ScoreModule.Run(ctx, nil)

	// Keep the main goroutine alive until the context is canceled.
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

	app.Logger.Info("Closing score module")
	app.ScoreModule.Close()

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
