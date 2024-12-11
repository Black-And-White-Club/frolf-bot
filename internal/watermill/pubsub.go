package watermillutil

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type PubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
}

func NewPubSub(config Config, logger watermill.LoggerAdapter) (*PubSub, error) {
	conn, err := nats.GetConnection(config.Nats)
	if err != nil {
		return nil, fmt.Errorf("failed to get NATS connection: %w", err)
	}

	pub, err := nats.NewPublisher(config.Nats, conn, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	sub, err := nats.NewSubscriber(config.Nats, conn, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &PubSub{
		publisher:  pub,
		subscriber: sub,
	}, nil
}

// Publish publishes a single message to the specified topic.
func (ps *PubSub) Publish(topic string, msg *message.Message) error {
	return ps.publisher.Publish(topic, msg)
}

func (ps *PubSub) Subscribe(topic string) (<-chan *message.Message, error) {
	return ps.subscriber.Subscribe(context.Background(), topic)
}

func (ps *PubSub) Close() error {
	err := ps.subscriber.Close()
	if err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	return nil
}
