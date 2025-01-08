package adapters

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillSubscriberAdapter adapts a Watermill subscriber to the Subscriber interface.
type WatermillSubscriberAdapter struct {
	Subscriber message.Subscriber
	logger     shared.LoggerAdapter
}

// NewWatermillSubscriberAdapter creates a new adapter for a Watermill subscriber.
func NewWatermillSubscriberAdapter(sub message.Subscriber, logger shared.LoggerAdapter) shared.EventBus { // Add logger parameter
	return &WatermillSubscriberAdapter{
		Subscriber: sub,
		logger:     logger,
	}
}

// Subscribe subscribes to the given topic and registers a handler
// function for received messages.
func (w *WatermillSubscriberAdapter) Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg shared.Message) error) error {
	wmMessages, err := w.Subscriber.Subscribe(ctx, topic)
	if err != nil {
		return err
	}

	go func() {
		for msg := range wmMessages {
			// Convert Watermill message to shared.Message
			sharedMsg := NewWatermillMessageAdapter(msg.UUID, msg.Payload)
			for key, value := range msg.Metadata {
				sharedMsg.SetMetadata(key, value)
			}

			if err := handler(ctx, sharedMsg); err != nil {
				w.logger.Error("Handler error", err, shared.LogFields{
					"message_uuid": sharedMsg.UUID(), // Access UUID from sharedMsg
					"topic":        topic,
				})
				sharedMsg.Nack()
				continue
			}
		}
	}()

	return nil
}

// Close closes the subscriber.
func (w *WatermillSubscriberAdapter) Close() error {
	return w.Subscriber.Close()
}

// These methods are required to implement shared.EventBus, but they are not used in this adapter.
func (w *WatermillSubscriberAdapter) Publish(ctx context.Context, eventType shared.EventType, msg shared.Message) error {
	return nil
}
func (w *WatermillSubscriberAdapter) RegisterNotFoundHandler(handler func(eventType shared.EventType) error) {
}

func (w *WatermillSubscriberAdapter) PublishWithMetadata(ctx context.Context, eventType shared.EventType, msg shared.Message, metadata map[string]string) error {
	return nil
}
