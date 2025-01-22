package eventbus

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"
)

// InitializeStreams creates the necessary streams in JetStream during application startup.
func InitializeStreams(js jetstream.JetStream, logger *slog.Logger) error {
	streamConfigs := []jetstream.StreamConfig{
		{
			Name:     "user",        // Use a single stream name
			Subjects: []string{"*"}, // Subscribe to all subjects on this stream
		},
	}

	for _, streamConfig := range streamConfigs {
		_, err := js.Stream(context.Background(), streamConfig.Name)
		if err == jetstream.ErrStreamNotFound {
			_, err := js.CreateStream(context.Background(), streamConfig)
			if err != nil {
				logger.Error("Failed to create JetStream stream", slog.String("stream", streamConfig.Name), slog.Any("error", err))
				return err
			}
			logger.Info("Created JetStream stream", slog.String("stream", streamConfig.Name))
		} else if err != nil {
			return fmt.Errorf("failed to check stream: %w", err)
		}
	}
	return nil
}
