package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	nc "github.com/nats-io/nats.go"
)

// NewScheduledTaskSubscriber creates a new NATS JetStream subscriber
// specifically for scheduled tasks.
func NewScheduledTaskSubscriber(natsURL string, logger watermill.LoggerAdapter) (*NatsSubscriber, error) {
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context here
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
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

	return &NatsSubscriber{
		conn: conn,
		config: nats.SubscriberConfig{
			Unmarshaler:       &nats.GobMarshaler{},
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger: logger,
		js:     js,
	}, nil
}
