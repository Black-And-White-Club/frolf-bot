package natsutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
)

// NewWatermillPublisher creates a new Watermill NATS publisher using a pooled connection.
func NewWatermillPublisher(logger watermill.LoggerAdapter) (message.Publisher, error) {
	// Get a connection from the pool
	_, err := GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection: %w", err)
	}

	pub, err := nats.NewPublisher(nats.PublisherConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	return pub, nil
}

// NewWatermillSubscriber creates a new Watermill NATS subscriber using a pooled connection.
func NewWatermillSubscriber(logger watermill.LoggerAdapter) (message.Subscriber, error) {
	// Get a connection from the pool
	_, err := GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection: %w", err)
	}

	sub, err := nats.NewSubscriber(nats.SubscriberConfig{}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return sub, nil
}
