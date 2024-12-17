package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// NewCommandBus creates a new Watermill command bus with NATS JetStream publisher.
func NewCommandBus(natsURL string, logger watermill.LoggerAdapter) (*cqrs.CommandBus, error) {
	publisher, err := NewPublisher(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	commandBus, err := cqrs.NewCommandBusWithConfig(publisher, cqrs.CommandBusConfig{
		GeneratePublishTopic: func(params cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
			return params.CommandName, nil
		},
		Marshaler: cqrs.JSONMarshaler{},
		Logger:    logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill command bus: %w", err)
	}

	return commandBus, nil
}
