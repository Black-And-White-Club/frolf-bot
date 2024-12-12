package app

import (
	"fmt"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module defines the interface for application modules.
type Module interface {
	RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error
}

// RegisterHandlers registers all the event handlers for the application modules.
func RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber, modules ...Module) error {
	for _, module := range modules {
		if err := module.RegisterHandlers(router, pubsub); err != nil {
			return fmt.Errorf("failed to register module handlers: %w", err)
		}
	}

	return nil
}
