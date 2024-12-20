package stream_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

func TestStreamCreation(t *testing.T) {
	logger := watermill.NewStdLogger(true, true) // Enable debug logging
	natsURL := "nats://localhost:4222"           // Your NATS URL

	// Set up NATS options.
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.ErrorHandler(func(_ *nc.Conn, s *nc.Subscription, err error) {
			if s != nil {
				logger.Error("Error in subscription", err, watermill.LogFields{
					"subject": s.Subject,
					"queue":   s.Queue,
				})
			} else {
				logger.Error("Error in connection", err, nil)
			}
		}),
	}

	// Create a Watermill Publisher with AutoProvision enabled.
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         natsURL,
			NatsOptions: options,
			Marshaler:   &nats.NATSMarshaler{},
			JetStream: nats.JetStreamConfig{
				Disabled:      false,
				AutoProvision: true,
			},
		},
		logger,
	)
	if err != nil {
		t.Fatalf("Failed to create Watermill NATS publisher: %v", err)
	}
	defer publisher.Close()

	// Define a stream name.
	streamName := "teststream"

	// Publish a message to trigger stream creation.
	msg := message.NewMessage(watermill.NewUUID(), []byte("test message"))
	if err := publisher.Publish(streamName, msg); err != nil {
		t.Fatalf("Failed to publish message and auto-create stream: %v", err)
	}

	fmt.Printf("Stream '%s' should have been created (or ensured to exist)\n", streamName)
}
