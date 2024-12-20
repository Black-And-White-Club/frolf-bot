package usersubscribers

import (
	"context"

	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserEventSubscriber defines the interface for subscribing to user events.
type UserSubscribers interface {
	SubscribeToUserEvents(
		ctx context.Context,
		subscriber message.Subscriber,
		handlers userhandlers.Handlers,
		logger watermill.LoggerAdapter,
	) error
}
