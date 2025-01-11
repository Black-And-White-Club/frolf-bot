package user

import (
	"context"
	"log/slog"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	usersubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/subscribers"
	userinterfaces "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
)

// Module represents the user module.
type Module struct {
	EventBus         shared.EventBus
	UserService      userservice.Service
	Handlers         userinterfaces.Handlers
	Subscribers      usersubscribers.UserEventSubscriber
	logger           *slog.Logger
	config           *config.Config
	SubscribersReady chan struct{} // Add a channel to signal readiness
}

// NewUserModule initializes the user module.
func NewUserModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, userDB userdb.UserDB, eventBus shared.EventBus) (*Module, error) {
	logger.Info("user.NewUserModule called") // Log function call

	// Initialize user service.
	userService := userservice.NewUserService(userDB, eventBus, logger)

	// Initialize user handlers.
	userHandlers := userhandlers.NewHandlers(userService, eventBus, logger)

	// Initialize user subscribers.
	userSubscribers := usersubscribers.NewSubscribers(eventBus, userHandlers, logger)

	module := &Module{
		EventBus:         eventBus,
		UserService:      userService,
		Handlers:         userHandlers,
		Subscribers:      userSubscribers,
		logger:           logger,
		config:           cfg,
		SubscribersReady: make(chan struct{}),
	}

	go func() {
		subscriberCtx := context.Background()

		if err := module.Subscribers.SubscribeToUserEvents(subscriberCtx, eventBus, userHandlers, logger); err != nil {
			logger.Error("Failed to subscribe to user events", slog.Any("error", err))
			return
		}

		logger.Info("User module subscribers are ready")

		// Signal that initialization is complete
		close(module.SubscribersReady)
	}()

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
