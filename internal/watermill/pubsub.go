package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

type PubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	js         nats.JetStreamContext // Our JetStream context field
	nc         *nats.Conn            // Add this field to store the NATS connection
}

func NewPubSub(natsURL string, logger watermill.LoggerAdapter) (*PubSub, error) {
	pub, err := NewPublisher(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	sub, err := NewSubscriber(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	// Get the JetStream context from the subscriber
	js := sub.js

	return &PubSub{
		publisher:  pub,
		subscriber: sub,
		js:         js,       // Initialize the js field
		nc:         sub.conn, // Store the NATS connection from the subscriber
	}, nil
}

// Publish publishes a single message to the specified topic.
func (ps *PubSub) Publish(topic string, messages ...*message.Message) error {
	return ps.publisher.Publish(topic, messages...)
}

func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return ps.subscriber.Subscribe(ctx, topic)
}

func (ps *PubSub) Close() error {
	var errs []error

	if err := ps.subscriber.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close subscriber: %w", err))
	}

	if err := ps.publisher.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close publisher: %w", err))
	}

	if ps.nc != nil {
		ps.nc.Close()
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors occurred during close: %v", errs)
	}

	return nil
}

func (ps *PubSub) GetJetStreamContext() nats.JetStreamContext {
	return ps.js // Simply return the stored JetStream context
}

// Add a method to get the NATS connection
func (ps *PubSub) NatsConn() *nats.Conn {
	return ps.nc
}
