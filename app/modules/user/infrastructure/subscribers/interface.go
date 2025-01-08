package usersubscribers

import (
	"context"

	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// UserEventSubscriber defines the interface for subscribing to user events.
type UserEventSubscriber interface {
	SubscribeToUserEvents(ctx context.Context, eventBus shared.EventBus, handlers user.Handlers, logger shared.LoggerAdapter) error
}
