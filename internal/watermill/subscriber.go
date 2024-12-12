package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsSubscriber implements the Watermill Subscriber interface.
type NatsSubscriber struct {
	conn   *nc.Conn
	config nats.SubscriberConfig
	logger watermill.LoggerAdapter
}

// NewSubscriber creates a new NATS JetStream subscriber.
func NewSubscriber(natsURL string, logger watermill.LoggerAdapter) (*NatsSubscriber, error) {
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
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
	}, nil
}

func (s *NatsSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	sub, err := nats.NewSubscriber(s.config, s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
	}
	return sub.Subscribe(ctx, topic)
}

func (s *NatsSubscriber) Close() error {
	s.conn.Close()
	return nil
}
