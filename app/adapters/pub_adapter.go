// adapters/watermill_publisher_adapter.go
package adapters

import (
	"github.com/Black-And-White-Club/tcr-bot/app/types"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillPublisherAdapter adapts a Watermill publisher to the Publisher interface.
type WatermillPublisherAdapter struct {
	Publisher message.Publisher
}

// NewWatermillPublisherAdapter creates a new adapter for a Watermill publisher.
func NewWatermillPublisherAdapter(pub message.Publisher) types.Publisher {
	return &WatermillPublisherAdapter{Publisher: pub}
}

// Publish publishes messages to the given topic.
func (w *WatermillPublisherAdapter) Publish(topic string, msg types.Message) error {
	return w.Publisher.Publish(topic, msg.(*WatermillMessageAdapter).Message)
}
