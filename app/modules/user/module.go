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
)

// Module represents the user module.
type Module struct {
	EventBus      eventbus.EventBus
	UserService   userservice.Service
	config        *config.Config
	UserRouter    *userrouter.UserRouter
	cancelFunc    context.CancelFunc
	helper        utils.Helpers
	observability observability.Observability
}

// NewUserModule creates a new instance of the User module.
func NewUserModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	userDB userdb.UserDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
) (*Module, error) {
	// Extract observability components
	logger := obs.GetLogger()
	metrics := obs.GetMetrics().UserMetrics()
	tracer := obs.GetTracer()

	logger.Info("user.NewUserModule called")

	// Initialize user service with observability components
	userService := userservice.NewUserService(userDB, eventBus, logger, metrics, tracer)

	// Initialize user router with observability
	userRouter := userrouter.NewUserRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the user service.
	if err := userRouter.Configure(userService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	module := &Module{
		EventBus:      eventBus,
		UserService:   userService,
		config:        cfg,
		UserRouter:    userRouter,
		helper:        helpers,
		observability: obs,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.GetLogger()
	logger.Info("Starting user module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	// If we have a wait group, mark as done when this method exits
	if wg != nil {
		defer wg.Done()
	}

	// Keep this goroutine alive until the context is canceled
	<-ctx.Done()
	logger.Info("User module goroutine stopped")
}

func (m *Module) Close() error {
	logger := m.observability.GetLogger()
	logger.Info("Stopping user module") // Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	logger.Info("User  module stopped")
	return nil
}
