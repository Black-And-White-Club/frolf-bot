package adapters

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type WatermillMessageAdapter struct {
	*message.Message
}

// NewWatermillMessageAdapter creates a new adapter for a Watermill message.
func NewWatermillMessageAdapter(uuid string, payload []byte) types.Message {
	if uuid == "" {
		uuid = watermill.NewUUID()
	}
	return &WatermillMessageAdapter{
		&message.Message{
			UUID:     uuid,
			Payload:  payload,
			Metadata: make(map[string]string), // Initialize Metadata here
		},
	}
}

func (w *WatermillMessageAdapter) Ack() {
	w.Message.Ack()
}

func (w *WatermillMessageAdapter) Nack() {
	w.Message.Nack()
}

func (w *WatermillMessageAdapter) Context() context.Context {
	return w.Message.Context()
}

func (w *WatermillMessageAdapter) SetContext(ctx context.Context) {
	w.Message.SetContext(ctx)
}

// UUID returns the UUID of the message.
func (w *WatermillMessageAdapter) UUID() string {
	return w.Message.UUID
}

// Payload returns the payload of the message.
func (w *WatermillMessageAdapter) Payload() []byte {
	return w.Message.Payload
}

// Metadata returns the metadata of the message.
func (w *WatermillMessageAdapter) Metadata() map[string]string {
	return w.Message.Metadata
}

// SetMetadata sets a metadata key-value pair in the message.
func (w *WatermillMessageAdapter) SetMetadata(key, value string) {
	w.Message.Metadata[key] = value
}
