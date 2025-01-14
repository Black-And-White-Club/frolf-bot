package shared

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type EventBus interface {
	// Publish publishes a Watermill message to the specified stream.
	// The message metadata should contain the "subject" key with the event type string.
	Publish(ctx context.Context, streamName string, msg *message.Message) error

	// Subscribe subscribes a handler function to a specific stream and subject.
	Subscribe(ctx context.Context, streamName string, subject string, handler func(ctx context.Context, msg *message.Message) error) error

	// CreateStream creates a stream with the given name.
	CreateStream(ctx context.Context, streamName string) error

	// Close closes the underlying resources held by the EventBus implementation.
	Close() error
}
