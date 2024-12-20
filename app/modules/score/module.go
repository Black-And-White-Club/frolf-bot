package score

import (
	"context"
	"fmt"
	"sync"
	"time"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/handlers"
	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/service"
	scoresubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// Module represents the score module.
type Module struct {
	Subscriber  message.Subscriber
	Publisher   message.Publisher
	Service     scoreservice.Service
	Handlers    scorehandlers.Handlers
	logger      watermill.LoggerAdapter
	config      *config.Config
	initialized bool
	initMutex   sync.Mutex
}

func NewScoreModule(ctx context.Context, cfg *config.Config, logger watermill.LoggerAdapter, scoreDB scoredb.ScoreDB) (*Module, error) {
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

	scoreService := scoreservice.NewScoreService(publisher, subscriber, scoreDB, logger)
	scoreHandlers := scorehandlers.NewScoreHandlers(scoreService, publisher, logger)

	module := &Module{
		Subscriber: subscriber,
		Publisher:  publisher,
		Service:    scoreService,
		Handlers:   scoreHandlers,
		logger:     logger,
		config:     cfg,
	}

	go func() {
		scoreSubscribers := scoresubscribers.NewScoreSubscribers(subscriber, logger, scoreHandlers, scoreService)

		if err := scoreSubscribers.SubscribeToScoreEvents(ctx); err != nil {
			logger.Error("Failed to subscribe to score events", err, nil)
			fmt.Printf("Fatal error subscribing to score events: %v\n", err)
		} else {
			logger.Info("Score module subscribers are ready", nil)
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
