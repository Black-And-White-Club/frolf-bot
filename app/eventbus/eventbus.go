package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// eventBus implements the shared.EventBus interface.
type eventBus struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	logger     *slog.Logger
}

// NewEventBus creates and returns an EventBus with a connection to NATS JetStream.
func NewEventBus(natsURL string, logger *slog.Logger) (shared.EventBus, error) {
	// Initialize the Watermill publisher
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         natsURL,
			Marshaler:   nats.GobMarshaler{}, // Use GobMarshaler for Watermill compatibility
			NatsOptions: []nc.Option{nc.RetryOnFailedConnect(true)},
		},
		watermill.NewSlogLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create publisher: %w", err)
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:         natsURL,
			Unmarshaler: nats.GobMarshaler{}, // Use GobMarshaler for Watermill compatibility
			NatsOptions: []nc.Option{nc.RetryOnFailedConnect(true)},
		},
		watermill.NewSlogLogger(logger),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	return &eventBus{
		publisher:  publisher,
		subscriber: subscriber,
		logger:     logger,
	}, nil
}

// Publish publishes a message to the specified subject.
func (eb *eventBus) Publish(ctx context.Context, topic string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	return eb.publisher.Publish(topic, msg)
}

// Subscribe subscribes to a subject and returns a channel of messages.
func (eb *eventBus) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return eb.subscriber.Subscribe(ctx, topic)
}

// Close closes the publisher and subscriber.
func (eb *eventBus) Close() error {
	if err := eb.publisher.Close(); err != nil {
		return fmt.Errorf("failed to close publisher: %w", err)
	}
	if err := eb.subscriber.Close(); err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	return nil
}
