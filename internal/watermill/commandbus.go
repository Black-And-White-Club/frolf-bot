package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewCommandBus(natsURL string, logger watermill.LoggerAdapter) (*cqrs.CommandBus, error) {
	publisher, err := NewPublisher(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	// Create a publisher that satisfies the message.Publisher interface
	wmPublisher := &watermillPublisher{publisher}

	commandBus, err := cqrs.NewCommandBusWithConfig(wmPublisher, cqrs.CommandBusConfig{
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

// watermillPublisher is a wrapper to adapt NatsPublisher to message.Publisher
type watermillPublisher struct {
	*NatsPublisher
}

// Publish implements the message.Publisher interface
func (p *watermillPublisher) Publish(topic string, messages ...*message.Message) error {
	// Call the underlying NatsPublisher.Publish with a background context
	return p.NatsPublisher.Publish(context.Background(), topic, messages...)
}
