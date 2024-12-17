package app

import (
	"fmt"
	"log"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module defines the interface for application modules.
type Module interface {
	RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error
}

// RegisterHandlers registers all the event handlers for the application modules.
func RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber, modules ...Module) error {
	if router == nil {
		log.Println("Error: router is nil")
		return fmt.Errorf("router is nil")
	}
	if pubsub == nil {
		log.Println("Error: pubsub is nil")
		return fmt.Errorf("pubsub is nil")
	}
	for _, module := range modules {
		log.Printf("Registering handlers for module: %T", module)

		if err := module.RegisterHandlers(router, pubsub); err != nil {
			return fmt.Errorf("failed to register handlers for module %T: %w", module, err)
		}

	}

	return nil
}
