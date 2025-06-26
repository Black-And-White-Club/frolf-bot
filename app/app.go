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

	app.Helpers = utils.NewHelper(app.Observability.Provider.Logger)

	logger.Info("App Initialize finished")
	return nil
}

// initializeModules initializes the application modules.
func (app *App) initializeModules(ctx context.Context, routerRunCtx context.Context) error {
	var err error

	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}

	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.LeaderboardDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}

	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}

	if app.ScoreModule, err = score.NewScoreModule(ctx, app.Config, app.Observability, app.DB.ScoreDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize score module", attr.Error(err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}

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

	routerRunCtx, routerRunCancel := context.WithCancel(ctx)
	defer routerRunCancel()

	if err := app.initializeModules(ctx, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed during module initialization", attr.Error(err))
		cancel()
		return fmt.Errorf("failed during module initialization: %w", err)
	}

	go func() {
		if err := app.Router.Run(routerRunCtx); err != nil && err != context.Canceled {
			app.Observability.Provider.Logger.Error("Error running Watermill router", attr.Error(err))
			cancel()
		}
	}()

	select {
	case <-app.Router.Running():
		app.Observability.Provider.Logger.Info("Main router started and running")
	case <-time.After(5 * time.Second):
		app.Observability.Provider.Logger.Error("Timeout waiting for main router to start")
		cancel()
		return fmt.Errorf("timeout waiting for main router to start")
	}

	<-ctx.Done()

	app.Observability.Provider.Logger.Info("Shutting down...")
	if err := app.Close(); err != nil {
		return err
	}

	app.Observability.Provider.Logger.Info("Graceful shutdown complete.")
	return nil
}

func (app *App) Close() error {
	// Modules' Close methods should handle closing their internal components
	if app.UserModule != nil {
		app.UserModule.Close()
	}
	if app.LeaderboardModule != nil {
		app.LeaderboardModule.Close()
	}
	if app.RoundModule != nil {
		app.RoundModule.Close()
	}
	if app.ScoreModule != nil {
		app.ScoreModule.Close()
	}

	if app.EventBus != nil {
		if err := app.EventBus.Close(); err != nil {
			app.Observability.Provider.Logger.Error("Error closing event bus", attr.Error(err))
		}
	}

	return nil
}
