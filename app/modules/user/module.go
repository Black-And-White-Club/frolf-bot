package user

import (
	"context"
	"fmt"
	"sync"
	"time"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/service"
	usersubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// Module represents the user module.
type Module struct {
	Subscriber  message.Subscriber
	Publisher   message.Publisher
	UserService userservice.Service
	Handlers    userhandlers.Handlers
	logger      watermill.LoggerAdapter
	config      *config.Config
	initialized bool
	initMutex   sync.Mutex
}

func NewUserModule(ctx context.Context, cfg *config.Config, logger watermill.LoggerAdapter, userDB userdb.UserDB) (*Module, error) {
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

	userService := userservice.NewUserService(publisher, subscriber, userDB, logger)
	userHandlers := userhandlers.NewHandlers(userService, publisher, logger)

	module := &Module{
		Subscriber:  subscriber,
		Publisher:   publisher,
		UserService: userService,
		Handlers:    userHandlers,
		logger:      logger,
		config:      cfg,
		initialized: false,
	}

	go func() {
		userSubscriber, closer, err := usersubscribers.NewUserSubscribers(subscriber, userHandlers, logger)
		if err != nil {
			logger.Error("Failed to create user subscribers", err, nil)
			return // Or other appropriate error handling
		}
		defer closer.Close() // Close the subscriber when the module is done

		if err := userSubscriber.SubscribeToUserEvents(ctx); err != nil {
			logger.Error("Failed to subscribe to user events", err, nil)
			fmt.Printf("Fatal error subscribing to user events: %v\n", err)
			return // Or other appropriate error handling
		}

		logger.Info("User module subscribers are ready", nil)
		module.initMutex.Lock()
		module.initialized = true
		module.initMutex.Unlock()
	}()

	return module, nil
}

// IsInitialized safely checks module initialization
func (m *Module) IsInitialized() bool {
	m.initMutex.Lock()
	defer m.initMutex.Unlock()
	return m.initialized
}
