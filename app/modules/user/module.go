package user

import (
	"context"
	"fmt"
	"sync"
	"time"

	eventbus "github.com/Black-And-White-Club/tcr-bot/app/events"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/repositories"
	usersubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/subscribers"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	nc "github.com/nats-io/nats.go"
)

// Module represents the user module.
type Module struct {
	EventBus    eventbus.EventBus
	UserService userservice.Service
	Handlers    userhandlers.Handlers
	Subscribers usersubscribers.EventSubscribers
	logger      types.LoggerAdapter
	config      *config.Config
	initialized bool
	initMutex   sync.Mutex
}

func NewUserModule(ctx context.Context, cfg *config.Config, logger watermill.LoggerAdapter, userDB userdb.UserDB) (*Module, error) {
	logger.Info("NewUserModule started", watermill.LogFields{"contextErr": ctx.Err()})

	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.ErrorHandler(func(_ *nc.Conn, s *nc.Subscription, err error) {
			if s != nil {
				logger.Error("Error in subscription", err, watermill.LogFields{
					"subject": s.Subject,
					"queue":   s.Queue,
				})
			} else {
				logger.Error("Error in connection", err, nil)
			}
		}),
	}

	jsConfig := nats.JetStreamConfig{
		Disabled: false,
	}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         cfg.NATS.URL,
			NatsOptions: options,
			Marshaler:   &nats.NATSMarshaler{},
			JetStream:   jsConfig,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:         cfg.NATS.URL,
			NatsOptions: options,
			Unmarshaler: &nats.NATSMarshaler{},
			JetStream:   jsConfig,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Wrap the publisher and subscriber to fulfill the EventBus interface
	eventBus := eventbus.NewEventBus(publisher, subscriber)

	userService := userservice.NewUserService(eventBus, userDB, logger)     // Pass eventBus instead of publisher and subscriber
	userHandlers := userhandlers.NewHandlers(userService, eventBus, logger) // Pass eventBus instead of publisher
	userSubscribers := usersubscribers.NewSubscribers(userService, eventBus, logger)

	module := &Module{
		EventBus:    eventBus,
		UserService: userService,
		Handlers:    userHandlers,
		Subscribers: userSubscribers,
		logger:      logger,
		config:      cfg,
		initialized: false,
	}

	// Use a WaitGroup to synchronize the subscriber goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		// Create a new background context for the subscriber
		subscriberCtx := context.Background()

		// Logging the subscriber details
		module.logger.Info("Created NATS subscriber", watermill.LogFields{
			"nats_url": cfg.NATS.URL,
		})

		if err := usersubscribers.SubscribeToUserEvents(subscriberCtx, eventBus, userHandlers, logger); err != nil {
			// Decide how to handle subscriber errors - log, panic or return
			logger.Error("Failed to subscribe to user events", err, nil)
			return
		}

		logger.Info("User module subscribers are ready", nil)
		module.initMutex.Lock()
		module.initialized = true
		module.initMutex.Unlock()
	}()

	wg.Wait() // Wait for the subscriber goroutine to finish subscribing

	return module, nil
}

// IsInitialized safely checks module initialization
func (m *Module) IsInitialized() bool {
	m.initMutex.Lock()
	defer m.initMutex.Unlock()
	return m.initialized
}
