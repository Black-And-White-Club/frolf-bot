package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
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
	Observability     observability.Observability // Use the unified interface
	Router            *message.Router
	UserModule        *user.Module
	LeaderboardModule *leaderboard.Module
	RoundModule       *round.Module
	ScoreModule       *score.Module
	DB                *bundb.DBService
	EventBus          eventbus.EventBus
	Helpers           utils.Helpers
}

func (app *App) GetHealthCheckers() []eventbus.HealthChecker {
	if app.EventBus == nil {
		return nil
	}
	return app.EventBus.GetHealthCheckers()
}

// Initialize initializes the application.
func (app *App) Initialize(ctx context.Context, cfg *config.Config, obs observability.Observability) error {
	app.Config = cfg
	app.Observability = obs

	logger := app.Observability.GetLogger()
	logger.Info("App Initialize started")
	app.Config = cfg

	// Create a new observability service
	obs, err := observability.NewObservability(ctx, observability.Config{
		LokiURL:         cfg.Observability.LokiURL,
		LokiTenantID:    cfg.Observability.LokiTenantID,
		ServiceName:     "frolf-bot",
		MetricsAddress:  cfg.Observability.MetricsAddress,
		TempoEndpoint:   cfg.Observability.TempoEndpoint,
		TempoInsecure:   cfg.Observability.TempoInsecure,
		TempoSampleRate: cfg.Observability.TempoSampleRate,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}
	app.Observability = obs

	// Initialize database
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create Watermill logger adapter
	watermillLogger := lokifrolfbot.ToWatermillAdapter(logger)

	// Create Router
	app.Router, err = message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}

	// Add Middleware
	app.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		lokifrolfbot.LokiLoggingMiddleware(watermillLogger),
	)

	// Initialize EventBus
	app.EventBus, err = eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend", obs.GetMetrics().EventBusMetrics(), obs.GetTracer())
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	// Initialize modules
	if err := app.initializeModules(ctx); err != nil {
		return fmt.Errorf("failed to initialize modules: %w", err)
	}

	logger.Info("All modules initialized successfully")
	return nil
}

func (app *App) initializeModules(ctx context.Context) error {
	logger := app.Observability.GetLogger()
	logger.Info("Entering initializeModules")

	// Initialize User Module
	var err error
	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	logger.Info("User  module initialized successfully")

	// Initialize Leaderboard Module
	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.LeaderboardDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	logger.Info("Leaderboard module initialized successfully")

	// Initialize Round Module
	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("❌ Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	logger.Info("✅ Round module initialized successfully")

	// Initialize Score Module
	if app.ScoreModule, err = score.NewScoreModule(ctx, app.Config, app.Observability, app.DB.ScoreDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("Failed to initialize score module", attr.Error(err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
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
		app.Observability.GetLogger().Info("Interrupt signal received, shutting down...")
		cancel()
	}()

	// Start the Watermill router in a separate goroutine
	go func() {
		app.Observability.GetLogger().Info("Starting main Watermill router in goroutine")
		if err := app.Router.Run(ctx); err != nil {
			app.Observability.GetLogger().Error("Error running Watermill router", attr.Error(err))
			cancel() // Signal other goroutines to stop
		}
		app.Observability.GetLogger().Info("Main Watermill router stopped")
	}()

	// Wait for the main router to start running
	app.Observability.GetLogger().Info("Waiting for main router to start running")
	select {
	case <-app.Router.Running():
		app.Observability.GetLogger().Info("Main router started and running")
	case <-time.After(time.Second * 5): // Increased timeout
		app.Observability.GetLogger().Error("Timeout waiting for main router to start")
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
	app.Observability.GetLogger().Info("Shutting down...")
	if err := app.Close(); err != nil {
		return err
	}

	app.Observability.GetLogger().Info("Graceful shutdown complete.")
	return nil
}

func (app *App) Close() error {
	app.Observability.GetLogger().Info("Starting app.Close()")

	// Close modules first
	if app.UserModule != nil {
		app.Observability.GetLogger().Info("Closing user module")
		app.UserModule.Close()
	}

	if app.LeaderboardModule != nil {
		app.Observability.GetLogger().Info("Closing leaderboard module")
		app.LeaderboardModule.Close()
	}

	if app.RoundModule != nil {
		app.Observability.GetLogger().Info("Closing round module")
		app.RoundModule.Close()
	}

	if app.ScoreModule != nil {
		app.Observability.GetLogger().Info("Closing score module")
		app.ScoreModule.Close()
	}

	// Then close the Watermill router
	if app.Router != nil {
		app.Observability.GetLogger().Info("Closing Watermill router")
		if err := app.Router.Close(); err != nil {
			app.Observability.GetLogger().Error("Error closing Watermill router", attr.Error(err))
		}
	}

	// Finally, close the event bus
	if app.EventBus != nil {
		app.Observability.GetLogger().Info("Closing event bus")
		if err := app.EventBus.Close(); err != nil {
			app.Observability.GetLogger().Error("Error closing event bus", attr.Error(err))
		}
	}
	app.Observability.GetLogger().Info("Finished app.Close()")
	return nil
}
