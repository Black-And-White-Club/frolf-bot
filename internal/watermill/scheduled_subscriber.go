package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	nc "github.com/nats-io/nats.go"
)

// NewScheduledTaskSubscriber creates a new NATS JetStream context
// specifically for scheduled tasks.
func NewScheduledTaskSubscriber(natsURL string, logger watermill.LoggerAdapter) (nc.JetStreamContext, error) {
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context here
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return js, nil // Return the JetStream context directly
}
