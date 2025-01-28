package user

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/router"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Module struct {
	EventBus    shared.EventBus
	UserService userservice.Service
	logger      *slog.Logger
	config      *config.Config
	UserRouter  *userrouter.UserRouter
	cancelFunc  context.CancelFunc
}

func NewUserModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, userDB userdb.UserDB, eventBus shared.EventBus, router *message.Router) (*Module, error) {
	logger.Info("user.NewUserModule called")

	// Initialize user service.
	userService, err := userservice.NewUserService(userDB, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service: %w", err)
	}

	// Initialize user router.
	userRouter := userrouter.NewUserRouter(logger, router, eventBus)

	// Configure the router with user service.
	if err := userRouter.Configure(userService); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	module := &Module{
		EventBus:    eventBus,
		UserService: userService,
		logger:      logger,
		config:      cfg,
		UserRouter:  userRouter, // Set the UserRouter
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting user module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	// Keep this goroutine alive until the context is canceled
	<-ctx.Done()
	m.logger.Info("User module goroutine stopped")
}

func (m *Module) Close() error {
	m.logger.Info("Stopping user module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.logger.Info("User module stopped")
	return nil
}
