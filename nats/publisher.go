package natsjetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

var (
	globalPublisher message.Publisher
)

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(natsURL string, logger watermill.LoggerAdapter) (message.Publisher, error) {
	marshaler := &nats.GobMarshaler{}
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}

	jsConfig := nats.JetStreamConfig{
		Disabled:       false,
		AutoProvision:  true,
		PublishOptions: []nc.PubOpt{}, // Use nc.PubOpt
	}

	Publisher, err := nats.NewPublisher( // Changed 'publisher' to 'Publisher'
		nats.PublisherConfig{
			URL:               natsURL,
			NatsOptions:       options,
			Marshaler:         marshaler,
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	return Publisher, nil // Return 'Publisher'
}

// PublishEvent publishes an event to the NATS Jetstream event bus.
func PublishEvent(ctx context.Context, event interface{}, topic string) error { // Add topic argument
	// Get the publisher
	publisher := GetPublisher()

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
