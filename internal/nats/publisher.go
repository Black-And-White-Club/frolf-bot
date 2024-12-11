package natsutil

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(config ConnectionConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
	conn, err := GetConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection: %w", err)
	}

	jsConfig := nats.JetStreamConfig{
		Disabled:       false,
		AutoProvision:  true,
		PublishOptions: []nc.PubOpt{}, // Use nc.PubOpt
	}

	publisher, err := nats.NewPublisher( // Changed 'publisher' to 'Publisher'
		nats.PublisherConfig{
			Conn:              conn, // Use the provided connection
			Marshaler:         &nats.GobMarshaler{},
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	return publisher, nil // Return 'Publisher'
}

// PublishEvent publishes an event to the NATS JetStream event bus.
func PublishEvent(ctx context.Context, event interface{}, topic string) error {
	// Get the publisher (you might need to handle the case where the publisher is not initialized)
	publisher := GetPublisher()
	if publisher == nil {
		return fmt.Errorf("publisher is not initialized")
	}

	// Marshal the event (you might need to adjust this based on your event types)
	payload, err := json.Marshal(event) // Or use another marshaling method
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create a Watermill message
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Publish the message to the specified topic
	if err := publisher.Publish(topic, msg); err != nil { // Use topic in Publish call
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func GetPublisher() message.Publisher {
	return globalPublisher
}

// InitPublisher initializes the global publisher.
func InitPublisher(publisher message.Publisher) {
	globalPublisher = publisher
}
