package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// NatsPublisher implements the Watermill Publisher interface.
type NatsPublisher struct {
	conn   *nc.Conn
	config nats.PublisherConfig
	logger watermill.LoggerAdapter
}

// NewPublisher creates a new NATS JetStream publisher.
func NewPublisher(natsURL string, logger watermill.LoggerAdapter) (*NatsPublisher, error) {
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	jsConfig := nats.JetStreamConfig{
		Disabled:      false,
		AutoProvision: true,
		// PublishOptions is no longer available, so it's removed
	}

	return &NatsPublisher{
		conn: conn,
		config: nats.PublisherConfig{
			Marshaler:         &nats.GobMarshaler{},
			JetStream:         jsConfig,
			SubjectCalculator: nats.DefaultSubjectCalculator,
		},
		logger: logger,
	}, nil
}

func (p *NatsPublisher) Publish(topic string, messages ...*message.Message) error {
	pub, err := nats.NewPublisher(p.config, p.logger)
	if err != nil {
		return fmt.Errorf("failed to create NATS publisher: %w", err)
	}
	return pub.Publish(topic, messages...)
}

func (p *NatsPublisher) Close() error {
	p.conn.Close()
	return nil
}
