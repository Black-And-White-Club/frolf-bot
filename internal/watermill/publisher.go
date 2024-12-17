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

// NatsPublisher implements the Watermill Publisher interface.
type NatsPublisher struct {
	conn          *nc.Conn
	config        nats.PublisherConfig
	logger        watermill.LoggerAdapter
	reconnectOpts []nc.Option
}

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(natsURL string, logger watermill.LoggerAdapter, opts ...nc.Option) (message.Publisher, error) {
	logger.Info("Connecting to NATS for publisher", nil)

	// Add reconnect options
	reconnectOpts := []nc.Option{
		nc.MaxReconnects(-1),
		nc.ReconnectWait(2 * time.Second),
	}
	reconnectOpts = append(reconnectOpts, opts...)

	conn, err := nc.Connect(natsURL, reconnectOpts...)
	if err != nil {
		logger.Error("Failed to connect to NATS", err, nil) // Log the error
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Info("Connected to NATS for publisher", nil)

	jsConfig := nats.JetStreamConfig{
		Disabled:      false,
		AutoProvision: true,
	}

	logger.Info("Creating NATS publisher", nil)
	return &NatsPublisher{
		conn:          conn,
		reconnectOpts: reconnectOpts,
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

	p.logger.Info("Publishing message", watermill.LogFields{"topic": topic})
	// Use the existing connection to publish messages
	for _, msg := range messages {
		if err := p.conn.Publish(topic, msg.Payload); err != nil {
			return fmt.Errorf("failed to publish message to NATS: %w", err) // More specific error
		}
	}

	return nil
}

// Close closes the publisher connection.
func (p *NatsPublisher) Close() error {
	p.logger.Info("Closing NATS publisher connection", nil)
	p.conn.Close() // Close the connection without checking for an error
	return nil
}
