package usersubscribers

import (
	"context"

	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
)

// UserEventSubscriber defines the interface for subscribing to user events.
type UserEventSubscriber interface {
	SubscribeToUserEvents(ctx context.Context, subscriber types.Subscriber, handlers user.Handlers, logger types.LoggerAdapter) error
}
