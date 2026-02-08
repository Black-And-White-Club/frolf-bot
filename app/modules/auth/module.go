package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	authhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/handlers"
	authjwt "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/jwt"
	authnats "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/nats"
	authrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/router"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/uptrace/bun"
)

// Module represents the unified auth module.
type Module struct {
	config        *config.Config
	observability observability.Observability
	service       authservice.Service
	handlers      authhandlers.Handlers
	router        *authrouter.Router
	cancelFunc    context.CancelFunc
	logger        *slog.Logger
}

// NewModule creates a new auth module.
func NewModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	nc *nats.Conn,
	eventBus eventbus.EventBus,
	helper utils.Helpers,
	userRepo userdb.Repository,
	httpRouter chi.Router,
	db *bun.DB,
) (*Module, error) {
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "Initializing auth module")

	// Create JWT provider
	jwtProvider := authjwt.NewProvider(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.Audience)

	// Create NATS JWT builder if auth callout is enabled
	var userJWTBuilder authnats.UserJWTBuilder
	if cfg.AuthCallout.Enabled {
		// For centralized auth callout, sign with the account key directly
		// The issuer key must match auth_callout.issuer in NATS config
		// See: https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_callout
		accountKey, err := nkeys.FromSeed([]byte(cfg.AuthCallout.IssuerNKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse account key: %w", err)
		}
		userJWTBuilder = authnats.NewUserJWTBuilder(accountKey, cfg.AuthCallout.IssuerNKey)
	}

	// Create service config
	serviceConfig := authservice.Config{
		PWABaseURL: cfg.PWA.BaseURL,
		DefaultTTL: cfg.JWT.DefaultTTL,
	}

	// Create service
	service := authservice.NewService(
		jwtProvider,
		userJWTBuilder,
		userRepo,
		serviceConfig,
		logger,
		tracer,
		db,
	)

	// Use secure cookies unless in development or using localhost
	secureCookies := cfg.Observability.Environment != "development"
	if strings.Contains(cfg.PWA.BaseURL, "localhost") || strings.HasPrefix(cfg.PWA.BaseURL, "http://") {
		secureCookies = false
	}

	// Create handlers
	handlers := authhandlers.NewAuthHandlers(service, eventBus, helper, logger, tracer, secureCookies)

	// Create router
	router := authrouter.NewRouter(handlers, nc)

	// Register HTTP routes
	if httpRouter != nil {
		limiter := authhandlers.NewIPRateLimiter(5, 10)
		httpRouter.Route("/api/auth", func(r chi.Router) {
			r.Use(authhandlers.CORSMiddleware(cfg.HTTP.AllowedOrigins))
			r.Use(authhandlers.RateLimitMiddleware(limiter))

			// Public routes
			r.Get("/callback", handlers.HandleHTTPLogin)

			// Protected routes
			r.Group(func(r chi.Router) {
				r.Use(authhandlers.AuthMiddleware)
				r.Get("/ticket", handlers.HandleHTTPTicket)
				r.Post("/logout", handlers.HandleHTTPLogout)
			})
		})
	}

	module := &Module{
		config:        cfg,
		observability: obs,
		service:       service,
		handlers:      handlers,
		router:        router,
		logger:        logger,
	}

	return module, nil
}

// Run starts the auth module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.InfoContext(ctx, "Starting auth module")

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	// Start the router (subscribes to NATS subjects)
	authCalloutSubject := m.config.AuthCallout.Subject
	if err := m.router.Start(authCalloutSubject); err != nil {
		m.logger.ErrorContext(ctx, "Failed to start auth router",
			"error", err,
		)
		return
	}

	m.logger.InfoContext(ctx, "Auth module started",
		"magic_link_subject", authrouter.MagicLinkRequestSubject,
		"auth_callout_enabled", m.config.AuthCallout.Enabled,
	)

	<-ctx.Done()
	m.logger.InfoContext(ctx, "Auth module goroutine stopped")
}

// Close stops the auth module.
func (m *Module) Close() error {
	m.logger.Info("Stopping auth module")

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.router != nil {
		if err := m.router.Stop(); err != nil {
			m.logger.Error("Error stopping auth router", "error", err)
			return fmt.Errorf("error stopping router: %w", err)
		}
	}

	m.logger.Info("Auth module stopped")
	return nil
}

// GetService returns the auth service for use by other modules.
func (m *Module) GetService() authservice.Service {
	return m.service
}
