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
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
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

	logger := obs.Provider.Logger
	logger.Info("App Initialize started")

	// Initialize database
	var err error
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create Watermill logger adapter
	watermillLogger := loggerfrolfbot.ToWatermillAdapter(logger)

	// Create Router
	app.Router, err = message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}

	// Add Middleware
	app.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(obs.Registry.Tracer),
	)

	// Initialize EventBus
	tracer := obs.Provider.TracerProvider.Tracer("eventbus")

	app.EventBus, err = eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend", obs.Registry.EventBusMetrics, tracer)
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
	logger := app.Observability.Provider.Logger
	logger.Info("Entering initializeModules")

	var err error

	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	logger.Info("User module initialized successfully")

	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.LeaderboardDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	logger.Info("Leaderboard module initialized successfully")

	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.EventBus, app.Router, app.Helpers); err != nil {
		logger.Error("❌ Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	logger.Info("✅ Round module initialized successfully")

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

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interrupt
		app.Observability.Provider.Logger.Info("Interrupt signal received, shutting down...")
		cancel()
	}()

	go func() {
		app.Observability.Provider.Logger.Info("Starting main Watermill router in goroutine")
		if err := app.Router.Run(ctx); err != nil {
			app.Observability.Provider.Logger.Error("Error running Watermill router", attr.Error(err))
			cancel()
		}
		app.Observability.Provider.Logger.Info("Main Watermill router stopped")
	}()

	app.Observability.Provider.Logger.Info("Waiting for main router to start running")
	select {
	case <-app.Router.Running():
		app.Observability.Provider.Logger.Info("Main router started and running")
	case <-time.After(5 * time.Second):
		app.Observability.Provider.Logger.Error("Timeout waiting for main router to start")
		cancel()
		return fmt.Errorf("timeout waiting for main router to start")
	}

	app.UserModule.Run(ctx, nil)
	app.LeaderboardModule.Run(ctx, nil)
	app.RoundModule.Run(ctx, nil)
	app.ScoreModule.Run(ctx, nil)

	<-ctx.Done()

	app.Observability.Provider.Logger.Info("Shutting down...")
	if err := app.Close(); err != nil {
		return err
	}

	app.Observability.Provider.Logger.Info("Graceful shutdown complete.")
	return nil
}

func (app *App) Close() error {
	app.Observability.Provider.Logger.Info("Starting app.Close()")

	if app.UserModule != nil {
		app.Observability.Provider.Logger.Info("Closing user module")
		app.UserModule.Close()
	}
	if app.LeaderboardModule != nil {
		app.Observability.Provider.Logger.Info("Closing leaderboard module")
		app.LeaderboardModule.Close()
	}
	if app.RoundModule != nil {
		app.Observability.Provider.Logger.Info("Closing round module")
		app.RoundModule.Close()
	}
	if app.ScoreModule != nil {
		app.Observability.Provider.Logger.Info("Closing score module")
		app.ScoreModule.Close()
	}

	if app.Router != nil {
		app.Observability.Provider.Logger.Info("Closing Watermill router")
		if err := app.Router.Close(); err != nil {
			app.Observability.Provider.Logger.Error("Error closing Watermill router", attr.Error(err))
		}
	}

	if app.EventBus != nil {
		app.Observability.Provider.Logger.Info("Closing event bus")
		if err := app.EventBus.Close(); err != nil {
			app.Observability.Provider.Logger.Error("Error closing event bus", attr.Error(err))
		}
	}

	app.Observability.Provider.Logger.Info("Finished app.Close()")
	return nil
}
