package watermillcmd

import (
	"github.com/Black-And-White-Club/tcr-bot/config"
	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
)

// NewCommandBus initializes and configures the Watermill CommandBus with the provided configuration.
func NewCommandBus(config *config.Config) (*cqrs.CommandBus, error) {
	logger := watermill.NewStdLogger(false, false)

	publisher, err := natsjetstream.NewPublisher(config.NATS.URL, logger)
	if err != nil {
		return nil, err
	}

	// Create a new command bus using cqrs.NewCommandBusWithConfig
	commandBus, err := cqrs.NewCommandBusWithConfig(publisher, cqrs.CommandBusConfig{
		GeneratePublishTopic: func(params cqrs.CommandBusGeneratePublishTopicParams) (string, error) {
			// Define your topic generation logic here
			return params.CommandName, nil // For example, use the command name as the topic
		},
		Marshaler: cqrs.JSONMarshaler{}, // Use the Protobuf marshaler
		Logger:    logger,
	})
	if err != nil {
		return nil, err
	}

	return commandBus, nil
}
