package adapters

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillSubscriberAdapter adapts a Watermill subscriber to the Subscriber interface.
type WatermillSubscriberAdapter struct {
	Subscriber message.Subscriber
}

// NewWatermillSubscriberAdapter creates a new adapter for a Watermill subscriber.
func NewWatermillSubscriberAdapter(sub message.Subscriber) types.Subscriber {
	return &WatermillSubscriberAdapter{Subscriber: sub}
}

// Subscribe subscribes to the given topic and returns a channel of messages.
func (w *WatermillSubscriberAdapter) Subscribe(ctx context.Context, topic string) (<-chan types.Message, error) {
	messages := make(chan types.Message) // Changed to chan types.Message
	wmMessages, err := w.Subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range wmMessages {
			adapter := NewWatermillMessageAdapter(msg.UUID, msg.Payload) // No need to use var and &
			messages <- adapter                                          // Send the adapter directly
		}
		close(messages)
	}()

	return messages, nil
}

// Close closes the subscriber.
func (w *WatermillSubscriberAdapter) Close() error {
	return w.Subscriber.Close()
}
