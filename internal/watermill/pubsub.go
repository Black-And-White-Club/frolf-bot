package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// PubSub combines Publisher and Subscriber with JetStream context.
type PubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	js         nats.JetStreamContext
	nc         *nats.Conn
}

// NewPubSub creates a new PubSub instance with optional publisher and subscriber.
func NewPubSub(natsURL string, logger watermill.LoggerAdapter, publisher message.Publisher, subscriber message.Subscriber) (*PubSub, error) {
	logger.Info("Creating new PubSub instance", nil) // Log entry for creating PubSub

	if publisher == nil {
		logger.Info("Creating new Publisher since none was provided", nil)
		var err error
		publisher, err = NewPublisher(natsURL, logger)
		if err != nil {
			logger.Error("Failed to create Watermill publisher", err, nil) // Log the error
			return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
		}
	}

	if subscriber == nil {
		logger.Info("Creating new Subscriber since none was provided", nil)
		var err error
		subscriber, err = NewSubscriber(natsURL, logger)
		if err != nil {
			logger.Error("Failed to create Watermill subscriber", err, nil) // Log the error
			return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
		}
	}

	logger.Info("Returning new PubSub instance", nil)
	return &PubSub{
		publisher:  publisher,
		subscriber: subscriber,
		js:         subscriber.(*NatsSubscriber).GetJetStreamContext(),
		nc:         subscriber.(*NatsSubscriber).conn,
	}, nil
}

// Publish publishes messages to a topic.
func (ps *PubSub) Publish(topic string, messages ...*message.Message) error {
	if err := ps.publisher.Publish(topic, messages...); err != nil {
		return fmt.Errorf("failed to publish message: %w", err) // Enhanced error handling
	}
	return nil
}

// Subscribe subscribes to messages on a topic.
func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	messages, err := ps.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err) // Enhanced error handling
	}
	return messages, nil
}

// Close closes the publisher and subscriber connections.
func (ps *PubSub) Close() error {
	if err := ps.publisher.Close(); err != nil {
		return fmt.Errorf("failed to close publisher: %w", err)
	}
	if err := ps.subscriber.Close(); err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	if ps.nc != nil {
		ps.nc.Close()
	}
	return nil
}

// JetStreamContext returns the JetStream context.
func (ps *PubSub) JetStreamContext() nats.JetStreamContext {
	return ps.js
}

// NatsConn returns the NATS connection.
func (ps *PubSub) NatsConn() *nats.Conn {
	return ps.nc
}
