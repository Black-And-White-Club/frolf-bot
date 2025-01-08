package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	eventbus "github.com/Black-And-White-Club/tcr-bot/app/events"
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
	EventBus     shared.EventBus
	EventAdapter shared.EventAdapterInterface
	UserService  userservice.Service
	Handlers     userinterfaces.Handlers
	Subscribers  usersubscribers.UserEventSubscriber
	logger       shared.LoggerAdapter
	config       *config.Config
	initialized  bool
	initMutex    sync.Mutex
}

// NewUserModule initializes the user module.
func NewUserModule(ctx context.Context, cfg *config.Config, logger shared.LoggerAdapter, userDB userdb.UserDB) (*Module, error) {
	logger.Info("NewUserModule started", shared.LogFields{"contextErr": ctx.Err()})

	// Initialize EventAdapter.
	eventAdapter, err := adapters.NewEventAdapter(cfg.NATS.URL, logger, eventbus.NewStreamNamer())
	if err != nil {
		return nil, fmt.Errorf("failed to create event adapter: %w", err)
	}

	// Initialize EventBus.
	eventBus := eventbus.NewEventBus(eventAdapter)

	// Initialize user service.
	userService := userservice.NewUserService(userDB, eventBus, logger, eventAdapter)

	// Initialize user handlers.
	userHandlers := userhandlers.NewHandlers(userService, eventBus, logger)

	// Initialize user subscribers.
	userSubscribers := usersubscribers.NewSubscribers(eventBus, userHandlers, logger)

	module := &Module{
		EventBus:     eventBus,
		EventAdapter: eventAdapter,
		UserService:  userService,
		Handlers:     userHandlers,
		Subscribers:  userSubscribers,
		logger:       logger,
		config:       cfg,
		initialized:  false,
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		subscriberCtx := context.Background()

		module.logger.Info("Created NATS subscriber", shared.LogFields{
			"nats_url": cfg.NATS.URL,
		})

		if err := module.Subscribers.SubscribeToUserEvents(subscriberCtx, eventBus, userHandlers, logger); err != nil {
			logger.Error("Failed to subscribe to user events", err, nil)
			return
		}

		logger.Info("User module subscribers are ready", nil)
		module.initMutex.Lock()
		module.initialized = true
		module.initMutex.Unlock()
	}()

	wg.Wait()

	return module, nil
}

// IsInitialized safely checks module initialization.
func (m *Module) IsInitialized() bool {
	m.initMutex.Lock()
	defer m.initMutex.Unlock()
	return m.initialized
}
