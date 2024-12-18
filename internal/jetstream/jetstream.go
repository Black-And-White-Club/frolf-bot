package jetstream

import (
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
)

// CreateStream creates a new JetStream stream.
func CreateStream(js nats.JetStreamContext, streamName string) error {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{fmt.Sprintf("%s.>", streamName)},
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return fmt.Errorf("failed to add stream: %w", err)
	}
	return nil
}
