package watermillutil

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SendCommand is a helper function to publish commands.
func SendCommand(ctx context.Context, publisher message.Publisher, marshaler cqrs.CommandEventMarshaler, cmd interface{}, commandName string) error {
	msg, err := marshaler.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := publisher.Publish(commandName, msg); err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	return nil
}
