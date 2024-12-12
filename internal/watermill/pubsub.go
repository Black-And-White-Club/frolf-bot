package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type PubSub struct {
	publisher  message.Publisher
	subscriber message.Subscriber
}

func NewPubSub(natsURL string, logger watermill.LoggerAdapter) (*PubSub, error) {
	pub, err := NewPublisher(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	sub, err := NewSubscriber(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &PubSub{
		publisher:  pub,
		subscriber: sub,
	}, nil
}

// Publish publishes a single message to the specified topic.
func (ps *PubSub) Publish(topic string, messages ...*message.Message) error {
	return ps.publisher.Publish(topic, messages...)
}

func (ps *PubSub) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	return ps.subscriber.Subscribe(ctx, topic)
}

func (ps *PubSub) Close() error {
	err := ps.subscriber.Close()
	if err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	return nil
}
