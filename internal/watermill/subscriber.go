package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsSubscriber implements the Watermill Subscriber interface.
type NatsSubscriber struct {
	conn   *nc.Conn
	config nats.SubscriberConfig
	logger watermill.LoggerAdapter
	js     nc.JetStreamContext
}

// NewSubscriber creates a new NATS JetStream subscriber.
func NewSubscriber(natsURL string, logger watermill.LoggerAdapter) (*NatsSubscriber, error) {
	logger.Info("Connecting to NATS for subscriber", nil)
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
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

	jsConfig := nats.JetStreamConfig{
		Disabled:         false,
		AutoProvision:    true,
		SubscribeOptions: subscribeOptions,
	}

	logger.Info("Creating NATS subscriber", nil)
	return &NatsSubscriber{
		conn: conn,
		config: nats.SubscriberConfig{
			Unmarshaler:       &nats.GobMarshaler{},
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger: logger,
		js:     js,
	}, nil
}

func (s *NatsSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	s.logger.Info("Creating NATS subscriber for Subscribe", nil)
	sub, err := nats.NewSubscriber(s.config, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
	}

	s.logger.Info("Subscribing to topic", watermill.LogFields{"topic": topic})
	return sub.Subscribe(ctx, topic)
}

func (s *NatsSubscriber) Close() error {
	s.logger.Info("Closing NATS subscriber connection", nil)
	s.conn.Close()
	return nil
}

// GetJetStreamContext provides an exported method to access the JetStream context
func (s *NatsSubscriber) GetJetStreamContext() nc.JetStreamContext {
	return s.js
}
