package leaderboard

import (
	"context"
	"fmt"
	"sync"
	"time"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/handlers"
	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/service"
	leaderboardsubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// Module represents the leaderboard module.
type Module struct {
	Subscriber  message.Subscriber
	Publisher   message.Publisher
	Service     leaderboardservice.Service
	Handlers    leaderboardhandlers.Handlers
	logger      watermill.LoggerAdapter
	config      *config.Config
	initialized bool
	initMutex   sync.Mutex
}

// NewModule creates a new LeaderboardModule with the provided dependencies.
func NewModule(ctx context.Context, cfg *config.Config, logger watermill.LoggerAdapter, leaderboardDB leaderboarddb.LeaderboardDB) (*Module, error) {
	// Initialize NATS publisher and subscriber
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

	// Initialize leaderboard service and handlers
	leaderboardService := leaderboardservice.NewLeaderboardService(publisher, leaderboardDB, logger)
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, publisher, logger)

	module := &Module{
		Subscriber: subscriber,
		Publisher:  publisher,
		Service:    leaderboardService,
		Handlers:   leaderboardHandlers,
		logger:     logger,
		config:     cfg,
	}

	// Subscribe to leaderboard events
	go func() {
		leaderboardSubscribers := leaderboardsubscribers.NewLeaderboardSubscribers(subscriber, logger, leaderboardHandlers)

		if err := leaderboardSubscribers.SubscribeToLeaderboardEvents(ctx); err != nil {
			logger.Error("Failed to subscribe to leaderboard events", err, nil)
			fmt.Printf("Fatal error subscribing to leaderboard events: %v\n", err)
		} else {
			logger.Info("Leaderboard module subscribers are ready", nil)
			module.initMutex.Lock()
			module.initialized = true
			module.initMutex.Unlock()
		}
	}()

	return module, nil
}

// IsInitialized safely checks module initialization
func (m *Module) IsInitialized() bool {
	m.initMutex.Lock()
	defer m.initMutex.Unlock()
	return m.initialized
}
