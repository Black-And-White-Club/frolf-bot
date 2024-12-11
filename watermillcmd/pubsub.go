// watermillcmd/pubsub.go
package watermillcmd

import (
	"context"
	"fmt"

	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
)

type NatsPubSub struct {
	subscriber *nats.Subscriber
} // Concrete implementation using NATS

// NewNatsPubSub creates a new NatsPubSub instance.
func NewNatsPubSub() *NatsPubSub {
	return &NatsPubSub{}
}

// PublishCommand publishes a command to the NATS Jetstream event bus.
func (ps *NatsPubSub) PublishCommand(ctx context.Context, topic string, command interface{}) error {
	// Reuse the Publish method
	return ps.Publish(ctx, topic, command)
}

// Publish publishes a message to the specified topic using NATS Jetstream.
func (ps *NatsPubSub) Publish(ctx context.Context, topic string, message interface{}) error {
	// Use the global natsjetstream.PublishEvent function
	return natsjetstream.PublishEvent(ctx, message, topic)
}

// PublishEvent publishes an event to the NATS Jetstream event bus.
func (ps *NatsPubSub) PublishEvent(ctx context.Context, topic string, event interface{}) error {
	// Use the global natsjetstream.PublishEvent function
	return natsjetstream.PublishEvent(ctx, event, topic)
}

// Subscribe subscribes to the specified topic on NATS Jetstream.
func (ps *NatsPubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	// Create a nats.Subscriber if it doesn't exist
	if ps.subscriber == nil {
		subscriber, err := nats.NewSubscriber(nats.SubscriberConfig{}, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
		}
		ps.subscriber = subscriber
	}

	// Subscribe to the topic using the subscriber
	subscription, err := ps.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	return subscription, nil
}

// Close closes the NATS Jetstream subscriber.
func (ps *NatsPubSub) Close() error {
	if ps.subscriber != nil {
		if err := ps.subscriber.Close(); err != nil {
			return fmt.Errorf("failed to close NATS subscriber: %w", err)
		}
	}
	return nil
}
