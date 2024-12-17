package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsPublisher implements the Watermill Publisher interface.
type NatsPublisher struct {
	conn   *nc.Conn
	config nats.PublisherConfig
	logger watermill.LoggerAdapter
}

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(natsURL string, logger watermill.LoggerAdapter) (message.Publisher, error) {
	logger.Info("Connecting to NATS for publisher", nil)
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Info("Connected to NATS for publisher", nil)

	jsConfig := nats.JetStreamConfig{
		Disabled:      false,
		AutoProvision: true,
	}

	logger.Info("Creating NATS publisher", nil)
	return &NatsPublisher{
		conn: conn,
		config: nats.PublisherConfig{
			Marshaler:         &nats.GobMarshaler{},
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger: logger,
	}, nil
}

// Publish implements the message.Publisher interface.
func (p *NatsPublisher) Publish(topic string, messages ...*message.Message) error {
	return p.publish(context.Background(), topic, messages...)
}

func (p *NatsPublisher) publish(ctx context.Context, topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		msg.SetContext(ctx)
	}
	p.logger.Info("Creating NATS publisher for Publish", nil)
	pub, err := nats.NewPublisher(p.config, p.logger)
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}
	p.logger.Info("Publishing message", watermill.LogFields{"topic": topic})
	return pub.Publish(topic, messages...)
}

func (p *NatsPublisher) Close() error {
	p.logger.Info("Closing NATS publisher connection", nil)
	p.conn.Close()
	return nil
}
