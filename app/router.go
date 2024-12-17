package app

import (
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/types"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module defines the interface for application modules.
type Module interface {
	RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error
	GetHandlers() map[string]types.Handler
}

// RegisterHandlers registers all the event handlers for the application modules.
func RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber, modules ...Module) error {
	for _, module := range modules {
		log.Printf("Registering handlers for module: %T", module) // Log module type

		if err := module.RegisterHandlers(router, pubsub); err != nil {
			log.Printf("Failed to register handlers for module %T: %v", module, err) // More specific error logging
			return fmt.Errorf("failed to register module handlers: %w", err)
		}

		// Log the registered handlers for each module
		for handlerName, handler := range module.GetHandlers() {
			log.Printf("Registered handler: %s, Topic: %s, PubSub: %p", handlerName, handler.Topic, pubsub) // Log pubsub address
		}
	}

	return nil
}
