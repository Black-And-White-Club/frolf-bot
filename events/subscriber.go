package events

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NewSubscriber creates a new NATS JetStream subscriber.
func NewSubscriber(natsURL string, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	marshaler := &nats.GobMarshaler{}
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}
	subscribeOptions := []nc.SubOpt{
		nc.DeliverAll(),
		nc.AckExplicit(),
	}

	jsConfig := nats.JetStreamConfig{
		Disabled:         false,
		AutoProvision:    true,
		SubscribeOptions: subscribeOptions,
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:               natsURL,
			CloseTimeout:      30 * time.Second,
			AckWaitTimeout:    30 * time.Second,
			NatsOptions:       options,
			Unmarshaler:       marshaler,
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
	}

	return subscriber, nil
}
