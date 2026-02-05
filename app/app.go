package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/club"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/guild"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/score"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/user"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot/db/bundb"
	"github.com/ThreeDotsLabs/watermill/message"
	wm_middleware "github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/go-chi/chi/v5"
	chi_middleware "github.com/go-chi/chi/v5/middleware"
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
	ClubModule        *club.Module
	AuthModule        *auth.Module
	DB                *bundb.DBService
	EventBus          eventbus.EventBus
	Helpers           utils.Helpers
	HTTPRouter        *chi.Mux
	HTTPServer        *http.Server
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
	// Initialize HTTP Router
	app.HTTPRouter = chi.NewRouter()
	app.HTTPRouter.Use(chi_middleware.Logger)
	app.HTTPRouter.Use(chi_middleware.Recoverer)
	app.HTTPRouter.Use(chi_middleware.RealIP)
	app.HTTPRouter.Use(SecurityHeaders)
	// Add CORS if needed (PWA and Backend might be on different subdomains)
	// app.HTTPRouter.Use(cors.Handler(cors.Options{...}))

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
		wm_middleware.CorrelationID,
		wm_middleware.Recoverer,
		tracingfrolfbot.TraceHandler(obs.Registry.Tracer),
	)

	// Initialize EventBus
	fmt.Printf("DEBUG: Initializing EventBus with NATS URL: %s\n", app.Config.NATS.URL)
	eventBusTracer := obs.Provider.TracerProvider.Tracer("eventbus")

	app.EventBus, err = eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend", obs.Registry.EventBusMetrics, eventBusTracer)
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
	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	fmt.Println("DEBUG: User module initialized successfully")

	fmt.Println("DEBUG: Initializing leaderboard module...")
	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.GetDB(), app.DB.LeaderboardDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.EventBus.GetJetStream(), app.UserModule.UserService); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	fmt.Println("DEBUG: Leaderboard module initialized successfully")

	fmt.Println("DEBUG: Initializing round module...")
	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.DB.GetDB(), app.DB.UserDB, app.UserModule.UserService, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	fmt.Println("DEBUG: Round module initialized successfully")

	fmt.Println("DEBUG: Initializing guild module...")
	if app.GuildModule, err = guild.NewGuildModule(ctx, app.Config, app.Observability, app.DB.GuildDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize guild module", attr.Error(err))
		return fmt.Errorf("failed to initialize guild module: %w", err)
	}
	fmt.Println("DEBUG: Guild module initialized successfully")

	fmt.Println("DEBUG: Initializing score module...")
	if app.ScoreModule, err = score.NewScoreModule(ctx, app.Config, app.Observability, app.DB.ScoreDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize score module", attr.Error(err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
	fmt.Println("DEBUG: Score module initialized successfully")

	fmt.Println("DEBUG: Initializing club module...")
	if app.ClubModule, err = club.NewClubModule(ctx, app.Observability, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize club module", attr.Error(err))
		return fmt.Errorf("failed to initialize club module: %w", err)
	}
	fmt.Println("DEBUG: Club module initialized successfully")

	// Initialize auth module (handles magic links and auth callout)
	fmt.Println("DEBUG: Initializing auth module...")
	if app.AuthModule, err = auth.NewModule(ctx, app.Config, app.Observability, app.EventBus.GetNATSConnection(), app.EventBus, app.Helpers, app.DB.UserDB, app.HTTPRouter); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize auth module", attr.Error(err))
		return fmt.Errorf("failed to initialize auth module: %w", err)
	}
	fmt.Println("DEBUG: Auth module initialized successfully")

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

	// Start Auth Module (runs its own NATS router)
	var wg sync.WaitGroup // Create a WaitGroup for modules that need it
	wg.Add(1)
	go app.AuthModule.Run(ctx, &wg)

	go func() {
		fmt.Println("DEBUG: Starting Watermill router...")
		fmt.Println("DEBUG: About to call app.Router.Run() - this should block until router is ready")
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

	// Start HTTP Server
	port := app.Config.HTTP.Port
	if port == "" {
		port = ":3001"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	app.HTTPServer = &http.Server{
		Addr:              port,
		Handler:           app.HTTPRouter,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		fmt.Printf("DEBUG: Starting HTTP server on %s\n", port)
		if err := app.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("FATAL: HTTP server failed: %v\n", err)
			app.Observability.Provider.Logger.Error("HTTP server failed", attr.Error(err))
			cancel()
		}
	}()

	fmt.Println("DEBUG: Waiting for router to signal it's running...")
	select {
	case <-app.Router.Running():
		fmt.Println("DEBUG: Router signaled it's running successfully!")
		app.Observability.Provider.Logger.Info("Main router started and running")
	case <-time.After(30 * time.Second): // Increased from 5 to 30 seconds
		fmt.Println("DEBUG: Timeout waiting for router to start - this is a critical error")
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
	// Shutdown HTTP Server
	if app.HTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.HTTPServer.Shutdown(ctx); err != nil {
			app.Observability.Provider.Logger.Error("Error during HTTP server shutdown", attr.Error(err))
		} else {
			app.Observability.Provider.Logger.Info("HTTP server shut down successfully")
		}
	}

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
	if app.ClubModule != nil {
		app.ClubModule.Close()
	}
	if app.AuthModule != nil {
		app.AuthModule.Close()
	}

	if app.EventBus != nil {
		if err := app.EventBus.Close(); err != nil {
			app.Observability.Provider.Logger.Error("Error closing event bus", attr.Error(err))
		}
	}

	return nil
}

// SecurityHeaders adds standard security headers to the response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none';")
		next.ServeHTTP(w, r)
	})
}
