package shared

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
)

// Message defines the interface for an event message.
type Message interface {
	Ack()
	Nack()
	Context() context.Context
	SetContext(ctx context.Context)
	UUID() string
	Payload() []byte
	Metadata() map[string]string
	SetMetadata(key, value string)
}

// NewUUID generates a new unique identifier for a message.
func NewUUID() string {
	return watermill.NewUUID()
}
