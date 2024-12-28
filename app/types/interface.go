package types

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
)

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

func NewUUID() string {
	return watermill.NewUUID()
}

// Publisher interface for publishing messages
type Publisher interface {
	Publish(topic string, msg Message) error
}

type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan Message, error)
	Close() error
}

// LoggerAdapter interface for logging
type LoggerAdapter interface {
	Error(msg string, err error, fields LogFields)
	Info(msg string, fields LogFields)
	Debug(msg string, fields LogFields)
	Trace(msg string, fields LogFields)
	With(fields LogFields) LoggerAdapter
}

// LogFields represents a map of log fields.
type LogFields map[string]interface{}
