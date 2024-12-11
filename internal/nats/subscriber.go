package natsutil

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NewSubscriber creates a new NATS JetStream subscriber.
func NewSubscriber(config watermill.NatsConnectionConfig, conn *nc.Conn, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	marshaler := &nats.GobMarshaler{}
	subscribeOptions := []nc.SubOpt{
		nc.DeliverAll(),
		nc.AckExplicit(),
	}

	jsConfig := nats.JetStreamConfig{
		Disabled:         false,
		AutoProvision:    true,
		SubscribeOptions: subscribeOptions,
	}

	subscriber, err := nats.NewSubscriber(nats.WithConn(conn), // Use the provided connection
		nats.SubscriberConfig{
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
