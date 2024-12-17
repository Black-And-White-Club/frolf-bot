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
func NewPublisher(natsURL string, logger watermill.LoggerAdapter) (*NatsPublisher, error) {
	logger.Info("Connecting to NATS for publisher", nil) // Log before connecting
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Info("Connected to NATS for publisher", nil) // Log after connecting

	jsConfig := nats.JetStreamConfig{
		Disabled:      false,
		AutoProvision: true,
	}

	logger.Info("Creating NATS publisher", nil) // Log before creating publisher
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

func (p *NatsPublisher) Publish(ctx context.Context, topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		msg.SetContext(ctx)
	}
	p.logger.Info("Creating NATS publisher for Publish", nil) // Log before creating publisher
	pub, err := nats.NewPublisher(p.config, p.logger)
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}
	p.logger.Info("Publishing message", watermill.LogFields{"topic": topic}) // Log before publishing
	return pub.Publish(topic, messages...)
}

func (p *NatsPublisher) Close() error {
	p.logger.Info("Closing NATS publisher connection", nil) // Log before closing
	p.conn.Close()
	return nil
}
