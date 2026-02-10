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
	app.Config = cfg
	app.Observability = obs

	logger := obs.Provider.Logger
	logger.Info("App Initialize started")
	// Initialize HTTP Router
	app.HTTPRouter = chi.NewRouter()
	app.HTTPRouter.Use(chi_middleware.Logger)
	app.HTTPRouter.Use(chi_middleware.Recoverer)
	app.HTTPRouter.Use(chi_middleware.RealIP)
	app.HTTPRouter.Use(SecurityHeaders)

	// Health endpoint for Kubernetes probes (served by the same chi router)
	app.HTTPRouter.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

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
		wm_middleware.CorrelationID,
		wm_middleware.Recoverer,
		tracingfrolfbot.TraceHandler(obs.Registry.Tracer),
	)

	// Initialize EventBus
	eventBusTracer := obs.Provider.TracerProvider.Tracer("eventbus")

	app.EventBus, err = eventbus.NewEventBus(ctx, app.Config.NATS.URL, logger, "backend", obs.Registry.EventBusMetrics, eventBusTracer)
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
	if app.UserModule, err = user.NewUserModule(ctx, app.Config, app.Observability, app.DB.UserDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize user module", attr.Error(err))
		return fmt.Errorf("failed to initialize user module: %w", err)
	}
	if app.RoundModule, err = round.NewRoundModule(ctx, app.Config, app.Observability, app.DB.RoundDB, app.DB.GetDB(), app.DB.UserDB, app.UserModule.UserService, app.EventBus, app.Router, app.Helpers, routerRunCtx); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize round module", attr.Error(err))
		return fmt.Errorf("failed to initialize round module: %w", err)
	}
	if app.LeaderboardModule, err = leaderboard.NewLeaderboardModule(ctx, app.Config, app.Observability, app.DB.GetDB(), app.DB.LeaderboardDB, app.RoundModule.RoundService, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.EventBus.GetJetStream(), app.UserModule.UserService); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize leaderboard module", attr.Error(err))
		return fmt.Errorf("failed to initialize leaderboard module: %w", err)
	}
	if app.GuildModule, err = guild.NewGuildModule(ctx, app.Config, app.Observability, app.DB.GuildDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize guild module", attr.Error(err))
		return fmt.Errorf("failed to initialize guild module: %w", err)
	}
	if app.ScoreModule, err = score.NewScoreModule(ctx, app.Config, app.Observability, app.DB.ScoreDB, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize score module", attr.Error(err))
		return fmt.Errorf("failed to initialize score module: %w", err)
	}
	if app.ClubModule, err = club.NewClubModule(ctx, app.Observability, app.EventBus, app.Router, app.Helpers, routerRunCtx, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize club module", attr.Error(err))
		return fmt.Errorf("failed to initialize club module: %w", err)
	}

	// Initialize auth module (handles magic links and auth callout)
	if app.AuthModule, err = auth.NewModule(ctx, app.Config, app.Observability, app.EventBus.GetNATSConnection(), app.EventBus, app.Helpers, app.DB.UserDB, app.HTTPRouter, app.DB.GetDB()); err != nil {
		app.Observability.Provider.Logger.Error("Failed to initialize auth module", attr.Error(err))
		return fmt.Errorf("failed to initialize auth module: %w", err)
	}

	app.Observability.Provider.Logger.Info("All modules initialized successfully")
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

	// Start Auth Module (runs its own NATS router)
	var wg sync.WaitGroup // Create a WaitGroup for modules that need it
	wg.Add(1)
	go app.AuthModule.Run(ctx, &wg)

	go func() {
		if err := app.Router.Run(routerRunCtx); err != nil && err != context.Canceled {
			app.Observability.Provider.Logger.Error("Watermill router failed", attr.Error(err))

			errorStr := fmt.Sprintf("%v", err)
			if strings.Contains(errorStr, "stream not found") {
				app.Observability.Provider.Logger.Error("Required NATS JetStream stream is missing",
					attr.String("error", errorStr),
					attr.String("solution", "Ensure all required NATS streams are created before starting the backend"))
			}

			cancel()
		} else if err == context.Canceled {
			app.Observability.Provider.Logger.Info("Watermill router stopped due to context cancellation")
		} else {
			app.Observability.Provider.Logger.Error("Watermill router finished unexpectedly")
			cancel()
		}
	}()

	// Start HTTP Server
	port := app.Config.HTTP.Port
	if port == "" {
		port = ":8080"
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
		app.Observability.Provider.Logger.Info("Starting HTTP server", attr.String("port", port))
		if err := app.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Observability.Provider.Logger.Error("HTTP server failed", attr.Error(err))
			cancel()
		}
	}()

	select {
	case <-app.Router.Running():
		app.Observability.Provider.Logger.Info("Main router started and running")
	case <-time.After(30 * time.Second):
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none';")
		next.ServeHTTP(w, r)
	})
}
