package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/application"
	authhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/handlers"
	authjwt "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/jwt"
	authnats "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/nats"
	authrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
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
) (*Module, error) {
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "Initializing auth module")

	// Create JWT provider
	jwtProvider := authjwt.NewProvider(cfg.JWT.Secret)

	// Create NATS JWT builder if auth callout is enabled
	var userJWTBuilder authnats.UserJWTBuilder
	if cfg.AuthCallout.Enabled {
		signingKey, err := nkeys.FromSeed([]byte(cfg.AuthCallout.SigningNKey))
		if err != nil {
			return nil, fmt.Errorf("failed to parse signing key: %w", err)
		}
		userJWTBuilder = authnats.NewUserJWTBuilder(signingKey, cfg.AuthCallout.IssuerNKey)
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
		serviceConfig,
		logger,
		tracer,
	)

	// Create handlers
	handlers := authhandlers.NewAuthHandlers(service, eventBus, helper, logger, tracer)

	// Create router
	router := authrouter.NewRouter(handlers, nc)

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
