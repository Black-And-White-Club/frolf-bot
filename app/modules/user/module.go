package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	userrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
)

// Module represents the user module.
type Module struct {
	EventBus           eventbus.EventBus
	UserService        userservice.Service
	config             *config.Config
	UserRouter         *userrouter.UserRouter
	cancelFunc         context.CancelFunc
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

// NewUserModule creates a new instance of the User module.
// It creates the Prometheus Registry and passes it to the router.
func NewUserModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	userDB userdb.UserDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.UserMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "user.NewUserModule called")

	userService := userservice.NewUserService(userDB, eventBus, logger, metrics, tracer)

	// Create a new Prometheus Registry for this module's router and metrics.
	// This registry will be used by the router's metrics builder.
	prometheusRegistry := prometheus.NewRegistry()

	// Initialize user router with observability and the Prometheus Registry.
	// The router will conditionally add metrics based on environment.
	userRouter := userrouter.NewUserRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helpers,
		tracer,
		prometheusRegistry,
	)

	// Pass the original EventBus to configure if it needs it for stream creation etc.
	if err := userRouter.Configure(userService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		UserService:        userService,
		config:             cfg,
		UserRouter:         userRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting user module")

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	<-ctx.Done()
	logger.InfoContext(ctx, "User module goroutine stopped")
}

func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping user module")
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	logger.Info("User module stopped")
	return nil
}
