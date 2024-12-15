package app

import (
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/internal/handlers" // Assuming your handlers are in this package
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
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

	// Get the NATS URL from the PubSuber
	natsConn := pubsub.(*watermillutil.PubSub).NatsConn() // Get the NATS connection using the new method
	natsURL := natsConn.Opts.Url

	scheduledTaskSubscriber, err := watermillutil.NewScheduledTaskSubscriber(
		natsURL,
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task subscriber: %w", err)
	}

	// Add a route for the scheduled tasks
	router.AddNoPublisherHandler(
		"ScheduledTaskHandler",
		"scheduled_tasks", // Your scheduled tasks stream name
		scheduledTaskSubscriber,
		handlers.ScheduledTaskHandler, // Assuming you have a ScheduledTaskHandler in your handlers package
	)

	return nil
}
