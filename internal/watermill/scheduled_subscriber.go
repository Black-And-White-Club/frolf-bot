package watermillutil

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	nc "github.com/nats-io/nats.go"
)

// NewScheduledTaskSubscriber creates a new NATS JetStream context
// specifically for scheduled tasks.
func NewScheduledTaskSubscriber(natsURL string, logger watermill.LoggerAdapter) (nc.JetStreamContext, error) {
	logger.Info("Connecting to NATS for scheduled tasks", nil)
	conn, err := nc.Connect(natsURL, nc.Name("App Service"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Info("Connected to NATS for scheduled tasks", nil)

	logger.Info("Creating JetStream context for scheduled tasks", nil)
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}
	logger.Info("Created JetStream context for scheduled tasks", nil)

	return js, nil
}
