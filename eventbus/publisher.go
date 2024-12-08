package events

import (
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

func GetPublisher() message.Publisher {
	return globalPublisher
}

// InitPublisher initializes the global publisher.
func InitPublisher(publisher message.Publisher) {
	globalPublisher = publisher
}
