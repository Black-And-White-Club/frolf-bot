package round

import (
	"context"
	"fmt"
	"sync"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/service"
	roundsubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// Module represents the round module.
type Module struct {
	Subscriber  message.Subscriber
	Publisher   message.Publisher
	Service     roundservice.Service
	Handlers    roundhandlers.Handlers
	logger      watermill.LoggerAdapter
	config      *config.Config
	initialized bool
	initMutex   sync.Mutex
}

// NewRoundModule creates a new instance of the Round module.
func NewRoundModule(ctx context.Context, cfg *config.Config, logger watermill.LoggerAdapter, roundDB rounddb.RoundDB) (*Module, error) {
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

	roundService := &roundservice.RoundService{
		RoundDB:    roundDB,
		Publisher:  publisher,
		Subscriber: subscriber,
	}

	roundHandlers := &roundhandlers.RoundHandlers{
		RoundService: roundService,
		Publisher:    publisher,
	}

	module := &Module{
		Subscriber:  subscriber,
		Publisher:   publisher,
		Service:     roundService,
		Handlers:    roundHandlers,
		logger:      logger,
		config:      cfg,
		initialized: false,
	}

	go func() {
		// 3. Set up subscribers
		subscribers := &roundsubscribers.RoundSubscribers{
			Subscriber: subscriber,
			Handlers:   *roundHandlers,
		}

		if err := subscribers.SubscribeToRoundManagementEvents(ctx); err != nil {
			panic(err)
		}
		if err := subscribers.SubscribeToParticipantManagementEvents(ctx); err != nil {
			panic(err)
		}
		if err := subscribers.SubscribeToRoundFinalizationEvents(ctx); err != nil {
			panic(err)
		}
		if err := subscribers.SubscribeToRoundStartedEvents(ctx); err != nil {
			panic(err)
		}

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
