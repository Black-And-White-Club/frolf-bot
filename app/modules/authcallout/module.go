package authcallout

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	authservice "github.com/Black-And-White-Club/frolf-bot/app/modules/authcallout/application"
	authhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/authcallout/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/Black-And-White-Club/frolf-bot/pkg/jwt"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

const (
	// AuthCalloutSubject is the NATS subject for auth callout requests.
	AuthCalloutSubject = "$SYS.REQ.USER.AUTH"
)

// Module represents the auth callout module.
type Module struct {
	config        *config.Config
	observability observability.Observability
	service       authservice.Service
	handler       *authhandlers.AuthHandler
	subscription  *nats.Subscription
	nc            *nats.Conn
	cancelFunc    context.CancelFunc
	logger        *slog.Logger
}

// NewModule creates a new auth callout module.
func NewModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	jwtService jwt.Service,
	nc *nats.Conn,
) (*Module, error) {
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "Initializing auth callout module")

	// Parse the signing key from config
	signingKey, err := nkeys.FromSeed([]byte(cfg.AuthCallout.SigningKey))
	if err != nil {
		return nil, fmt.Errorf("failed to parse signing key: %w", err)
	}

	// Create service
	service := authservice.NewService(
		jwtService,
		signingKey,
		cfg.AuthCallout.IssuerAccount,
		logger,
		tracer,
	)

	// Create handler
	handler := authhandlers.NewAuthHandler(service, logger, tracer)

	module := &Module{
		config:        cfg,
		observability: obs,
		service:       service,
		handler:       handler,
		nc:            nc,
		logger:        logger,
	}

	return module, nil
}

// Run starts the auth callout module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.InfoContext(ctx, "Starting auth callout module")

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	// Subscribe to auth callout subject
	subject := m.config.AuthCallout.Subject
	if subject == "" {
		subject = AuthCalloutSubject
	}

	var err error
	m.subscription, err = m.nc.Subscribe(subject, m.handler.HandleAuthCallout)
	if err != nil {
		m.logger.ErrorContext(ctx, "Failed to subscribe to auth callout subject",
			"subject", subject,
			"error", err,
		)
		return
	}

	m.logger.InfoContext(ctx, "Subscribed to auth callout subject",
		"subject", subject,
	)

	<-ctx.Done()
	m.logger.InfoContext(ctx, "Auth callout module goroutine stopped")
}

// Close stops the auth callout module.
func (m *Module) Close() error {
	m.logger.Info("Stopping auth callout module")

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.subscription != nil {
		if err := m.subscription.Unsubscribe(); err != nil {
			m.logger.Error("Error unsubscribing from auth callout subject", "error", err)
			return fmt.Errorf("error unsubscribing: %w", err)
		}
	}

	m.logger.Info("Auth callout module stopped")
	return nil
}
