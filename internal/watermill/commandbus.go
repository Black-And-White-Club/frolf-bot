package watermillutil

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// NewCommandBus initializes and configures the Watermill CommandBus with the provided configuration.
func NewCommandBus(natsConfig nats.NatsConnectionConfig, logger watermill.LoggerAdapter) (*cqrs.CommandBus, error) {
	conn, err := nats.GetConnection(natsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection: %w", err)
	}

	natsConfigWatermill := nats.Config{
		Conn: conn, // Use the existing connection
		JetStream: nats.JetStreamConfig{
			Enabled: true,
		},
	}

	publisher, err := nats.NewPublisher(natsConfigWatermill, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	// Create a new command bus using cqrs.NewCommandBusWithConfig
	commandBus, err := cqrs.NewCommandBusWithConfig(publisher, cqrs.CommandBusConfig{
		GeneratePublishTopic: func(params cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
			// Define your topic generation logic here
			return params.CommandName, nil // For example, use the command name as the topic
		},
		Marshaler: cqrs.JSONMarshaler{},
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill command bus: %w", err)
	}

	return commandBus, nil
}
