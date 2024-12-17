package watermillutil

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsSubscriber implements the Watermill Subscriber interface.
type NatsSubscriber struct {
	conn          *nc.Conn
	config        nats.SubscriberConfig
	logger        watermill.LoggerAdapter
	js            nc.JetStreamContext
	reconnectOpts []nc.Option
	sub           *nats.Subscriber // Store the subscriber instance
}

// NewSubscriber creates a new NATS JetStream subscriber.
func NewSubscriber(natsURL string, logger watermill.LoggerAdapter, opts ...nc.Option) (*NatsSubscriber, error) {
	logger.Info("Connecting to NATS for subscriber", nil)

	// Add reconnect options
	reconnectOpts := []nc.Option{
		nc.MaxReconnects(-1),
		nc.ReconnectWait(2 * time.Second),
	}
	reconnectOpts = append(reconnectOpts, opts...)

	conn, err := nc.Connect(natsURL, reconnectOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Info("Connected to NATS for subscriber", nil)

	logger.Info("Creating JetStream context", nil)
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	logger.Info("Created JetStream context", nil)

	subscribeOptions := []nc.SubOpt{
		nc.DeliverAll(),
		nc.AckExplicit(),
	}

	// Create the SubscriberConfig
	subscriberConfig := nats.SubscriberConfig{
		Unmarshaler: &nats.GobMarshaler{},
		JetStream: nats.JetStreamConfig{
			Disabled:         false,
			AutoProvision:    true,
			SubscribeOptions: subscribeOptions,
		},
		SubjectCalculator: nats.DefaultSubjectCalculator,
	}

	logger.Info("Creating NATS subscriber", nil)
	sub, err := nats.NewSubscriber(subscriberConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
	}

	return &NatsSubscriber{
		conn:          conn,
		reconnectOpts: reconnectOpts,
		config:        subscriberConfig, // Use the correct SubscriberConfig
		logger:        logger,
		js:            js,
		sub:           sub, // Store the subscriber instance
	}, nil
}

// Subscribe subscribes to messages on a topic.
func (s *NatsSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	s.logger.Info("Subscribing to topic", watermill.LogFields{"topic": topic})
	messages, err := s.sub.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to NATS topic: %w", err) // More specific error
	}
	return messages, nil
}

// Close closes the subscriber connection.
func (s *NatsSubscriber) Close() error {
	s.logger.Info("Closing NATS subscriber connection", nil)
	s.conn.Close() // Close the connection without checking for an error
	return nil
}

// GetJetStreamContext provides an exported method to access the JetStream context
func (s *NatsSubscriber) GetJetStreamContext() nc.JetStreamContext {
	return s.js
}
