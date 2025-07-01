package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/guild"
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
	GuildModule       *guild.Module
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
	fmt.Println("DEBUG: App.Initialize started")
	app.Config = cfg
	app.Observability = obs

	logger := obs.Provider.Logger
	logger.Info("App Initialize started")

	// Initialize database
	fmt.Println("DEBUG: Initializing database...")
	var err error
	app.DB, err = bundb.NewBunDBService(ctx, cfg.Postgres)
	if err != nil {
		fmt.Printf("DEBUG: Database initialization failed: %v\n", err)
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	fmt.Println("DEBUG: Database initialized successfully")

	// Create Watermill logger adapter
	fmt.Println("DEBUG: Creating Watermill logger adapter...")
	watermillLogger := loggerfrolfbot.ToWatermillAdapter(logger)

	// Create Router
	fmt.Println("DEBUG: Creating Watermill router...")
	app.Router, err = message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		fmt.Printf("DEBUG: Router creation failed: %v\n", err)
		return fmt.Errorf("failed to create Watermill router: %w", err)
	}
	fmt.Println("DEBUG: Router created successfully")

	// Add Middleware
	fmt.Println("DEBUG: Adding middleware...")
	app.Router.AddMiddleware(
		middleware.CorrelationID,
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(obs.Registry.Tracer),
	)

	// Initialize EventBus
	fmt.Printf("DEBUG: Initializing EventBus with NATS URL: %s\n", app.Config.NATS.URL)
	tracer := obs.Provider.TracerProvider.Tracer("eventbus")

	app.EventBus, err = eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend", obs.Registry.EventBusMetrics, tracer)
	if err != nil {
		fmt.Printf("DEBUG: EventBus creation failed: %v\n", err)
		return fmt.Errorf("failed to create event bus: %w", err)
	}
	fmt.Println("DEBUG: EventBus initialized successfully")

	app.Helpers = utils.NewHelper(app.Observability.Provider.Logger)

	fmt.Println("DEBUG: App.Initialize finished successfully")
	logger.Info("App Initialize finished")
	return nil
}

// initializeModules initializes the application modules.
func (app *App) initializeModules(ctx context.Context, routerRunCtx context.Context) error {
	var err error

	fmt.Println("DEBUG: Starting module initialization...")

	fmt.Println("DEBUG: Initializing user module...")
	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	fmt.Println("DEBUG: User module initialized successfully")

	fmt.Println("DEBUG: Initializing leaderboard module...")
	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.LeaderboardDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	fmt.Println("DEBUG: Leaderboard module initialized successfully")

	fmt.Println("DEBUG: Initializing round module...")
	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	fmt.Println("DEBUG: Round module initialized successfully")

	fmt.Println("DEBUG: Initializing guild module...")
	if app.GuildModule, err = guild.NewGuildModule(ctx, app.Config, app.Observability, app.DB.GuildDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize guild module", attr.Error(err))
		return fmt.Errorf("failed to initialize guild module: %w", err)
	}
	fmt.Println("DEBUG: Guild module initialized successfully")

	fmt.Println("DEBUG: Initializing score module...")
	if app.ScoreModule, err = score.NewScoreModule(ctx, app.Config, app.Observability, app.DB.ScoreDB, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize score module", attr.Error(err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
	fmt.Println("DEBUG: Score module initialized successfully")

	fmt.Println("DEBUG: All modules initialized successfully")
	return nil
}

// Run starts the application.
func (app *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Println("DEBUG: Signal handler goroutine started")
		<-interrupt
		fmt.Println("DEBUG: Signal received in app.Run!")
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
		fmt.Println("DEBUG: Starting Watermill router...")
		if err := app.Router.Run(routerRunCtx); err != nil && err != context.Canceled {
			fmt.Printf("DEBUG: Watermill router failed with error: %v\n", err)
			app.Observability.Provider.Logger.Error("Error running Watermill router", attr.Error(err))

			// Check if it's a "stream not found" error - this is a FATAL error
			errorStr := fmt.Sprintf("%v", err)
			if strings.Contains(errorStr, "stream not found") {
				fmt.Println("FATAL: Required NATS JetStream stream is missing!")
				fmt.Printf("FATAL: Error details: %s\n", errorStr)
				fmt.Println("FATAL: The backend cannot function without all required NATS streams.")
				fmt.Println("FATAL: Please ensure all required streams are created before starting the backend.")
				fmt.Println("FATAL: This is a critical dependency - the app will exit now.")

				app.Observability.Provider.Logger.Error("FATAL: Required NATS JetStream stream is missing. The backend cannot function without it.",
					attr.String("error", errorStr),
					attr.String("solution", "Ensure all required NATS streams are created before starting the backend"))

				// This is a critical failure - we must exit immediately
				cancel()
				return
			}

			// For any other router error, also fail fast
			fmt.Printf("DEBUG: Router error is not stream-related, but still critical: %v\n", err)
			cancel()
		} else if err == context.Canceled {
			fmt.Println("DEBUG: Watermill router stopped due to context cancellation")
			app.Observability.Provider.Logger.Info("Watermill router stopped due to context cancellation")
		} else {
			// Router finished normally - this should NOT happen for long-running event-driven apps
			fmt.Println("DEBUG: Watermill router finished unexpectedly - this indicates a problem")
			app.Observability.Provider.Logger.Error("Watermill router finished unexpectedly. This should not happen for a long-running event-driven application.")
			cancel()
		}
	}()

	select {
	case <-app.Router.Running():
		app.Observability.Provider.Logger.Info("Main router started and running")
	case <-time.After(30 * time.Second): // Increased from 5 to 30 seconds
		app.Observability.Provider.Logger.Error("Timeout waiting for main router to start")
		cancel()
		return fmt.Errorf("timeout waiting for main router to start")
	}

	<-ctx.Done()
	app.Observability.Provider.Logger.Info("Context was cancelled, beginning shutdown...")
	fmt.Println("DEBUG: App.Run context cancelled, shutting down...")

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
	if app.GuildModule != nil {
		app.GuildModule.Close()
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
