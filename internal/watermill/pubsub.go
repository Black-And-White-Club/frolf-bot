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

// NewPubSub creates a new PubSub instance.
func NewPubSub(natsURL string, logger watermill.LoggerAdapter) (*PubSub, error) {
	pub, err := NewPublisher(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	sub, err := NewSubscriber(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &PubSub{
		publisher:  &watermillPublisher{pub},
		subscriber: sub,
		js:         sub.GetJetStreamContext(),
		nc:         sub.conn,
	}, nil
}

// Publish publishes messages to a topic.
func (ps *PubSub) Publish(topic string, messages ...*message.Message) error {
	return ps.publisher.Publish(topic, messages...)
}

// Subscribe subscribes to messages on a topic.
func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return ps.subscriber.Subscribe(ctx, topic)
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
