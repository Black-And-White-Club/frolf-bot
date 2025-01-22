package user

import (
	"context"
	"fmt"
	"log/slog"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	userrouter "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/router"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module represents the user module.
type Module struct {
	EventBus         shared.EventBus
	UserService      userservice.Service
	logger           *slog.Logger
	config           *config.Config
	UserRouter       *userrouter.UserRouter // Use the UserRouter
	SubscribersReady chan struct{}
}

// NewUserModule initializes the user module.
func NewUserModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, userDB userdb.UserDB, eventBus shared.EventBus, router *message.Router) (*Module, error) {
	logger.Info("user.NewUserModule called")

	// Initialize user service.
	userService, err := userservice.NewUserService(userDB, eventBus, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create user service: %w", err)
	}

	// Initialize user router.
	userRouter := userrouter.NewUserRouter(logger)

	// Configure the router
	if err := userRouter.Configure(router, userService); err != nil {
		return nil, fmt.Errorf("failed to configure user router: %w", err)
	}

	module := &Module{
		EventBus:         eventBus,
		UserService:      userService,
		logger:           logger,
		config:           cfg,
		UserRouter:       userRouter,
		SubscribersReady: make(chan struct{}),
	}

	// Signal that initialization is complete
	close(module.SubscribersReady)

	return module, nil
}

// IsInitialized checks if the subscribers are ready.
func (m *Module) IsInitialized() bool {
	select {
	case <-m.SubscribersReady:
		return true
	default:
		return false
	}
}
